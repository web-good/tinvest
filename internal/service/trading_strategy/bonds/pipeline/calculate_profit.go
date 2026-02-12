package pipeline

import (
	"context"
	"log/slog"
	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/bonds/computable"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"
)

func CalculateProfit(ctx context.Context, doneCh chan struct{}, bondsCh <-chan *pkgmodel.Bond, computableSrc computable.Computable) <-chan domain.BondReport {
	out := make(chan domain.BondReport)

	go func() {
		defer close(out)
		for bond := range bondsCh {
			profit, err := computableSrc.CalculateProfit(ctx, bond)
			if err != nil {
				logger.ErrorContext(ctx, "error in CalculateProfit", slog.String("error_msg", err.Error()))
				close(doneCh)
			}

			select {
			case <-doneCh:
				return
			case out <- profit:
			}
		}
	}()

	return out
}
