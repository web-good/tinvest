package app

import (
	"context"
	"tinvest/pkg/logger"
)

const (
	address = "invest-public-api.tinkoff.ru:443"
)

func (a *App) initGrpcClient(ctx context.Context) error {
	logger.InfoContext(ctx, "Initialize gRPC client")
	_, err := a.sp.GetGrpcClient(a.config.GrpcClient.AddressProd, a.config.GrpcClient.TokenProd)

	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "Initialize gRPC client")

	return nil
}
