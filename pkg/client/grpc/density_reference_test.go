package grpc

import (
	"context"
	"testing"
	investapi "tinvest/internal/pb/v1"

	googlegrpc "google.golang.org/grpc"
)

type referenceFake struct {
	investapi.MarketDataServiceClient
}

func (referenceFake) GetLastPrices(_ context.Context, r *investapi.GetLastPricesRequest, _ ...googlegrpc.CallOption) (*investapi.GetLastPricesResponse, error) {
	return &investapi.GetLastPricesResponse{LastPrices: []*investapi.LastPrice{{InstrumentUid: r.InstrumentId[0], Price: &investapi.Quotation{Units: 100}}}}, nil
}
func TestDensityReferencePricesAreReadOnly(t *testing.T) {
	m := &DensityMarketData{market: referenceFake{}, auth: NewAuth("test")}
	p, err := m.ReferencePrices(context.Background(), []string{"uid"})
	if err != nil || p["uid"].Units != 100 {
		t.Fatalf("%v %v", p, err)
	}
}
