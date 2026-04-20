package grpc

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	BondByID(ctx context.Context, id string) (*pkgmodel.Bond, error)
	GetBondCoupons(instrumentId string, from time.Time, to time.Time) ([]*pkgmodel.BondCoupon, error)
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

func (c *instrumentsServiceClient) GetBondCoupons(instrumentId string, from time.Time, to time.Time) ([]*pkgmodel.BondCoupon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsApi.GetBondCoupons(
		ctx,
		&investapi.GetBondCouponsRequest{
			From:         timestamppb.New(from),
			To:           timestamppb.New(to),
			InstrumentId: instrumentId,
		},
		NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request BondCoupons: %w", err)
	}

	return converter2.ConvertCouponsFromPb(resp.Events), nil
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

func (c *instrumentsServiceClient) BondByID(ctx context.Context, id string) (*pkgmodel.Bond, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsApi.BondBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID,
		Id:     id,
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request share by id: %w", err)
	}

	return converter2.ConvertBondModelFromBondPb(resp.Instrument), nil
}
