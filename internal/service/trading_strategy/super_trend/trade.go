package super_trend

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context) error {
	var shares []string
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		ema35, err35 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -7), 35)
		ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), 5)

		if err35 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema35", err35)
		}

		if err5 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema5", err5)
		}

		sp := specification.SuperTrendIntersection{}

		if sp.IsSatisfiedBy(ema5, ema35) == false {
			continue
		}

		logger.InfoContext(ctx, "Entered the condition ema intersection", share.Name)
		macDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 11, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 12)
		greenMacDSp := specification.GreenMacD{}

		if greenMacDSp.IsSatisfiedBy(macDModel) == false {
			continue
		}

		logger.InfoContext(ctx, "Entered the condition macd intersection", share.Name)
		//s.atr.TechAnalyse(ctx, &share.ID)
		shares = append(shares, share.Name)
	}

	if len(shares) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(shares))

		if err != nil {
			return err
		}
	}

	return nil
}
