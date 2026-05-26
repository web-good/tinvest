package golden_x

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"
	_ "time/tzdata"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/service/trading_strategy/golden_x/factory"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/service/trading_strategy/golden_x/percentile"
	"tinvest/internal/utils"
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

// ErrAdaptiveInsufficientHistory is returned when a share has fewer than
// settings.AdaptiveWindowMin weekly RSI values available.
var ErrAdaptiveInsufficientHistory = errors.New("adaptive tiers: insufficient RSI history")

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
	buyInfo := domain.NewInfo()
	sellInfo := domain.NewInfo()
	trends := make(map[string]gxmodel.TrendStatus)
	thresholds := make(map[string]gxmodel.Thresholds)
	sellThresholds := make(map[string]gxmodel.SellThresholds)
	divergences := make(map[string]bool)
	volumesConfirmed := make(map[string]bool)
	settings := factory.DefaultSettings()

	for _, share := range in.ShareList.All() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		candles, candleErr := s.fetchWeeklyCandles(ctx, share.ID, in.Interval, dateNow)
		if candleErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("get candles for %s: %w", share.Name, candleErr).Error())
			continue
		}
		weekly := compactCandles(candles)

		sig, detectErr := Detect(weekly, share.RSILength, in.Kind, in.UseTrendFilter, settings)
		if errors.Is(detectErr, ErrAdaptiveInsufficientHistory) {
			logger.InfoContext(ctx, "adaptive tiers: insufficient history", "share", share.Name)
			continue
		}
		if errors.Is(detectErr, ErrInsufficientHistory) {
			logger.InfoContext(ctx, "trend filter: insufficient history", "share", share.Name)
			continue
		}
		if detectErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("detect signal for %s: %w", share.Name, detectErr).Error())
			continue
		}

		thresholds[share.ID] = sig.Thresholds
		sellThresholds[share.ID] = sig.SellThresholds
		if in.UseTrendFilter {
			trends[share.ID] = sig.TrendStatus
		}

		if sig.GreenBuy || sig.YellowBuy {
			divergences[share.ID] = sig.DivergenceOK
			volumesConfirmed[share.ID] = sig.VolumeOK
		}

		buyTier := percentile.TierFromAdaptive(sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
		sellTier := percentile.SellTierFromAdaptive(sig.RSI, sig.SellThresholds, in.Kind)
		item := domain.Item{
			InstrumentName: share.Name,
			RSIValue:       sig.RSI,
		}
		// Buy and sell zones are mutually exclusive — RSI can't be both < p15 and > p80.
		switch {
		case buyTier != percentile.TierNone:
			buyInfo.WriteToMap(share.ID, item)
		case sellTier != percentile.TierNone:
			sellInfo.WriteToMap(share.ID, item)
		}
	}

	if len(buyInfo.Items()) > 0 || len(sellInfo.Items()) > 0 {
		msg := notif.Trade(buyInfo, sellInfo, in.Kind, trends, thresholds, sellThresholds, divergences, volumesConfirmed)
		if sendErr := s.tgClient.SendMessage(msg); sendErr != nil {
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
