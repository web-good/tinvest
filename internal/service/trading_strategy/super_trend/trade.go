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
		//роверяем что macd зелёный на дневке
		greenMacDSp := specification.GreenMacD{}
		dMacDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 5, timestamppb.New(time.Now().AddDate(0, 0, -20)), timestamppb.New(time.Now()), 12)

		if greenMacDSp.IsSatisfiedBy(dMacDModel) == false {
			continue
		}

		hMacD, err := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -8)), timestamppb.New(time.Now()), 12)

		if err != nil {
			logger.ErrorContext(ctx, "cannot get day macd", err, share.Name)

			continue
		}

		if s.yellow(hMacD) {
			s.addToPool(ctx, share, notification2.Yellow)

			continue
		}

		gr, err := s.green(ctx, share, hMacD)

		if err != nil {
			logger.ErrorContext(ctx, err.Error(), share.Name)
		}

		if gr == true {
			s.addToPool(ctx, share, notification2.Green)
		}
	}

	if len(n) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(n))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}

func (s *service) yellow(macdItems []*model.MacDItemTechAnalyse) bool {
	//ищем пересечение macd на часовике
	macDs := specification.MacDSpecification{}

	if macDs.IsSatisfiedBy(macdItems) {
		return true
	}

	return false
}

func (s *service) green(ctx context.Context, share *model.Share, macdItems []*model.MacDItemTechAnalyse) (bool, error) {
	rsiModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()))

	if err != nil {
		return false, fmt.Errorf("failed to get rsi :%w", err)
	}

	rsiS := specification.RsiSpecification{}

	if rsiS.IsSatisfiedBy(rsiModel) != true {
		return false, nil
	}

	macDSp := specification.GreenMacD{}

	if macDSp.IsSatisfiedBy(macdItems) != true {
		return false, nil
	}

	ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), 5)

	if err5 != nil {
		return false, fmt.Errorf("error in calculate ema5 :%w", err5)
	}

	ema35, err35 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -7), 35)

	if err35 != nil {
		return false, fmt.Errorf("error in calculate ema35 :%w", err35)
	}

	sp := specification.SuperTrendIntersection{}

	if sp.IsSatisfiedBy(ema5, ema35) != true {
		return false, nil
	}

	return true, nil
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
