package grpc

import (
	"context"
	"time"
	investapi "tinvest/internal/pb/v1"

	"google.golang.org/grpc"
)

type OrdersServiceClient interface {
	GetOrders(context.Context, *investapi.GetOrdersRequest, ...grpc.CallOption) (*investapi.GetOrdersResponse, error)
	CancelOrder(ctx context.Context, in *investapi.CancelOrderRequest, opts ...grpc.CallOption) (*investapi.CancelOrderResponse, error)
	GetOrderState(ctx context.Context, in *investapi.GetOrderStateRequest, opts ...grpc.CallOption) (*investapi.OrderState, error)
	PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error)
}

type ordersServiceClient struct {
	orderAPI investapi.OrdersServiceClient
	auth     *Auth
}

func NewOrdersServiceClient(conn grpc.ClientConnInterface, token string) OrdersServiceClient {
	return &ordersServiceClient{
		orderAPI: investapi.NewOrdersServiceClient(conn),
		auth:     NewAuth(token),
	}
}

func (c *ordersServiceClient) PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	opts = append(opts, NewRPCCredential(c.auth))
	return c.orderAPI.PostOrder(ctx, in, opts...)
}

func (c *ordersServiceClient) GetOrderState(ctx context.Context, in *investapi.GetOrderStateRequest, opts ...grpc.CallOption) (*investapi.OrderState, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts = append(opts, NewRPCCredential(c.auth))
	return c.orderAPI.GetOrderState(ctx, in, opts...)
}

func (c *ordersServiceClient) CancelOrder(ctx context.Context, in *investapi.CancelOrderRequest, opts ...grpc.CallOption) (*investapi.CancelOrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts = append(opts, NewRPCCredential(c.auth))
	return c.orderAPI.CancelOrder(ctx, in, opts...)
}

func (c *ordersServiceClient) GetOrders(ctx context.Context, in *investapi.GetOrdersRequest, opts ...grpc.CallOption) (*investapi.GetOrdersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts = append(opts, NewRPCCredential(c.auth))
	return c.orderAPI.GetOrders(ctx, in, opts...)
}
