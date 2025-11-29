package grpc

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"time"
	"tinvest/internal/converter"
	"tinvest/internal/domain/share"
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	converter2 "tinvest/pkg/client/grpc/converter"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

type InstrumentsServiceClient interface {
	Shares(ctx context.Context) ([]*model.Share, error)
	ShareByID(ctx context.Context, id string) (*share.Share, error)
	Bonds(ctx context.Context) ([]*pkgmodel.Bond, error)
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

func (c *instrumentsServiceClient) Bonds(ctx context.Context) ([]*pkgmodel.Bond, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsApi.Bonds(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus:   investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE.Enum(),
		InstrumentExchange: investapi.InstrumentExchangeType_INSTRUMENT_EXCHANGE_UNSPECIFIED.Enum(),
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request shares: %w", err)
	}

	return converter2.ConvertBondsFromPb(resp), nil
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

func (c *instrumentsServiceClient) ShareByID(ctx context.Context, id string) (*share.Share, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsApi.ShareBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID,
		Id:     id,
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request share by id: %w", err)
	}

	return converter2.ConvertShareFromPb(resp.Instrument), nil
}
