package computable

import (
	"context"
	"tinvest/internal/domain"
	"tinvest/pkg/client/grpc"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

type Computable interface {
	CalculateProfit(ctx context.Context, bond *pkgmodel.Bond) (domain.BondReport, error)
}

type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
}

func NewService(
	instrumentsServiceClient grpc.InstrumentsServiceClient,
	marketDataServiceGrpcClient grpc.MarketDataServiceClient) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
	}
}
