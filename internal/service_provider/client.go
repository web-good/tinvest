package service_provider

import (
	"fmt"
	internalgrpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type client struct {
	grpcClient  internalgrpc.GrpcClient
	telegramBot telegram.Client
}

func (s *ServiceProvider) GetGrpcClient() (internalgrpc.GrpcClient, error) {
	if serviceProvider.client.grpcClient == nil {
		var err error
		serviceProvider.client.grpcClient, err = internalgrpc.NewClientGrpc(
			s.appConfig.GrpcClient.AddressProd,
			s.appConfig.GrpcClient.TokenProd,
		)

		if err != nil {
			return nil, err
		}
	}

	return serviceProvider.client.grpcClient, nil
}

func (s *ServiceProvider) GetTelegramBotClient() (telegram.Client, error) {
	if serviceProvider.client.telegramBot != nil {
		return serviceProvider.client.telegramBot, nil
	}

	var err error
	serviceProvider.client.telegramBot, err = telegram.InitTelegramBot(
		s.appConfig.TelegramClient.Token,
		s.appConfig.TelegramClient.ChatID,
	)

	if err != nil {
		return nil, fmt.Errorf("could not init telegram bot: %w", err)
	}

	return serviceProvider.client.telegramBot, nil
}

func (s *ServiceProvider) GetTelegramBotClientWithProxy() (telegram.Client, error) {
	if serviceProvider.client.telegramBot != nil {
		return serviceProvider.client.telegramBot, nil
	}

	var err error
	serviceProvider.client.telegramBot, err = telegram.InitTelegramBotProxy(
		s.appConfig.TelegramClient.Token,
		s.appConfig.TelegramClient.ChatID,
		"dedicated.love-internet.xyz:4515",
	)

	if err != nil {
		return nil, fmt.Errorf("could not init telegram bot: %w", err)
	}

	return serviceProvider.client.telegramBot, nil
}
