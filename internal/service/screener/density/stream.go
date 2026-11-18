package density

import (
	"context"
	"fmt"
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
	EventCandle
	EventCandleAck
)

type Event struct {
	Kind                 EventKind
	InstrumentID         string
	Book                 Book
	Candle               Candle
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

// SubscribeClosedCandles requests exchange M1 bars after the minute ends.
func SubscribeClosedCandles(stream Stream, ids []string) error {
	source, ok := stream.(interface{ SubscribeClosedCandles([]string) error })
	if !ok {
		return fmt.Errorf("market stream does not support closed minute candles")
	}
	return source.SubscribeClosedCandles(ids)
}
