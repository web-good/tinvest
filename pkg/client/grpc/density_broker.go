package grpc

import (
	"errors"
	"fmt"
	"math"
	"strings"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/live"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DensityBroker rejects all mutations in check-only mode, before making an RPC.
type DensityBroker struct {
	orders      investapi.OrdersServiceClient
	stops       investapi.StopOrdersServiceClient
	users       investapi.UsersServiceClient
	operations  investapi.OperationsServiceClient
	auth        *Auth
	account     string
	trade       bool
	instruments map[string]density.Instrument
}

func NewDensityBroker(conn grpc.ClientConnInterface, token, account string, trade bool, instruments []density.Instrument) *DensityBroker {
	b := &DensityBroker{orders: investapi.NewOrdersServiceClient(conn), stops: investapi.NewStopOrdersServiceClient(conn), users: investapi.NewUsersServiceClient(conn), operations: investapi.NewOperationsServiceClient(conn), auth: NewAuth(token), account: account, trade: trade, instruments: map[string]density.Instrument{}}
	for _, i := range instruments {
		b.instruments[i.ID] = i
	}
	return b
}

// Never expose broker descriptions: they can contain account IDs.
func apiCode(err error) string {
	text := status.Convert(err).Message()
	if len(text) < 5 {
		return ""
	}
	code := text[:5]
	for _, c := range code {
		if c < '0' || c > '9' {
			return ""
		}
	}
	if len(text) > 5 && text[5] != ':' && text[5] != ' ' {
		return ""
	}
	return code
}
func brokerError(err error) error {
	if err == nil {
		return nil
	}
	code := apiCode(err)
	safe := fmt.Errorf("broker RPC failed: grpc=%s api=%s", status.Code(err), code)
	definite := status.Code(err) == codes.PermissionDenied || status.Code(err) == codes.Unauthenticated
	if status.Code(err) == codes.InvalidArgument && code != "" && code != "30057" && code != "30058" && code != "30049" && code != "30053" {
		definite = true
	}
	if definite {
		return errors.Join(live.ErrRejected, safe)
	}
	return safe
}
func quote(n int64) *investapi.Quotation {
	return &investapi.Quotation{Units: n / live.Nano, Nano: int32(n % live.Nano)}
}
func moneyNano(m *investapi.MoneyValue) (int64, error) {
	if m == nil {
		return 0, nil
	}
	if !strings.EqualFold(m.Currency, "rub") || m.Units > math.MaxInt64/live.Nano || m.Units < math.MinInt64/live.Nano || m.Nano <= -int32(live.Nano) || m.Nano >= int32(live.Nano) {
		return 0, fmt.Errorf("invalid non-RUB or overflowing broker money")
	}
	base := m.Units * live.Nano
	n := int64(m.Nano)
	if n > 0 && base > math.MaxInt64-n || n < 0 && base < math.MinInt64-n {
		return 0, fmt.Errorf("broker money overflow")
	}
	return base + n, nil
}
func (b *DensityBroker) validate(id, key string, lots, price int64, priced bool) error {
	i, ok := b.instruments[id]
	if !ok || i.Lot <= 0 || live.Price(i.PriceStep) <= 0 || lots != 1 {
		return fmt.Errorf("density-live only permits one-lot orders in selected instruments")
	}
	if _, err := uuid.Parse(key); err != nil {
		return fmt.Errorf("persistent request UUID is required")
	}
	if priced && (price <= 0 || price%live.Price(i.PriceStep) != 0) {
		return fmt.Errorf("price is not on instrument tick")
	}
	return nil
}

var _ live.Broker = (*DensityBroker)(nil)
