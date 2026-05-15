package golden_x

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	notif "tinvest/internal/service/trading_strategy/golden_x/notification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

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
		err := s.tgClient.SendMessage(notif.Trade(info, in.Kind))
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
