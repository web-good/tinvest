package grpc

import (
	"context"
	"google.golang.org/grpc"
	"time"
	investapi "tinvest/internal/pb/v1"
)

type OrdersServiceClient interface {
	PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error)
}

type ordersServiceClient struct {
	orderApi investapi.OrdersServiceClient
}

func NewOrdersServiceClient(conn grpc.ClientConnInterface) OrdersServiceClient {
	return &ordersServiceClient{
		orderApi: investapi.NewOrdersServiceClient(conn),
	}
}

func (c *ordersServiceClient) PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.orderApi.PostOrder(ctx, in, opts...)
}
