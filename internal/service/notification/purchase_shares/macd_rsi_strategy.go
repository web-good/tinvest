package purchase_shares

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

func (b *Service) MacdRsiStrategy(ctx context.Context) error {
	r, _ := b.instrumentServiceGrpcClient.Shares(ctx)
	for k, v := range r {
		fmt.Println(k, v)
	}
	//y, err := b.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, "e6123145-9665-43e0-8413-cd61b8aa9b13", 11, timestamppb.New(time.Now().Add(-24*time.Hour)), timestamppb.New(time.Now()))
	//y, err := b.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, "e6123145-9665-43e0-8413-cd61b8aa9b13", 11, timestamppb.New(time.Now().Add(-24*time.Hour)), timestamppb.New(time.Now()))
	//y, err := b.marketDataServiceGrpcClient.GetTechAnalyseEma(ctx, "e6123145-9665-43e0-8413-cd61b8aa9b13", 11, timestamppb.New(time.Now().Add(-24*time.Hour)), timestamppb.New(time.Now()), 40)
	y, err := b.marketDataServiceGrpcClient.GetTechAnalyseBB(ctx, "e6123145-9665-43e0-8413-cd61b8aa9b13", 11, timestamppb.New(time.Now().Add(-24*time.Hour)), timestamppb.New(time.Now()))

	for _, v := range y {
		fmt.Println(v, "from", timestamppb.New(time.Now().Add(-8*time.Hour)).AsTime(), time.Now())
	}
	if err != nil {
		return err
	}

	return nil
}
