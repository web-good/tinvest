package service_provider

import "context"

type ServiceProvider struct {
	ctx     context.Context
	service service
	client  client
}

var serviceProvider *ServiceProvider

func GetServiceProvider(ctx context.Context) *ServiceProvider {
	if serviceProvider == nil {
		serviceProvider = new(ServiceProvider)
		serviceProvider.ctx = ctx
	}

	return serviceProvider
}
