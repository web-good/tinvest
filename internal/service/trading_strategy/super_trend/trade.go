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
		fastEma, _ := s.marketDataServiceGrpcClient.GetTechAnalyseEma(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 5)
		openEma, _ := s.marketDataServiceGrpcClient.GetTechAnalyseEma(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 35)
		sp := specification.SuperTrendIntersection{}

		if sp.IsSatisfiedBy(fastEma, openEma) == false {
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
	fmt.Println(shares)
	return nil
}
