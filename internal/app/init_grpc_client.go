package app

import (
	"context"
	"tinvest/pkg/logger"
)

func (a *App) initGrpcClient(ctx context.Context) error {
	logger.InfoContext(ctx, "Initialize gRPC client")
	_, err := a.sp.GetGrpcClient()

	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "Initialize gRPC client")

	return nil
}
