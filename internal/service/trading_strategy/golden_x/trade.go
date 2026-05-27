package golden_x

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"sync"
	"time"
	_ "time/tzdata"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/service/trading_strategy/golden_x/percentile"
	"tinvest/internal/utils"
	"tinvest/pkg/collection"
	"tinvest/pkg/logger"
)

// candleLookbackWeeks is how many weekly candles we request per share per tick.
// Covers EMA200 warmup (Growth-only) and adaptive-tier RSI history (both
// instances) in one RPC, staying under the Tinkoff weekly cap (300 per call).
// Out of scope for D2: this is a fetch-policy knob, not an algorithm knob.
const candleLookbackWeeks = 260

// divergenceFractalK is the half-window for fractal pivot detection: a candle
// at index i is a confirmed pivot low iff its Low is strictly less than the
// Lows of its 2*k neighbors (k on each side). k=2 is the classical Williams
// fractal setting; on a weekly TF it gives a swing-low at least 2 weeks old.
// Out of scope for D2: indicator-internal pivot width, not on the knob list.
const divergenceFractalK = 2

// fetchConcurrencyLimit caps parallel candle RPCs so we don't overwhelm the
// Tinkoff API gateway or exhaust local file descriptors.
const fetchConcurrencyLimit = 5

// ErrAdaptiveInsufficientHistory is returned when a share has fewer than
// settings.AdaptiveWindowMin weekly RSI values available.
var ErrAdaptiveInsufficientHistory = errors.New("adaptive tiers: insufficient RSI history")

// fetchResult holds the outcome of a single share's candle fetch.
type fetchResult struct {
	share   collection.Instrument
	candles []*model.CandleItemTechAnalyse
	err     error
}

// detectResult holds a successfully detected signal for a single share.
type detectResult struct {
	share  collection.Instrument
	signal gxmodel.Signal
}

func (s *service) Trade(ctx context.Context, in dto.Trade) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("panic in golden_x.Trade: %v\n%s", r, debug.Stack()))
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	loc, locErr := time.LoadLocation("Europe/Moscow")
	if locErr != nil {
		return fmt.Errorf("load timezone: %w", locErr)
	}
	dateNow := time.Now().In(loc)

	fetched := s.fetchAll(ctx, in, dateNow)
	signals := detectAll(ctx, fetched, in, s.settings)
	result := classify(signals, in)
	result = capSectors(result)
	return s.notify(ctx, result)
}

// fetchAll concurrently fetches and compacts weekly candles for every share.
// Individual fetch errors are captured per-share; the caller decides how to
// handle them. The errgroup concurrency is capped at fetchConcurrencyLimit.
func (s *service) fetchAll(ctx context.Context, in dto.Trade, dateNow time.Time) []fetchResult {
	allShares := in.ShareList.All()
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(fetchConcurrencyLimit)

	var mu sync.Mutex
	results := make([]fetchResult, 0, len(allShares))

	for _, share := range allShares {
		g.Go(func() error {
			candles, err := s.fetchWeeklyCandles(gCtx, share.ID, in.Interval, dateNow)
			fr := fetchResult{share: share, err: err}
			if err == nil {
				fr.candles = compactCandles(candles)
			}
			mu.Lock()
			results = append(results, fr)
			mu.Unlock()
			return nil // errors are tracked per-share, never cancel the group
		})
	}
	_ = g.Wait()
	return results
}

// detectAll runs Detect on every successfully fetched share. Shares with
// fetch errors or insufficient history are logged and skipped.
func detectAll(ctx context.Context, fetched []fetchResult, in dto.Trade, settings gxmodel.Settings) []detectResult {
	results := make([]detectResult, 0, len(fetched))
	for _, fr := range fetched {
		if fr.err != nil {
			logger.ErrorContext(ctx, fmt.Errorf("get candles for %s: %w", fr.share.Name, fr.err).Error())
			continue
		}

		sig, detectErr := Detect(fr.candles, fr.share.RSILength, in.Kind, in.UseTrendFilter, settings)
		if errors.Is(detectErr, ErrAdaptiveInsufficientHistory) {
			logger.InfoContext(ctx, "adaptive tiers: insufficient history", "share", fr.share.Name)
			continue
		}
		if errors.Is(detectErr, ErrInsufficientHistory) {
			logger.InfoContext(ctx, "trend filter: insufficient history", "share", fr.share.Name)
			continue
		}
		if detectErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("detect signal for %s: %w", fr.share.Name, detectErr).Error())
			continue
		}

		results = append(results, detectResult{share: fr.share, signal: sig})
	}
	return results
}

