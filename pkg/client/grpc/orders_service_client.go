package grpc

import (
	"context"
	"time"
	investapi "tinvest/internal/pb/v1"

	"google.golang.org/grpc"
)

type OrdersServiceClient interface {
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
