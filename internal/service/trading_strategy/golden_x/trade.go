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
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	var shareRSI dto.Share
	info := domain.NewInfo()
	limit := int32(3)

	for _, share := range t {
		flag := false
		for _, DTOShare := range in.ShareList {
			if DTOShare.ID == share.ID {
				flag = true
				shareRSI = DTOShare
			}
		}

		if !flag {
			continue
		}

		rsi, rsiErr := s.rsi.CalculateRSI(
			ctx,
			share.ID,
			in.Interval,
			utils.TimeStampPbGenerator(dateNow, -20, in.Interval),
			timestamppb.New(dateNow),
			int32(shareRSI.RSILength),
		)

		if rsiErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate RSI :%w", rsiErr).Error())

			continue
		}

		if utils.CombinePrice(rsi[0].SignalLine.Units, rsi[0].SignalLine.Nano) > 30 {
			continue
		}

		candles, candlesErr := s.marketDataServiceGrpcClient.GetCandles(ctx, &share.ID, in.Interval.ToNumberInvestApi(), utils.TimeStampPbGenerator(dateNow, -20, in.Interval), timestamppb.New(dateNow), &limit, true)
		if candlesErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate Candle :%w", candlesErr).Error())
		}

		procent := calculateProcentToDevident(utils.CombinePrice(candles[len(candles)-1].Close.Units, candles[len(candles)-1].Close.Nano), shareRSI.AverageDevident)
		info.WriteToMap(share.ID, domain.Item{InstrumentName: share.Name, ProcentPrice: procent, RSIValue: shareRSI.RSILength})

		if len(info.Items()) > 0 {
			err := s.tgClient.SendMessage(notif.Trade(info))

			if err != nil {
				logger.ErrorContext(ctx, "message is not sent", err)

				return err
			}
		}
	}

	return nil
}

func calculateProcentToDevident(price float64, devident float64) float64 {
	return devident * 100 / price
}
