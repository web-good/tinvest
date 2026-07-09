package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

// fakeStopOrdersAPI implements investapi.StopOrdersServiceClient by embedding
// the interface (so only PostStopOrder is defined) and records the call
// options it receives, to prove the auth credential was attached.
type fakeStopOrdersAPI struct {
	investapi.StopOrdersServiceClient
	gotOpts []grpc.CallOption
}

func (f *fakeStopOrdersAPI) PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error) {
	f.gotOpts = opts
	return &investapi.PostStopOrderResponse{}, nil
}

func TestNewStopOrdersServiceClient_StoresToken(t *testing.T) {
	c := NewStopOrdersServiceClient(nil, "tok-xyz").(*stopOrdersServiceClient)
	if c.auth == nil || c.auth.token != "tok-xyz" {
		t.Fatalf("auth = %+v, want token tok-xyz", c.auth)
	}
}

func TestStopOrdersServiceClient_PostStopOrderAttachesAuth(t *testing.T) {
	fake := &fakeStopOrdersAPI{}
	c := &stopOrdersServiceClient{api: fake, auth: NewAuth("tok-123")}

	if _, err := c.PostStopOrder(context.Background(), &investapi.PostStopOrderRequest{}); err != nil {
		t.Fatalf("PostStopOrder returned error: %v", err)
	}
	if len(fake.gotOpts) != 1 {
		t.Fatalf("PostStopOrder passed %d call options, want 1 (auth credential)", len(fake.gotOpts))
	}
}
