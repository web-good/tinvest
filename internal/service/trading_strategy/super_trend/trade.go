package super_trend

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	notification2 "tinvest/internal/domain/notification"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
	"tinvest/pkg/logger"
)

var (
	n []notification2.SuperTrend
)

func (s *service) Trade(ctx context.Context) error {
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		//проверяем что macd зелёный на дневке
		greenMacDSp := specification.GreenMacD{}
		dMacDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 5, timestamppb.New(time.Now().AddDate(0, 0, -20)), timestamppb.New(time.Now()), 12)

		if greenMacDSp.IsSatisfiedBy(dMacDModel) == false {
			continue
		}

		//смотрим что на 4ч ema5 выше ema 35
		ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, 11, time.Now().AddDate(0, 0, -10), 5)

		if err5 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema5 4h :%w,  %s", err5, share.Name).Error())

			continue
		}

		ema20, err20 := s.ema.TechAnalyse(ctx, &share.ID, 11, time.Now().AddDate(0, 0, -20), 20)

		if err20 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema20 4h :%w,  %s", err20, share.Name).Error())

			continue
		}

		sp := specification.SuperTrendIntersection{}

		if sp.IsSatisfiedBy(ema5, ema20) != true {
			continue
		}

		//смотрим что на 1ч ema5 выше ema 35
		ema1h10, err1h10 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), 10)

		if err1h10 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema10 1h:%w,  %s", err1h10, share.Name).Error())
		}

		ema1h40, err1h40 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -7), 40)

		if err1h40 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema40 1h:%w,  %s", err1h40, share.Name).Error())
		}

		sp1h := specification.SuperTrendIntersection{}

		if sp1h.IsSatisfiedBy(ema1h10, ema1h40) != true {
			continue
		}

		//смотрим что rsi пересёк 50 55
		rsiModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()))

		if err != nil {
			logger.ErrorContext(ctx, fmt.Errorf("failed to get rsi :%w,  %s", err, share.Name).Error())
		}

		rsiS := specification.RsiSpecification{}

		if rsiS.IsSatisfiedBy(rsiModel) != true {
			continue
		}

		s.addToPool(ctx, share, notification2.Green)
	}

	if len(n) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(n))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}

		n = []notification2.SuperTrend{}
	}

	return nil
}

func (s *service) addToPool(ctx context.Context, share *model.Share, indicator notification2.Color) {
	notif := notification2.SuperTrend{
		Share:     *share,
		Indicator: indicator,
	}

	var atrErr error
	notif.Atr, atrErr = s.atr.TechAnalyse(ctx, &share.ID)

	if atrErr != nil {
		logger.ErrorContext(ctx, "Failed to get ATR", share.Name)
	}

	n = append(n, notif)
}
