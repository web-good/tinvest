package purchase_shares

import (
	"context"
	"tinvest/pkg/client/grpc"
)

var _ PurchaseShares = (*Service)(nil)

type Service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
}

type PurchaseShares interface {
	MacdRsiStrategy(context.Context) error
}

func NewService(instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient) PurchaseShares {
	return &Service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
	}
}
