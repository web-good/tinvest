package grpc

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"time"
	"tinvest/internal/converter"
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

type InstrumentsServiceClient interface {
	Shares(ctx context.Context) ([]*model.Share, error)
}

type instrumentsServiceClient struct {
	instrumentsApi investapi.InstrumentsServiceClient
	auth           *Auth
}

func NewInstrumentsServiceClient(conn grpc.ClientConnInterface, token string) InstrumentsServiceClient {
	return &instrumentsServiceClient{
		instrumentsApi: investapi.NewInstrumentsServiceClient(conn),
		auth:           NewAuth(token),
	}
}

func (c *instrumentsServiceClient) Shares(ctx context.Context) ([]*model.Share, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsApi.Shares(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus:   investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE.Enum(),
		InstrumentExchange: investapi.InstrumentExchangeType_INSTRUMENT_EXCHANGE_UNSPECIFIED.Enum(),
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request shares: %w", err)
	}

	return converter.ConvertSharesFromPb(resp.Instruments), nil
}
