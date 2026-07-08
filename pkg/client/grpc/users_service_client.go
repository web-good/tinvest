package grpc

import (
	"context"
	"fmt"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/converter"
	"tinvest/pkg/client/grpc/model"

	"google.golang.org/grpc"
)

type UsersServiceClient interface {
	GetAccounts(ctx context.Context) ([]model.Account, error)
}

type usersServiceClient struct {
	usersApi investapi.UsersServiceClient
	auth     *Auth
}

func New(conn grpc.ClientConnInterface, token string) UsersServiceClient {
	return &usersServiceClient{
		usersApi: investapi.NewUsersServiceClient(conn),
		auth:     NewAuth(token),
	}
}

func (u *usersServiceClient) GetAccounts(ctx context.Context) ([]model.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	status := investapi.AccountStatus_ACCOUNT_STATUS_OPEN

	resp, err := u.usersApi.GetAccounts(ctx, &investapi.GetAccountsRequest{
		Status: &status,
	}, NewRPCCredential(u.auth))

	if err != nil {
		return nil, fmt.Errorf("failed to request account: %w", err)
	}

	return converter.ConvertAccountsFromBp(resp), nil
}
