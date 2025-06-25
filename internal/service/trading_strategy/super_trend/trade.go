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
		ema35, _ := s.emaData.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), int32(35))
		ema5, _ := s.emaData.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -1), int32(5))

		if len(ema35) == 0 || len(ema5) == 0 {
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
