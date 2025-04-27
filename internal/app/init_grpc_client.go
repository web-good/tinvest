package app

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpcpkg "tinvest/pkg/client/grpc"
	"tinvest/pkg/closer"
	"tinvest/pkg/logger"
)

const (
	address = "invest-public-api.tinkoff.ru:443"
)

func (a *App) initGrpcClient(ctx context.Context) error {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return fmt.Errorf("failed connect to Grpc server: %w", err)
	}

	closer.Add(func() (err error) {
		err = conn.Close()

		if err != nil {
			return fmt.Errorf("failed close connect to Grpc server: %w", err)
		}

		return
	})

	a.grpcClient = grpcpkg.NewClientGrpc(conn)
	logger.InfoContext(ctx, "Initialize gRPC client")

	return nil
}
