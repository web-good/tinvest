package app

import (
	"context"
	"tinvest/internal/service_provider"
	"tinvest/pkg/logger"
)

func (a *App) initServiceProvider(ctx context.Context) error {
	logger.InfoContext(ctx, "Start initializing service provider")
	a.sp = service_provider.GetServiceProvider(ctx)
	logger.InfoContext(ctx, "Initializing service provider")

	return nil
}
