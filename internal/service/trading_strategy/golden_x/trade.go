package golden_x

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// adaptiveWindowMax is the maximum number of historical RSI values we keep for
// percentile computation. Matches the original ~200 from the design spec.
const adaptiveWindowMax = 200

// adaptiveWindowMin is the lower bound; shares with fewer closed-week RSI
// values than this are skipped (consistent with C1's insufficient-history rule).
const adaptiveWindowMin = 100

// candleLookbackWeeks is how many weekly candles we request per share per tick.
// Covers EMA200 warmup (Growth-only) and adaptive-tier RSI history (both
// instances) in one RPC, staying under the Tinkoff weekly cap (300 per call).
const candleLookbackWeeks = 260

// divergenceFractalK is the half-window for fractal pivot detection: a candle
// at index i is a confirmed pivot low iff its Low is strictly less than the
// Lows of its 2*k neighbors (k on each side). k=2 is the classical Williams
// fractal setting; on a weekly TF it gives a swing-low at least 2 weeks old.
const divergenceFractalK = 2

// divergenceLookbackWeeks bounds how far back we search for the prior pivot
// low. Older pivots are ignored — a year-old swing low is "stale" evidence
// for current week behavior on a weekly TF.
const divergenceLookbackWeeks = 52

// ErrAdaptiveInsufficientHistory is returned when a share has fewer than
// adaptiveWindowMin closed weekly RSI values available.
var ErrAdaptiveInsufficientHistory = errors.New("adaptive tiers: insufficient RSI history")

