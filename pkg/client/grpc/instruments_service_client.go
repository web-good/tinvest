package grpc

import (
	"context"
	"google.golang.org/grpc"
	"time"
	investapi "tinvest/internal/pb/v1"
)

type InstrumentsServiceClient interface {
	Shares(ctx context.Context, in *investapi.InstrumentsRequest, opts ...grpc.CallOption) (*investapi.SharesResponse, error)
}

type instrumentsServiceClient struct {
	instrumentsApi investapi.InstrumentsServiceClient
}

func NewInstrumentsServiceClient(conn grpc.ClientConnInterface) InstrumentsServiceClient {
	return &instrumentsServiceClient{
		instrumentsApi: investapi.NewInstrumentsServiceClient(conn),
	}
}

func (c *instrumentsServiceClient) Shares(ctx context.Context, in *investapi.InstrumentsRequest, opts ...grpc.CallOption) (*investapi.SharesResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.instrumentsApi.Shares(ctx, in, opts...)
}
