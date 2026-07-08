package scalping_rsi

import (
	"context"
	"fmt"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/scalping_rsi/dto"
	notif "tinvest/internal/service/trading_strategy/scalping_rsi/notification"
	"tinvest/internal/service/trading_strategy/scalping_rsi/specification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	//info := domain.NewInfo()
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	emaSp := specification.Ema{}
	rsiSp := specification.Rsi{Value: 29}
	info := domain.NewInfo()

	for _, share := range t {
		fmt.Println(share.Name)
		time.Sleep(500 * time.Millisecond)
		interval := int32(3)
		ema, errEma := s.ema.TechAnalyse(ctx, &share.ID, int32(in.Interval), utils.TimeGenerator(dateNow, -400, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 150)
		if errEma != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema :%w", errEma).Error())

			continue
		}

		price, errPrice := s.marketDataServiceGrpcClient.GetCandles(ctx, &share.ID, int32(in.Interval), utils.TimeStampPbGenerator(dateNow, -2, in.Interval), timestamppb.New(dateNow), &interval, true)
		if errPrice != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate price :%w", errPrice).Error())

			continue
		}

		rsi, rsiErr := s.rsi.CalculateRSI(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -15, in.Interval), timestamppb.New(dateNow), 5)
		if rsiErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate RSI :%w", rsiErr).Error())

			continue
		}

		if !emaSp.IsSatisfiedBy(ema[len(ema)-1], *price[1]) {
			continue
		}

		if !rsiSp.IsSatisfiedBy(*rsi[1]) {
			continue
		}

		info.WriteToMap(share.ID, domain.Item{InstrumentName: share.Name})

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
