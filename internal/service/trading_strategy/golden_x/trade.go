package golden_x

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
	_ "time/tzdata"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/classifier"
	"tinvest/internal/service/trading_strategy/golden_x/detector"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// candleLookbackWeeks is how many weekly candles we request per share per tick.
// Covers EMA200 warmup (Growth-only) and adaptive-tier RSI history (both
// instances) in one RPC, staying under the Tinkoff weekly cap (300 per call).
// Out of scope for D2: this is a fetch-policy knob, not an algorithm knob.
const candleLookbackWeeks = 260

// fetchConcurrencyLimit caps parallel candle RPCs so we don't overwhelm the
// Tinkoff API gateway or exhaust local file descriptors.
const fetchConcurrencyLimit = 5

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
	signals := detector.DetectAll(ctx, fetched, in, s.settings, s.rankProvider.RankBonus)
	result := classifier.Classify(signals, in)
	result = classifier.CapSectors(result)

	return s.notify(ctx, result)
}

// fetchAll concurrently fetches and compacts weekly candles for every share.
// Individual fetch errors are captured per-share; the caller decides how to
// handle them. The errgroup concurrency is capped at fetchConcurrencyLimit.
func (s *service) fetchAll(ctx context.Context, in dto.Trade, dateNow time.Time) []gxmodel.FetchResult {
	allShares := in.ShareList.All()
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(fetchConcurrencyLimit)

	var mu sync.Mutex
	results := make([]gxmodel.FetchResult, 0, len(allShares))

	for _, share := range allShares {
		g.Go(func() error {
			candles, err := s.fetchWeeklyCandles(gCtx, share.ID, in.Interval, dateNow)
			fr := gxmodel.FetchResult{Share: share, Err: err}
			if err == nil {
				fr.Candles = compactCandles(candles)
			}
			mu.Lock()
			results = append(results, fr)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return results
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
		interval.ToNumberInvestAPI(),
		utils.TimeStampPbGenerator(dateNow, -int64(candleLookbackWeeks), interval),
		timestamppb.New(dateNow),
		&limit,
		true,
	)
}
