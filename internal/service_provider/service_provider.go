package service_provider

import (
	"context"
	"tinvest/internal/config"
)

type ServiceProvider struct {
	ctx       context.Context
	appConfig *config.Config
	service   service
	client    client
}

var serviceProvider *ServiceProvider

func GetServiceProvider(ctx context.Context, appConfig *config.Config) *ServiceProvider {
	if serviceProvider == nil {
		serviceProvider = new(ServiceProvider)
		serviceProvider.ctx = ctx
		serviceProvider.appConfig = appConfig
	}

	return serviceProvider
}
