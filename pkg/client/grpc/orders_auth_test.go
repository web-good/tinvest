package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

// fakeOrdersAPI implements investapi.OrdersServiceClient by embedding the
// interface (so only PostOrder is defined) and records the call options it
// receives, to prove the auth credential was attached.
type fakeOrdersAPI struct {
	investapi.OrdersServiceClient
	gotOpts []grpc.CallOption
}

func (f *fakeOrdersAPI) PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	f.gotOpts = opts
	return &investapi.PostOrderResponse{}, nil
}

func TestNewOrdersServiceClient_StoresToken(t *testing.T) {
	c := NewOrdersServiceClient(nil, "tok-xyz").(*ordersServiceClient)
	if c.auth == nil || c.auth.token != "tok-xyz" {
		t.Fatalf("auth = %+v, want token tok-xyz", c.auth)
	}
}

func TestOrdersServiceClient_PostOrderAttachesAuth(t *testing.T) {
	fake := &fakeOrdersAPI{}
	c := &ordersServiceClient{orderApi: fake, auth: NewAuth("tok-123")}

	if _, err := c.PostOrder(context.Background(), &investapi.PostOrderRequest{}); err != nil {
		t.Fatalf("PostOrder returned error: %v", err)
	}
	if len(fake.gotOpts) != 1 {
		t.Fatalf("PostOrder passed %d call options, want 1 (auth credential)", len(fake.gotOpts))
	}
}
