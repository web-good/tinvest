package grpc

import (
	"context"
	"google.golang.org/grpc"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/converter"
	"tinvest/pkg/client/grpc/model"
)

type OperationsServiceClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]model.Position, error)
}

type operationsServiceClient struct {
	operationApi investapi.OperationsServiceClient
	auth         *Auth
}

func NewOperationsServiceClient(conn grpc.ClientConnInterface, token string) OperationsServiceClient {
	return &operationsServiceClient{
		operationApi: investapi.NewOperationsServiceClient(conn),
		auth:         NewAuth(token),
	}
}

func (o *operationsServiceClient) GetPortfolio(ctx context.Context, accountID string) ([]model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var cur investapi.PortfolioRequest_CurrencyRequest
	cur = investapi.PortfolioRequest_RUB
	resp, err := o.operationApi.GetPortfolio(ctx, &investapi.PortfolioRequest{
		AccountId: accountID,
		Currency:  &cur,
	}, NewRPCCredential(o.auth))

	if err != nil {
		return nil, err
	}

	return converter.ConvertPortfolioFromBp(resp), nil
}