func (s *service) Trade(ctx context.Context, in dto.Trade) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("panic in golden_x.Trade: %v", r))
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	buyInfo := domain.NewInfo()
	sellInfo := domain.NewInfo()
	RSIInfo := domain.NewInfo()
	trends := make(map[string]dto.TrendStatus)
	thresholds := make(map[string]dto.Thresholds)
	sellThresholds := make(map[string]dto.SellThresholds)
	divergences := make(map[string]bool)

	for _, share := range in.ShareList.All() {
		candles, candleErr := s.fetchWeeklyCandles(ctx, share.ID, in.Interval, dateNow)
		if candleErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("get candles for %s: %w", share.Name, candleErr).Error())
			continue
		}
		closed := closedWeeklyCandles(candles, dateNow, loc)

		lastRSI, rsiSeries, sharedThresholds, adaptiveErr := adaptiveRSIForShare(closed, share.RSILength)
		if errors.Is(adaptiveErr, ErrAdaptiveInsufficientHistory) {
			logger.InfoContext(ctx, "adaptive tiers: insufficient history", "share", share.Name)
			continue
		}
		if adaptiveErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("adaptive tiers for %s: %w", share.Name, adaptiveErr).Error())
			continue
		}

		if in.UseTrendFilter {
			status, trendErr := trendStatusFromCandles(candles, trendEMAPeriod, dateNow, loc)
			if errors.Is(trendErr, ErrInsufficientHistory) {
				logger.InfoContext(ctx, "trend filter: insufficient history", "share", share.Name)
				continue
			}
			if trendErr != nil {
				logger.ErrorContext(ctx, fmt.Errorf("trend filter for %s: %w", share.Name, trendErr).Error())
				continue
			}
			trends[share.ID] = status
		}

		thresholds[share.ID] = sharedThresholds
		shareSellThresholds := adaptiveSellThresholds(rsiSeries)
		sellThresholds[share.ID] = shareSellThresholds

		buyTier := tierFromAdaptive(lastRSI, sharedThresholds.P5, sharedThresholds.P15)

		if buyTier != tierNone {
			lows := lowsAlignedToRSI(closed, share.RSILength, rsiSeries)
			if len(lows) > divergenceLookbackWeeks {
				lows = lows[len(lows)-divergenceLookbackWeeks:]
			}
			rsiTail := rsiSeries
			if len(rsiTail) > divergenceLookbackWeeks {
				rsiTail = rsiTail[len(rsiTail)-divergenceLookbackWeeks:]
			}
			if bullishDivergence(lows, rsiTail, divergenceFractalK) {
				divergences[share.ID] = true
			}
		}

		RSIInfo.WriteToMap(
			share.ID,
			domain.Item{
				InstrumentName: share.Name,
				RSILength:      share.RSILength,
				RSIValue:       lastRSI,
			})

		sellTier := sellTierFromAdaptive(lastRSI, shareSellThresholds, in.Kind)

		// Buy and sell zones are mutually exclusive on RSI — picks whichever
		// (if any) is non-None.
		finalTier := buyTier
		if sellTier != tierNone {
			finalTier = sellTier
		}

		if !s.state.ShouldAlert(share.ID, finalTier) {
			continue
		}

		item := domain.Item{
			InstrumentName: share.Name,
			RSIValue:       lastRSI,
		}
		switch finalTier {
		case tierYellow, tierGreen:
			buyInfo.WriteToMap(share.ID, item)
		case tierSellYellow, tierSellOrange, tierSellRed:
			sellInfo.WriteToMap(share.ID, item)
		}
	}

	if len(buyInfo.Items()) > 0 || len(sellInfo.Items()) > 0 {
		msg := notif.Trade(buyInfo, sellInfo, in.Kind, trends, thresholds, sellThresholds, divergences)
		if sendErr := s.tgClient.SendMessage(msg); sendErr != nil {
			logger.ErrorContext(ctx, "message is not sent", sendErr)
			return sendErr
		}
	}

	if len(RSIInfo.Items()) > 0 {
		if sendErr := s.tgClient.SendMessage(notif.RSIList(RSIInfo, in.Kind, thresholds)); sendErr != nil {
			logger.ErrorContext(ctx, "message is not sent", sendErr)
			return sendErr
		}
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

// adaptiveRSIForShare computes the share's last-closed-week RSI, the trimmed
// historical RSI slice used for percentiles, and the adaptive p5/p15 buy
// thresholds. Returns ErrAdaptiveInsufficientHistory if fewer than
// adaptiveWindowMin RSI values are available. The returned slice may be used
// by the caller to derive additional percentiles (e.g. sell thresholds)
// without recomputing RSI.
func adaptiveRSIForShare(closedCandles []*model.CandleItemTechAnalyse, rsiPeriod int) (float64, []float64, dto.Thresholds, error) {
	closes := make([]float64, len(closedCandles))
	for i, c := range closedCandles {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	full := computeRSISeries(closes, rsiPeriod)
	if len(full) <= rsiPeriod {
		return 0, nil, dto.Thresholds{}, ErrAdaptiveInsufficientHistory
	}
	rsi := full[rsiPeriod:]
	if len(rsi) < adaptiveWindowMin {
		return 0, nil, dto.Thresholds{}, ErrAdaptiveInsufficientHistory
	}
	if len(rsi) > adaptiveWindowMax {
		rsi = rsi[len(rsi)-adaptiveWindowMax:]
	}
	lastRSI := rsi[len(rsi)-1]
	return lastRSI, rsi, adaptiveThresholds(rsi), nil
}

// lowsAlignedToRSI extracts Low values from closedCandles and trims the head
// so the returned slice aligns 1-to-1 with the rsiSeries returned by
// adaptiveRSIForShare. The result has the same length as rsiSeries (or
// shorter, if there are fewer candles available — defensive).
func lowsAlignedToRSI(closedCandles []*model.CandleItemTechAnalyse, rsiPeriod int, rsiSeries []float64) []float64 {
	// adaptiveRSIForShare drops rsiPeriod warmup candles, then potentially
	// keeps only the last adaptiveWindowMax. Mirror the same trim here.
	start := rsiPeriod
	if len(closedCandles)-rsiPeriod > len(rsiSeries) {
		start = len(closedCandles) - len(rsiSeries)
	}
	if start < 0 {
		start = 0
	}
	out := make([]float64, 0, len(closedCandles)-start)
	for i := start; i < len(closedCandles); i++ {
		out = append(out, utils.CombinePrice(closedCandles[i].Low.Units, closedCandles[i].Low.Nano))
	}
	return out
}
