package live

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"tinvest/internal/service/screener/density"
)

var errInventoryBusy = errors.New("instrument inventory requires reconciliation")

// reconcileInventory accepts only our confirmed remainder plus a known candle
// slice. A shared ticker is not necessarily flat when a density episode closes.
func (e *Engine) reconcileInventory(a Account, i density.Instrument, ownLots int64) error {
	foreignUnits := int64(0)
	var stops, orders []string
	if e.coordinator != nil {
		state, err := e.coordinator.LegacyState()
		if err != nil {
			return err
		}
		if entry, ok := state[i.Ticker]; ok {
			if entry.Quantity < 0 {
				return fmt.Errorf("invalid candle-owned inventory")
			}
			foreignUnits = entry.Quantity
			stops = []string{entry.StopOrderID, entry.TakeProfitOrderID}
			orders = []string{entry.EntryOrderID, entry.FVGExitOrderID, entry.EntryRequestID, entry.ExitRequestID}
		}
	}
	if a.Holdings[i.ID].Units != ownLots*i.Lot+foreignUnits {
		return fmt.Errorf("%w: broker quantities do not match known owners", errInventoryBusy)
	}
	for _, order := range a.Orders {
		if order.InstrumentID == i.ID && !slices.Contains(orders, order.ID) {
			return fmt.Errorf("%w: unknown exchange order", errInventoryBusy)
		}
	}
	for id, uid := range a.Stops {
		if uid == i.ID && !slices.Contains(stops, id) {
			return fmt.Errorf("%w: unknown stop", errInventoryBusy)
		}
	}
	return nil
}

func (e *Engine) pendingCandleEntry(ctx context.Context, ticker string) (bool, error) {
	if e.coordinator == nil {
		return false, nil
	}
	claims, err := e.coordinator.CandleClaims(ctx)
	if err != nil {
		return false, err
	}
	if claim, ok := claims[ticker]; ok && claim.Status == "active" {
		state, err := e.coordinator.LegacyState()
		if err != nil {
			return false, err
		}
		_, known := state[ticker]
		return !known, nil
	}
	return false, nil
}

// Before cancelling protection, check that unrelated inventory is consistent.
// This keeps the existing stop working while another owner's fills settle.
func (e *Engine) checkExitInventory(ctx context.Context) error {
	ep := e.episode
	a, err := e.broker.Account(ctx)
	if err != nil {
		return err
	}
	stops := make(map[string]string, len(a.Stops))
	for id, uid := range a.Stops {
		if id != ep.Stop.ID && id != ep.Take.ID {
			stops[id] = uid
		}
	}
	a.Stops = stops
	orders := make([]ActiveOrder, 0, len(a.Orders))
	for _, order := range a.Orders {
		own := false
		for _, child := range []*Order{ep.Stop.Child, ep.Take.Child} {
			own = own || child != nil && child.ID == order.ID
		}
		if !own {
			orders = append(orders, order)
		}
	}
	a.Orders = orders
	return e.reconcileInventory(a, ep.Instrument, ep.Remaining())
}
