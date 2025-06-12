package grpc

import (
	"context"
	"crypto/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"strings"
	"time"
)

type GrpcClient interface {
	OrdersServiceClient() OrdersServiceClient
	InstrumentsServiceClient() InstrumentsServiceClient
	MarketDataServiceClient() MarketDataServiceClient
	Connection() grpc.ClientConnInterface
}

type Client struct {
	ordersServiceClient      OrdersServiceClient
	instrumentsServiceClient InstrumentsServiceClient
	marketDataServiceClient  MarketDataServiceClient
	conn                     grpc.ClientConnInterface
}

func (c Client) MarketDataServiceClient() MarketDataServiceClient {
	return c.marketDataServiceClient
}

func (c Client) InstrumentsServiceClient() InstrumentsServiceClient {
	return c.instrumentsServiceClient
}

func (c Client) OrdersServiceClient() OrdersServiceClient {
	return c.ordersServiceClient
}

func (c Client) Connection() grpc.ClientConnInterface {
	return c.conn
}

func NewClientGrpc(address string, token string) (GrpcClient, error) {
	params := strings.Split(address, ":")
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			ServerName: params[0],
			MinVersion: tls.VersionTLS13,
		})),
		grpc.WithUserAgent("t-invest"),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{PermitWithoutStream: true}),
		grpc.WithChainUnaryInterceptor(
			NewTimeoutUnaryInterceptor(60*time.Second),
			NewAppNameUnaryInterceptor("t-invest"),
		),
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		panic(err)
	}

	return &Client{
		ordersServiceClient:      NewOrdersServiceClient(conn),
		instrumentsServiceClient: NewInstrumentsServiceClient(conn, token),
		marketDataServiceClient:  NewMarketDataService(conn, token),
		conn:                     conn,
	}, nil
}

func NewAppNameUnaryInterceptor(appName string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, "x-app-name", appName)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

func NewTimeoutUnaryInterceptor(t time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, t)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
