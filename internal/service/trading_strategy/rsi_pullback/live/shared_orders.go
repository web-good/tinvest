package live

import (
	"context"
	"fmt"
	imodel "tinvest/internal/model"
	pb "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

// sharedOrderBlocks attributes active exchange orders using durable owner IDs.
// Unattributed orders block their instrument without being cancelled.
func (s *service) sharedOrderBlocks(ctx context.Context, shares map[string]*imodel.Share, state map[string]statestore.Entry, density map[string]accountcoord.DensityPosition) (map[string]error, error) {
	type owner struct {
		uid string
		dir pb.OrderDirection
	}
	known := map[string]owner{}
	add := func(id, uid string, dir pb.OrderDirection) {
		if id != "" {
			known[id] = owner{uid, dir}
		}
	}
	for ticker, e := range state {
		if sh := shares[ticker]; sh != nil {
			add(e.EntryOrderID, sh.ID, pb.OrderDirection_ORDER_DIRECTION_BUY)
			add(e.EntryRequestID, sh.ID, pb.OrderDirection_ORDER_DIRECTION_BUY)
			add(e.FVGExitOrderID, sh.ID, pb.OrderDirection_ORDER_DIRECTION_SELL)
			add(e.ExitRequestID, sh.ID, pb.OrderDirection_ORDER_DIRECTION_SELL)
		}
	}
	for _, d := range density {
		for _, id := range d.OrderIDs {
			add(id, d.InstrumentID, pb.OrderDirection_ORDER_DIRECTION_UNSPECIFIED)
		}
	}
	orders, err := s.slots[0].exec.ActiveOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared active orders unavailable: %w", err)
	}
	blocked := map[string]error{}
	for _, order := range orders {
		uid := order.GetInstrumentUid()
		if uid == "" {
			return nil, fmt.Errorf("active order has no instrument identity")
		}
		found, ok := known[order.GetOrderId()]
		if !ok {
			found, ok = known[order.GetOrderRequestId()]
		}
		if !ok {
			// A stop may trigger between the durable snapshot and this read. Resolve
			// only our saved stop IDs to exchange children before declaring it unknown.
			for ticker, e := range state {
				sh := shares[ticker]
				if sh == nil || sh.ID != uid {
					continue
				}
				sl := s.slotByName(adapter.Owner(e))
				if sl == nil {
					continue
				}
				for _, id := range []string{e.StopOrderID, e.TakeProfitOrderID, e.FVGExitStopID} {
					if id == "" || id == "market" {
						continue
					}
					child, fired, err := sl.stops.Triggered(ctx, id)
					if err != nil {
						return nil, fmt.Errorf("resolve owned stop child: %w", err)
					}
					if fired && child == order.GetOrderId() {
						found = owner{uid, pb.OrderDirection_ORDER_DIRECTION_SELL}
						ok = true
						break
					}
				}
				if ok {
					break
				}
			}
		}
		if !ok {
			blocked[uid] = fmt.Errorf("unattributed active order %s; reconciliation required", order.GetOrderId())
			continue
		}
		direction := found.dir
		if direction == pb.OrderDirection_ORDER_DIRECTION_UNSPECIFIED {
			direction = order.GetDirection()
		}
		if direction != pb.OrderDirection_ORDER_DIRECTION_BUY && direction != pb.OrderDirection_ORDER_DIRECTION_SELL {
			blocked[uid] = fmt.Errorf("active order direction invalid")
			continue
		}
		if err := validateOwnedOrder(order, found.uid, direction, 0); err != nil {
			blocked[uid] = err
			blocked[found.uid] = err
		}
	}
	return blocked, nil
}
