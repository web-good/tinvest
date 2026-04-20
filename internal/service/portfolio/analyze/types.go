package analyze

import (
	"context"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type Analyze interface {
	BondsPortfolio(context context.Context) error
}

type service struct {
	operationsServiceClient     grpc.OperationsServiceClient
	usersServiceClient          grpc.UsersServiceClient
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	tgClient                    telegram.Client
}

func NewService(operationsServiceClient grpc.OperationsServiceClient, usersServiceClient grpc.UsersServiceClient, tgClient telegram.Client, instrumentsServiceClient grpc.InstrumentsServiceClient) *service {
	return &service{
		operationsServiceClient:     operationsServiceClient,
		instrumentServiceGrpcClient: instrumentsServiceClient,
		usersServiceClient:          usersServiceClient,
		tgClient:                    tgClient,
	}
}
