package density

import (
	"context"
	"time"
)

type EventKind uint8

const (
	EventBook EventKind = iota + 1
	EventInvalidBook
	EventTrade
	EventTradingStatus
	EventBookAck
	EventTradeAck
	EventInfoAck
)

type Event struct {
	Kind                 EventKind
	InstrumentID         string
	Book                 Book
	Price                Price
	Quantity             int64
	Time                 time.Time
	TradingOpen, Success bool
}
type Stream interface {
	SubscribeBooks(ids []string, depth int32, subscribe bool) error
	SubscribeTrades(ids []string, subscribe bool) error
	SubscribeInfo(ids []string, subscribe bool) error
	Receive() ([]Event, error)
	Close() error
}
type StreamSource interface {
	OpenStream(context.Context, map[string]int64) (Stream, error)
}
