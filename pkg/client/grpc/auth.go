package grpc

import (
	"context"

	"google.golang.org/grpc"
)

type Auth struct {
	token string
}

func NewAuth(token string) *Auth {
	return &Auth{token: token}
}

func NewRPCCredential(auth *Auth) grpc.CallOption {
	return grpc.PerRPCCredentials(auth)
}

func (a Auth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + a.token,
	}, nil
}

func (a Auth) RequireTransportSecurity() bool {
	return true
}
