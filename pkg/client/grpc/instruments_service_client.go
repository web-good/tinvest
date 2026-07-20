package grpc

import (
	"context"
	"fmt"
	"time"
	"tinvest/internal/converter"
	"tinvest/internal/domain/share"
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	converter2 "tinvest/pkg/client/grpc/converter"
	pkgmodel "tinvest/pkg/client/grpc/model"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InstrumentsServiceClient interface {
	Shares(ctx context.Context) ([]*model.Share, error)
	ShareByID(ctx context.Context, id string) (*share.Share, error)
	Bonds(ctx context.Context) ([]*pkgmodel.Bond, error)
	BondByID(ctx context.Context, id string) (*pkgmodel.Bond, error)
	GetBondCoupons(instrumentID string, from time.Time, to time.Time) ([]*pkgmodel.BondCoupon, error)
	GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error)
}

type instrumentsServiceClient struct {
	instrumentsAPI investapi.InstrumentsServiceClient
	auth           *Auth
}

func NewInstrumentsServiceClient(conn grpc.ClientConnInterface, token string) InstrumentsServiceClient {
	return &instrumentsServiceClient{
		instrumentsAPI: investapi.NewInstrumentsServiceClient(conn),
		auth:           NewAuth(token),
	}
}

func (c *instrumentsServiceClient) Bonds(ctx context.Context) ([]*pkgmodel.Bond, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsAPI.Bonds(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus:   investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE.Enum(),
		InstrumentExchange: investapi.InstrumentExchangeType_INSTRUMENT_EXCHANGE_UNSPECIFIED.Enum(),
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request shares: %w", err)
	}

	return converter2.ConvertBondsFromPb(resp), nil
}

func (c *instrumentsServiceClient) GetBondCoupons(instrumentID string, from time.Time, to time.Time) ([]*pkgmodel.BondCoupon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsAPI.GetBondCoupons(
		ctx,
		&investapi.GetBondCouponsRequest{
			From:         timestamppb.New(from),
			To:           timestamppb.New(to),
			InstrumentId: instrumentID,
		},
		NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request BondCoupons: %w", err)
	}

	return converter2.ConvertCouponsFromPb(resp.Events), nil
}

func (c *instrumentsServiceClient) Shares(ctx context.Context) ([]*model.Share, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsAPI.Shares(ctx, &investapi.InstrumentsRequest{
		InstrumentStatus:   investapi.InstrumentStatus_INSTRUMENT_STATUS_BASE.Enum(),
		InstrumentExchange: investapi.InstrumentExchangeType_INSTRUMENT_EXCHANGE_UNSPECIFIED.Enum(),
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request shares: %w", err)
	}

	return converter.ConvertSharesFromPb(resp.Instruments), nil
}

func (c *instrumentsServiceClient) ShareByID(ctx context.Context, id string) (*share.Share, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsAPI.ShareBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID,
		Id:     id,
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request share by id: %w", err)
	}

	return converter2.ConvertShareFromPb(resp.Instrument), nil
}

func (c *instrumentsServiceClient) BondByID(ctx context.Context, id string) (*pkgmodel.Bond, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := c.instrumentsAPI.BondBy(ctx, &investapi.InstrumentRequest{
		IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID,
		Id:     id,
	}, NewRPCCredential(c.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request share by id: %w", err)
	}

	return converter2.ConvertBondModelFromBondPb(resp.Instrument), nil
}

func (c *instrumentsServiceClient) GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error) {
	const batchSize = 100
	res := make([]*model.Fundamentals, 0, len(assetUIDs))
	for start := 0; start < len(assetUIDs); start += batchSize {
		end := start + batchSize
		if end > len(assetUIDs) {
			end = len(assetUIDs)
		}
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		resp, err := c.instrumentsAPI.GetAssetFundamentals(
			reqCtx,
			&investapi.GetAssetFundamentalsRequest{Assets: assetUIDs[start:end]},
			NewRPCCredential(c.auth),
		)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("failed to request fundamentals: %w", err)
		}
		res = append(res, converter.ConvertFundamentalsFromPb(resp.Fundamentals)...)
	}
	return res, nil
}
