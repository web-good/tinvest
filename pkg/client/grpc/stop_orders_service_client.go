package grpc

import (
	"context"
	"time"
	investapi "tinvest/internal/pb/v1"

	"google.golang.org/grpc"
)

type StopOrdersServiceClient interface {
	PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error)
	GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error)
	CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error)
}

type stopOrdersServiceClient struct {
	api  investapi.StopOrdersServiceClient
	auth *Auth
}

func NewStopOrdersServiceClient(conn grpc.ClientConnInterface, token string) StopOrdersServiceClient {
	return &stopOrdersServiceClient{
		api:  investapi.NewStopOrdersServiceClient(conn),
		auth: NewAuth(token),
	}
}

func (c *stopOrdersServiceClient) PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts = append(opts, NewRPCCredential(c.auth))
	return c.api.PostStopOrder(ctx, in, opts...)
}

func (c *stopOrdersServiceClient) GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts = append(opts, NewRPCCredential(c.auth))
	return c.api.GetStopOrders(ctx, in, opts...)
}

func (c *stopOrdersServiceClient) CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts = append(opts, NewRPCCredential(c.auth))
	return c.api.CancelStopOrder(ctx, in, opts...)
}
