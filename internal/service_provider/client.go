package service_provider

import (
	"fmt"
	"tinvest/pkg/client/db"
	"tinvest/pkg/client/db/pg"
	internalgrpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/closer"
)

type client struct {
	grpcClient  internalgrpc.GrpcClient
	dbClient    db.Client
	telegramBot telegram.Client
}

func (s *ServiceProvider) GetGrpcClient(address string, token string) (internalgrpc.GrpcClient, error) {
	if serviceProvider.client.grpcClient == nil {
		var err error
		serviceProvider.client.grpcClient, err = internalgrpc.NewClientGrpc(address, token)

		if err != nil {
			return nil, err
		}
	}

	return serviceProvider.client.grpcClient, nil
}

func (s *ServiceProvider) GetDbClient(dsn string) (db.Client, error) {
	if serviceProvider.client.dbClient == nil {
		var err error
		serviceProvider.client.dbClient, err = pg.New(
			s.ctx,
			dsn,
		)

		if err != nil {
			return nil, fmt.Errorf("could not connect to database: %w", err)
		}

		closer.Add(func() error {
			err := serviceProvider.client.dbClient.Db().Close()

			if err != nil {
				return err
			}

			return nil
		})

		err = serviceProvider.client.dbClient.Db().Ping(s.ctx)

		if err != nil {
			return nil, fmt.Errorf("ping to database error: %w", err)
		}
	}

	return serviceProvider.client.dbClient, nil
}

func (s *ServiceProvider) GetTelegramBotClient(token string, chatId int64) (telegram.Client, error) {
	if serviceProvider.client.telegramBot != nil {
		return serviceProvider.client.telegramBot, nil
	}

	var err error
	serviceProvider.client.telegramBot, err = telegram.InitTelegramBot(token, chatId)

	if err != nil {
		return nil, fmt.Errorf("could not init telegram bot: %w", err)
	}

	return serviceProvider.client.telegramBot, nil
}
