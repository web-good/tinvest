package grpc

import (
	"context"
	"fmt"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/live"
)

// InstrumentByID resolves owned inventory even if it is no longer eligible to buy.
func (d *DensityMarketData) InstrumentByID(parent context.Context, id string) (density.Instrument, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := d.instruments.ShareBy(ctx, &investapi.InstrumentRequest{IdType: investapi.InstrumentIdType_INSTRUMENT_ID_TYPE_UID, Id: id}, NewRPCCredential(d.auth))
	if err != nil {
		return density.Instrument{}, brokerError(err)
	}
	share := resp.GetInstrument()
	i := densityShare(share).Instrument
	if share == nil || i.ID != id || i.Lot <= 0 || live.Price(i.PriceStep) <= 0 {
		return density.Instrument{}, fmt.Errorf("missing recovery instrument metadata")
	}
	return i, nil
}

// ReferencePrices are last-trade prices for preflight cost estimates only.
// They are never used to authorize a live entry or to simulate an execution.
func (d *DensityMarketData) ReferencePrices(parent context.Context, ids []string) (map[string]density.Price, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := d.market.GetLastPrices(ctx, &investapi.GetLastPricesRequest{InstrumentId: ids}, NewRPCCredential(d.auth))
	if err != nil {
		return nil, brokerError(err)
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	result := map[string]density.Price{}
	for _, p := range resp.GetLastPrices() {
		id := p.GetInstrumentUid()
		price := density.Price{Units: p.GetPrice().GetUnits(), Nano: p.GetPrice().GetNano()}
		if !selected[id] || live.Price(price) <= 0 {
			return nil, fmt.Errorf("invalid reference price response")
		}
		result[id] = price
	}
	return result, nil
}
