package super_trend

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
)

func (s *service) Trade(ctx context.Context) error {
	var shares []string
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		ema35, err35 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().Add(-70*time.Hour), 35)
		ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().Add(-10*time.Hour), 5)

		if err35 != nil || err5 != nil {
			continue
		}

		sp := specification.SuperTrendIntersection{}

		if sp.IsSatisfiedBy(ema5, ema35) == false {
			continue
		}

		macDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 11, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 12)
		greenMacDSp := specification.GreenMacD{}

		if greenMacDSp.IsSatisfiedBy(macDModel) == false {
			continue
		}
		s.atr.TechAnalyse(ctx, &share.ID)
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