// classify buckets detected signals into buy and sell maps based on adaptive
// percentile tiers. It is pure: no I/O, no context.
func classify(detected []detectResult, in dto.Trade) gxmodel.TradeResult {
	result := gxmodel.TradeResult{
		BuyShares:       make(map[string]gxmodel.ShareResult),
		SellShares:      make(map[string]gxmodel.ShareResult),
		CappedBuyShares: make(map[string]gxmodel.ShareResult),
		Kind:            in.Kind,
	}
	for _, dr := range detected {
		sig := dr.signal
		buyTier := percentile.TierFromAdaptive(sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
		sellTier := percentile.SellTierFromAdaptive(sig.RSI, sig.SellThresholds, in.Kind)

		sr := gxmodel.ShareResult{
			InstrumentName: dr.share.Name,
			RSI:            sig.RSI,
			BuyTier:        buyTier,
			SellTier:       sellTier,
			Thresholds:     sig.Thresholds,
			SellThresholds: sig.SellThresholds,
			Sector:         dr.share.Sector,
		}
		if in.UseTrendFilter {
			sr.TrendStatus = sig.TrendStatus
		}
		if sig.GreenBuy || sig.YellowBuy {
			sr.DivergenceOK = sig.DivergenceOK
			sr.VolumeOK = sig.VolumeOK
		}

		sr.Score = signalScore(sr)

		// Buy and sell zones are mutually exclusive — RSI can't be both < p15 and > p80.
		switch {
		case buyTier != gxmodel.TierNone:
			result.BuyShares[dr.share.ID] = sr
		case sellTier != gxmodel.TierNone:
			result.SellShares[dr.share.ID] = sr
		}
	}
	return result
}

// sectorCapPerTick is the maximum number of buy signals allowed per sector in a
// single Trade tick. Excess signals are moved to CappedBuyShares.
const sectorCapPerTick = 2

// capSectors enforces sector concentration limits: at most sectorCapPerTick
// buy signals per sector pass through; the rest move to CappedBuyShares.
// Shares with empty Sector are never capped. SellShares are not affected.
func capSectors(result gxmodel.TradeResult) gxmodel.TradeResult {
	// Group buy-share IDs by sector.
	sectorIDs := make(map[string][]string)
	for id, sr := range result.BuyShares {
		if sr.Sector == "" {
			continue // empty sector is never capped
		}
		sectorIDs[sr.Sector] = append(sectorIDs[sr.Sector], id)
	}

	for sector, ids := range sectorIDs {
		if len(ids) <= sectorCapPerTick {
			continue
		}

		// Sort by Score descending; use share ID as stable tiebreaker.
		sort.Slice(ids, func(i, j int) bool {
			si := result.BuyShares[ids[i]].Score
			sj := result.BuyShares[ids[j]].Score
			if si != sj {
				return si > sj
			}
			return ids[i] < ids[j]
		})

		_ = sector // used only as the map key above

		// Keep top-N, move the rest to CappedBuyShares.
		for _, id := range ids[sectorCapPerTick:] {
			result.CappedBuyShares[id] = result.BuyShares[id]
			delete(result.BuyShares, id)
		}
	}

	return result
}

// signalScore computes a composite buy-signal strength score for a ShareResult.
// Higher scores indicate stronger confluence of buy conditions.
//
//   - TierGreen buy tier:  +3
//   - TierYellow buy tier: +1
//   - TrendWith:           +2
//   - DivergenceOK:        +2
//   - VolumeOK:            +1
func signalScore(sr gxmodel.ShareResult) int {
	s := 0
	switch sr.BuyTier {
	case gxmodel.TierGreen:
		s += 3
	case gxmodel.TierYellow:
		s += 1
	}
	if sr.TrendStatus == gxmodel.TrendWith {
		s += 2
	}
	if sr.DivergenceOK {
		s += 2
	}
	if sr.VolumeOK {
		s += 1
	}
	return s
}

// notify sends the aggregated trade result to Telegram. If there are no
// buy or sell signals it is a no-op.
func (s *service) notify(ctx context.Context, result gxmodel.TradeResult) error {
	if len(result.BuyShares) == 0 && len(result.SellShares) == 0 {
		return nil
	}
	msg := notif.Trade(result)
	if sendErr := s.tgClient.SendMessage(msg); sendErr != nil {
		logger.ErrorContext(ctx, "message is not sent", sendErr)
		return sendErr
	}
	return nil
}

// fetchWeeklyCandles pulls up to ~candleLookbackWeeks weekly candles for the
// share — enough for both EMA200 (Growth-only) and adaptive RSI percentiles
// (both instances) in a single RPC.
func (s *service) fetchWeeklyCandles(ctx context.Context, shareID string, interval enum.Interval, dateNow time.Time) ([]*model.CandleItemTechAnalyse, error) {
	limit := int32(candleLookbackWeeks + 20)
	return s.marketDataServiceGrpcClient.GetCandles(
		ctx,
		&shareID,
		interval.ToNumberInvestApi(),
		utils.TimeStampPbGenerator(dateNow, -int64(candleLookbackWeeks), interval),
		timestamppb.New(dateNow),
		&limit,
		true,
	)
}
