package super_trend

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
)

func (s *service) Trade(ctx context.Context) error {
	var shares []string
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		if share.ID != "459a1a0a-0253-465a-bd4e-afaaf5e670b0" {
			continue
		}
		ema35, _ := s.emaData.TechAnalyse(ctx, &share.ID, 4, time.Now().Add(-35*time.Hour), int32(35))
		ema5, _ := s.emaData.TechAnalyse(ctx, &share.ID, 4, time.Now().Add(-6*time.Hour), int32(5))
		fmt.Println(share, "-----------", ema35, "================", ema5)
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
