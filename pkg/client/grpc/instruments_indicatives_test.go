package grpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

// fakeInstrumentsAPI встраивает интерфейс: определён только Indicatives.
type fakeInstrumentsAPI struct {
	investapi.InstrumentsServiceClient
	resp    *investapi.IndicativesResponse
	err     error
	gotOpts []grpc.CallOption
}

func (f *fakeInstrumentsAPI) Indicatives(_ context.Context, _ *investapi.IndicativesRequest, opts ...grpc.CallOption) (*investapi.IndicativesResponse, error) {
	f.gotOpts = opts
	return f.resp, f.err
}

func TestIndicativeUIDByTicker(t *testing.T) {
	list := &investapi.IndicativesResponse{Instruments: []*investapi.IndicativeResponse{
		{Ticker: "RTSI", Uid: "uid-rts"},
		{Ticker: "IMOEX", Uid: "uid-imoex"},
	}}
	t.Run("найден", func(t *testing.T) {
		fake := &fakeInstrumentsAPI{resp: list}
		c := &instrumentsServiceClient{instrumentsAPI: fake, auth: NewAuth("tok")}
		uid, err := c.IndicativeUIDByTicker(context.Background(), "IMOEX")
		if err != nil || uid != "uid-imoex" {
			t.Fatalf("= %q, %v; want uid-imoex, nil", uid, err)
		}
		if len(fake.gotOpts) != 1 {
			t.Fatalf("call options = %d, want 1 (auth credential)", len(fake.gotOpts))
		}
	})
	t.Run("не найден", func(t *testing.T) {
		c := &instrumentsServiceClient{instrumentsAPI: &fakeInstrumentsAPI{resp: list}, auth: NewAuth("tok")}
		if _, err := c.IndicativeUIDByTicker(context.Background(), "NOPE"); err == nil || !strings.Contains(err.Error(), "NOPE") {
			t.Fatalf("err = %v, want ошибку с тикером", err)
		}
	})
	t.Run("RPC упал", func(t *testing.T) {
		c := &instrumentsServiceClient{instrumentsAPI: &fakeInstrumentsAPI{err: errors.New("unavailable")}, auth: NewAuth("tok")}
		if _, err := c.IndicativeUIDByTicker(context.Background(), "IMOEX"); err == nil {
			t.Fatal("err = nil, want ошибку RPC")
		}
	})
}
