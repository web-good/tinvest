package analyze

import (
	"context"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"
)

type Analyze interface {
	BondsPortfolio(ctx context.Context, tg telegram.Client) error
}

type service struct {
	operationsServiceClient     grpc.OperationsServiceClient
	usersServiceClient          grpc.UsersServiceClient
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
}

func NewService(operationsServiceClient grpc.OperationsServiceClient, usersServiceClient grpc.UsersServiceClient, instrumentsServiceClient grpc.InstrumentsServiceClient) *service {
	return &service{
		operationsServiceClient:     operationsServiceClient,
		instrumentServiceGrpcClient: instrumentsServiceClient,
		usersServiceClient:          usersServiceClient,
	}
}
