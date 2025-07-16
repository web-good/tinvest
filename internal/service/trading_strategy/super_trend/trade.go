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

		ema35, err35 := s.ema.TechAnalyse(ctx, &share.ID, 11, time.Now().AddDate(0, 0, -20), 35)

		if err35 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema35 4h :%w,  %s", err35, share.Name).Error())

			continue
		}

		sp := specification.SuperTrendIntersection{}

		if sp.IsSatisfiedBy(ema5, ema35) != true {
			continue
		}

		//смотрим что на 1ч ema5 выше ema 35
		ema1h5, err1h5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), 5)

		if err1h5 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema5 1h:%w,  %s", err1h5, share.Name).Error())
		}

		ema1h35, err1h35 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -7), 35)

		if err1h35 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema35 1h:%w,  %s", err1h35, share.Name).Error())
		}

		sp1h := specification.SuperTrendIntersection{}

		if sp1h.IsSatisfiedBy(ema1h5, ema1h35) != true {
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
