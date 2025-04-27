package grpc

import (
	"google.golang.org/grpc"
)

type ClientGrpc interface {
	OrdersServiceClient() OrdersServiceClient
	InstrumentsServiceClient() InstrumentsServiceClient
}

type Client struct {
	ordersServiceClient      OrdersServiceClient
	instrumentsServiceClient InstrumentsServiceClient
}

func (c Client) InstrumentsServiceClient() InstrumentsServiceClient {
	return c.instrumentsServiceClient
}

func (c Client) OrdersServiceClient() OrdersServiceClient {
	return c.ordersServiceClient
}

func NewClientGrpc(conn grpc.ClientConnInterface) ClientGrpc {
	return &Client{
		ordersServiceClient: NewOrdersServiceClient(conn),
	}
}
