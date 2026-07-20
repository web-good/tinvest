package bonds

import (
	"context"
	"sync"
	"time"
	"tinvest/internal/service/trading_strategy/bonds/computable"
	"tinvest/internal/service/trading_strategy/bonds/pipeline"
	"tinvest/pkg/client/telegram"
)

func (s *service) Trade(ctx context.Context, tg telegram.Client) error {
	var wg sync.WaitGroup
	bonds, err := s.instrumentServiceGrpcClient.Bonds(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, r := range DefaultLadder(now) {
		wg.Add(1)
		go func(r Rung) {
			doneCh := make(chan struct{})
			pipeline.Sender(
				ctx,
				pipeline.CalculateProfit(
					ctx,
					doneCh,
					pipeline.Finder(doneCh, bonds, r.IsOfz, r.From, r.To),
					computable.NewService(s.instrumentServiceGrpcClient, s.marketDataServiceGrpcClient),
				),
				tg,
				&wg,
				r.From,
				r.To,
			)
		}(r)
	}
	wg.Wait()

	return nil
}
