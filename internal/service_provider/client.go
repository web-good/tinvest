package service_provider

import (
	"fmt"
	"log/slog"
	internalgrpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

type client struct {
	grpcClient          internalgrpc.GrpcClient
	reversionGrpcClient internalgrpc.GrpcClient
	telegramBot         *telegram.Bot
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

// GetReversionGrpcClient returns a gRPC client authenticated with the reversion
// strategy's dedicated token (REVERSION_TOKEN), separate from the shared T_BANK
// client. It dials the same AddressProd; the second connection is negligible for
// an hourly cron strategy and keeps the reversion account fully isolated.
func (s *ServiceProvider) GetReversionGrpcClient() (internalgrpc.GrpcClient, error) {
	if serviceProvider.client.reversionGrpcClient == nil {
		var err error
		serviceProvider.client.reversionGrpcClient, err = internalgrpc.NewClientGrpc(
			s.appConfig.GrpcClient.AddressProd,
			s.appConfig.Reversion.Token,
		)
		if err != nil {
			return nil, err
		}
	}

	return serviceProvider.client.reversionGrpcClient, nil
}

func (s *ServiceProvider) GetTelegramBot() (*telegram.Bot, error) {
	if serviceProvider.client.telegramBot != nil {
		return serviceProvider.client.telegramBot, nil
	}

	var err error
	serviceProvider.client.telegramBot, err = telegram.InitTelegramBot(
		s.appConfig.TelegramClient.Token,
		s.appConfig.TelegramClient.GroupChatID,
	)
	if err != nil {
		return nil, fmt.Errorf("could not init telegram bot: %w", err)
	}

	return serviceProvider.client.telegramBot, nil
}

func (s *ServiceProvider) GetTelegramBotClient() (telegram.Client, error) {
	return s.GetTelegramBot()
}

func (s *ServiceProvider) GetGoldenXSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicGoldenX, "golden_x")
}

func (s *ServiceProvider) GetReversionSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicReversion, "reversion")
}

// topicSender строит Client, привязанный к теме форума; при незаданном ID
// темы сообщения уходят в General (threadID 0), о чём предупреждаем в логе.
func (s *ServiceProvider) topicSender(threadID int, name string) (telegram.Client, error) {
	base, err := s.GetTelegramBot()
	if err != nil {
		return nil, err
	}
	if threadID == 0 {
		logger.Warn("telegram topic id is not set, sending to General", slog.String("topic", name))
	}

	return telegram.NewTopicSender(base, s.appConfig.TelegramClient.GroupChatID, threadID), nil
}
