package golden_x

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// trendCandleLookbackWeeks is how many weekly candles back we request when the
// trend filter is enabled. trendEMAPeriod (200) is the bare minimum to compute
// a single EMA point; we add headroom for warmup stability while staying under
// the Tinkoff weekly candle cap (300 per call) and within the 5-year window.
const trendCandleLookbackWeeks = 260

func (s *service) Trade(ctx context.Context, in dto.Trade) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("panic in golden_x.Trade: %v", r))
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	info := domain.NewInfo()
	RSIInfo := domain.NewInfo()
	trends := make(map[string]dto.TrendStatus)

	for _, share := range in.ShareList.All() {
		rsi, rsiErr := s.rsi.CalculateRSI(
			ctx,
			share.ID,
			in.Interval,
			utils.TimeStampPbGenerator(dateNow, -80, in.Interval),
			timestamppb.New(dateNow),
			int32(share.RSILength),
		)

		if rsiErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate RSI :%w", rsiErr).Error())

			continue
		}

		closedRSI, ok := lastClosedWeeklyRSI(rsi, dateNow, loc)
		if !ok {
			logger.InfoContext(ctx, "no closed weekly RSI candle for share", "share", share.Name)
			continue
		}
		rsiValue := utils.CombinePrice(closedRSI.SignalLine.Units, closedRSI.SignalLine.Nano)

		if in.UseTrendFilter {
			status, trendErr := s.trendStatus(ctx, share.ID, in.Interval, dateNow, loc)
			if errors.Is(trendErr, ErrInsufficientHistory) {
				logger.InfoContext(ctx, "trend filter: insufficient history", "share", share.Name)
				continue
			}
			if trendErr != nil {
				logger.ErrorContext(ctx, fmt.Errorf("trend filter: %w", trendErr).Error(), "share", share.Name)
				continue
			}
			trends[share.ID] = status
		}

		RSIInfo.WriteToMap(
			share.ID,
			domain.Item{
				InstrumentName: share.Name,
				RSILength:      share.RSILength,
				RSIValue:       rsiValue,
			})
		tier := tierFromRSI(rsiValue)
		if !s.state.ShouldAlert(share.ID, tier) {
			continue
		}

		info.WriteToMap(
			share.ID,
			domain.Item{
				InstrumentName: share.Name,
				RSIValue:       rsiValue,
			})
	}

	if len(info.Items()) > 0 {
		err := s.tgClient.SendMessage(notif.Trade(info, in.Kind, trends))
		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	if len(RSIInfo.Items()) > 0 {
		err := s.tgClient.SendMessage(notif.RSIList(RSIInfo, in.Kind))
		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}

// trendStatus loads weekly candles for the share and returns its EMA200-W trend
// status. Returns ErrInsufficientHistory when the share lacks enough closed
// weekly candles for the configured period.
func (s *service) trendStatus(ctx context.Context, shareID string, interval enum.Interval, dateNow time.Time, loc *time.Location) (dto.TrendStatus, error) {
	limit := int32(trendCandleLookbackWeeks + 20)
	candles, err := s.marketDataServiceGrpcClient.GetCandles(
		ctx,
		&shareID,
		interval.ToNumberInvestApi(),
		utils.TimeStampPbGenerator(dateNow, -int64(trendCandleLookbackWeeks), interval),
		timestamppb.New(dateNow),
		&limit,
		true,
	)
	if err != nil {
		return dto.TrendUnknown, fmt.Errorf("get candles: %w", err)
	}
	return trendStatusFromCandles(candles, trendEMAPeriod, dateNow, loc)
}
