package grpc

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/live"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *DensityBroker) CheckAccount(parent context.Context) error {
	if strings.TrimSpace(b.account) == "" {
		return fmt.Errorf("explicit account is required")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	resp, err := b.users.GetAccounts(ctx, &investapi.GetAccountsRequest{}, NewRPCCredential(b.auth))
	if err != nil {
		return brokerError(err)
	}
	found := false
	for _, a := range resp.GetAccounts() {
		if a.GetId() == b.account {
			if a.GetStatus() != investapi.AccountStatus_ACCOUNT_STATUS_OPEN || a.GetAccessLevel() != investapi.AccessLevel_ACCOUNT_ACCESS_LEVEL_FULL_ACCESS {
				return fmt.Errorf("selected account must be open with full token access")
			}
			found = true
		}
	}
	if !found {
		return fmt.Errorf("explicitly selected account is not accessible")
	}
	_, err = b.users.GetMarginAttributes(ctx, &investapi.GetMarginAttributesRequest{AccountId: b.account}, NewRPCCredential(b.auth))
	if status.Code(err) == codes.InvalidArgument && apiCode(err) == "30051" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot confirm disabled margin: %w", brokerError(err))
	}
	return fmt.Errorf("margin is enabled or its disabled state is not confirmed; real entries refused")
}

func (b *DensityBroker) Account(parent context.Context) (live.Account, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	a := live.Account{Holdings: map[string]live.Holding{}, Stops: map[string]string{}}
	positions, err := b.operations.GetPositions(ctx, &investapi.PositionsRequest{AccountId: b.account}, NewRPCCredential(b.auth))
	if err != nil {
		return a, brokerError(err)
	}
	if positions.GetLimitsLoadingInProgress() {
		return a, fmt.Errorf("account positions still loading")
	}
	if positions.GetAccountId() != "" && positions.GetAccountId() != b.account {
		return a, fmt.Errorf("broker returned another account")
	}
	for _, p := range positions.GetSecurities() {
		id := p.GetInstrumentUid()
		if id == "" && (p.GetBalance() != 0 || p.GetBlocked() != 0) {
			return a, fmt.Errorf("holding without instrument UID")
		}
		if _, exists := a.Holdings[id]; exists {
			return a, fmt.Errorf("duplicate holding UID")
		}
		free, blocked := p.GetBalance(), p.GetBlocked()
		if free < 0 || blocked < 0 || free > math.MaxInt64-blocked || p.GetExchangeBlocked() {
			return a, fmt.Errorf("negative, overflowing or exchange-blocked holdings")
		}
		a.Holdings[id] = live.Holding{Units: free + blocked}
	}
	orders, err := b.orders.GetOrders(ctx, &investapi.GetOrdersRequest{AccountId: b.account}, NewRPCCredential(b.auth))
	if err != nil {
		return a, brokerError(err)
	}
	for _, o := range orders.GetOrders() {
		if o.GetOrderId() == "" || o.GetInstrumentUid() == "" {
			return a, fmt.Errorf("active order without identifiers")
		}
		a.Orders = append(a.Orders, live.ActiveOrder{ID: o.GetOrderId(), InstrumentID: o.GetInstrumentUid()})
	}
	stops, err := b.stops.GetStopOrders(ctx, &investapi.GetStopOrdersRequest{AccountId: b.account, Status: investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE}, NewRPCCredential(b.auth))
	if err != nil {
		return a, brokerError(err)
	}
	for _, s := range stops.GetStopOrders() {
		if s.GetStopOrderId() == "" || s.GetInstrumentUid() == "" {
			return a, fmt.Errorf("active stop without identifiers")
		}
		a.Stops[s.GetStopOrderId()] = s.GetInstrumentUid()
	}
	return a, nil
}

func (b *DensityBroker) CanBuy(parent context.Context, id string, price, lots int64) (bool, error) {
	i, ok := b.instruments[id]
	if !ok || lots != 1 || price <= 0 || live.Price(i.PriceStep) <= 0 || price%live.Price(i.PriceStep) != 0 {
		return false, fmt.Errorf("invalid one-lot allocation request")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	limits, err := b.orders.GetMaxLots(ctx, &investapi.GetMaxLotsRequest{AccountId: b.account, InstrumentId: id, Price: quote(price)}, NewRPCCredential(b.auth))
	if err != nil {
		return false, brokerError(err)
	}
	if !strings.EqualFold(limits.GetCurrency(), "rub") || limits.GetBuyLimits().GetBuyMaxLots() < 1 {
		return false, nil
	}
	amount, err := b.orders.GetOrderPrice(ctx, &investapi.GetOrderPriceRequest{AccountId: b.account, InstrumentId: id, Price: quote(price), Direction: investapi.OrderDirection_ORDER_DIRECTION_BUY, Quantity: 1}, NewRPCCredential(b.auth))
	if err != nil {
		return false, brokerError(err)
	}
	total, err := moneyNano(amount.GetTotalOrderAmount())
	if err != nil {
		return false, err
	}
	available := live.Price(density.Price{Units: limits.GetBuyLimits().GetBuyMoneyAmount().GetUnits(), Nano: limits.GetBuyLimits().GetBuyMoneyAmount().GetNano()})
	if i.Lot <= 0 || price > math.MaxInt64/i.Lot || amount.GetLotsRequested() != 1 || total < price*i.Lot {
		return false, fmt.Errorf("invalid broker pre-trade amount")
	}
	return total > 0 && available >= total, nil
}
