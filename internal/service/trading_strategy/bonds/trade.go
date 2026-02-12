package bonds

import (
	"context"
	"sync"
	"time"
	"tinvest/internal/service/trading_strategy/bonds/computable"
	"tinvest/internal/service/trading_strategy/bonds/pipeline"
)

func (s *service) Trade(ctx context.Context) error {
	var wg sync.WaitGroup
	bonds, err := s.instrumentServiceGrpcClient.Bonds(ctx)
	if err != nil {
		return err
	}

	doneCh := make(chan struct{})
	wg.Add(1)
	go func() {
		pipeline.Sender(
			ctx,
			pipeline.CalculateProfit(
				ctx,
				doneCh,
				pipeline.Finder(doneCh, bonds, true, time.Now().AddDate(0, 0, 180), time.Now().AddDate(2, 0, 0)),
				computable.NewService(s.instrumentServiceGrpcClient, s.marketDataServiceGrpcClient),
			),
			s.tgClient,
			&wg,
			time.Now().AddDate(0, 0, 180),
			time.Now().AddDate(2, 0, 0),
		)
	}()
	wg.Add(1)
	go func() {
		pipeline.Sender(
			ctx,
			pipeline.CalculateProfit(
				ctx,
				doneCh,
				pipeline.Finder(doneCh, bonds, true, time.Now().AddDate(2, 0, 0), time.Now().AddDate(6, 0, 0)),
				computable.NewService(s.instrumentServiceGrpcClient, s.marketDataServiceGrpcClient),
			),
			s.tgClient,
			&wg,
			time.Now().AddDate(2, 0, 0),
			time.Now().AddDate(6, 0, 0),
		)
	}()
	wg.Add(1)
	go func() {
		pipeline.Sender(
			ctx,
			pipeline.CalculateProfit(
				ctx,
				doneCh,
				pipeline.Finder(doneCh, bonds, true, time.Now().AddDate(6, 0, 0), time.Now().AddDate(16, 0, 0)),
				computable.NewService(s.instrumentServiceGrpcClient, s.marketDataServiceGrpcClient),
			),
			s.tgClient,
			&wg,
			time.Now().AddDate(6, 0, 0),
			time.Now().AddDate(16, 0, 0),
		)
	}()
	wg.Add(1)
	go func() {
		pipeline.Sender(
			ctx,
			pipeline.CalculateProfit(
				ctx,
				doneCh,
				pipeline.Finder(doneCh, bonds, false, time.Now().AddDate(0, 0, 180), time.Now().AddDate(3, 0, 0)),
				computable.NewService(s.instrumentServiceGrpcClient, s.marketDataServiceGrpcClient),
			),
			s.tgClient,
			&wg,
			time.Now().AddDate(0, 0, 180),
			time.Now().AddDate(3, 0, 0),
		)
	}()
	wg.Wait()

	return nil
}
