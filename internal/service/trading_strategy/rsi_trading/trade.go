package rsi_trading

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/service/trading_strategy/rsi_trading/specification"
)

func (s *service) Trade(ctx context.Context, interval int) error {
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	for _, x := range t {

		//EMA
		//y, _ := s.marketDataServiceGrpcClient.GetTechAnalyseEma(ctx, x.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 200)
		//ema := specification.EmaSpecification{}
		//ema.IsSatisfiedBy(y)
		//	fmt.Println(x)
		rsiModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, x.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()))

		rsiS := specification.RsiSpecification{}
		macDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, x.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 9)
		macDSpecification := specification.MacDSpecification{}

		if macDSpecification.IsSatisfiedBy(macDModel) == true && rsiS.IsSatisfiedBy(rsiModel) == true {
			fmt.Println("|||||||||||||||||||||||||||", x)
		}
	}

	//y, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, "87db07bc-0e02-4e29-90bb-05e8ef791d7b", 11, timestamppb.New(time.Now().AddDate(0, 0, -3)), timestamppb.New(time.Now()))
	//for k, x := range y {
	//fmt.Println(k, x)
	//}
	//dSpecification := specification.MacDSpecification{}
	//fmt.Println(dSpecification.IsSatisfiedBy(y))

	return nil
}
