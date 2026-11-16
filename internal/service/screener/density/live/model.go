package live

import (
	"context"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

type Order struct {
	Key, ID, Kind, InstrumentID     string
	Lots, Filled, Amount, Fees      int64 // quantities in lots; money in nanorubles
	Sent, Terminal, CancelRequested bool
	Status                          string
	FeesKnown                       bool
}

type Stop struct {
	Key, ID, Kind, Status, ChildID string
	Sent, CancelRequested          bool
	Child                          *Order
}

type Episode struct {
	Instrument                                             density.Instrument
	Density, DensityLots, EntryPrice, StopPrice, TakePrice int64
	Created, Opened                                        time.Time
	Entry                                                  Order
	Stop, Take                                             Stop
	Exit                                                   *Order
	Reason, Blocked                                        string
}

func (e *Episode) Remaining() int64 {
	sold := int64(0)
	for _, s := range []*Stop{&e.Stop, &e.Take} {
		if s.Child != nil {
			sold += s.Child.Filled
		}
	}
	if e.Exit != nil {
		sold += e.Exit.Filled
	}
	return e.Entry.Filled - sold
}

type State struct {
	SharedRoot string
	// Episode and ReleasePending are read only when migrating a v1 journal.
	Episode        *Episode            `json:",omitempty"`
	ReleasePending *accountcoord.Claim `json:",omitempty"`
	Episodes       map[string]*Episode
	Releases       map[string]accountcoord.Claim
	Notifications  []Notification
	// Ownership retains order identities after closure for cross-strategy recovery.
	Ownership        map[string]accountcoord.DensityPosition
	OwnershipKeys    map[string]string
	Version          int
	AccountHash      string
	Cooldown         map[string]time.Time
	Closed, Canceled int
	Realized, Fees   int64
	PnLComplete      bool
	LastEvent        string
	Updated          time.Time
}

type OrderRequest struct {
	Key, InstrumentID, Kind string // buy=limit BUY, exit=market SELL
	Lots, Price             int64
}
type StopRequest struct {
	Key, InstrumentID, Kind string // stop=stop-market SELL; take=conditional limit SELL
	Lots, Price             int64
}
type BrokerStop struct {
	ID, Status, ChildID, InstrumentID string
	Activated                         bool
}
type Holding struct{ Units int64 }
type ActiveOrder struct{ ID, InstrumentID string }
type Account struct {
	Holdings map[string]Holding
	Orders   []ActiveOrder
	Stops    map[string]string // active stop ID -> instrument UID
}
type Broker interface {
	CheckAccount(context.Context) error
	Account(context.Context) (Account, error)
	CanBuy(context.Context, string, int64, int64) (bool, error)
	PostOrder(context.Context, OrderRequest) (Order, error)
	Order(context.Context, string, bool) (Order, error) // bool selects request UUID vs exchange ID
	CancelOrder(context.Context, string, bool) error
	PostStop(context.Context, StopRequest) (string, error)
	Stops(context.Context, time.Time) ([]BrokerStop, error)
	CancelStop(context.Context, string) error
}
