package yield

import (
	"context"

	"tinvest/internal/config"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

// Yield is the interface for portfolio yield computation.
type Yield interface {
	PortfolioYieldYTD(ctx context.Context, tg telegram.Client) error
}

type service struct {
	operationsServiceClient grpc.OperationsServiceClient
	usersServiceClient      grpc.UsersServiceClient
	manualStartValue        float64
}

// NewService creates a new yield service.
func NewService(
	operationsServiceClient grpc.OperationsServiceClient,
	usersServiceClient grpc.UsersServiceClient,
	cfg *config.PortfolioYieldConfig,
) *service {
	return &service{
		operationsServiceClient: operationsServiceClient,
		usersServiceClient:      usersServiceClient,
		manualStartValue:        cfg.ManualStartValue,
	}
}
