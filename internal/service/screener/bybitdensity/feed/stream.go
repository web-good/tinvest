package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	heartbeatInterval = 20 * time.Second
	streamIdleTimeout = 45 * time.Second
	spotTopicLimit    = 10
	topicBytesLimit   = 21_000
)

type subscription struct {
	ID   string   `json:"req_id"`
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

type streamRoute struct {
	symbol *streamSymbol
	kind   string
}

type streamSymbol struct {
	key        model.Key
	tick       decimal
	book       bookState
	tradeReady bool
}

type streamMessage struct {
	ID      string          `json:"req_id"`
	Op      string          `json:"op"`
	Topic   string          `json:"topic"`
	Type    string          `json:"type"`
	Text    string          `json:"ret_msg"`
	Success *bool           `json:"success"`
	TS      millis          `json:"ts"`
	CTS     millis          `json:"cts"`
	Data    json.RawMessage `json:"data"`
}

type millis int64

func (m *millis) UnmarshalJSON(raw []byte) error {
	text := string(raw)
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp %q: %w", text, err)
	}
	*m = millis(value)

	return nil
}

// Stream subscribes to books, public trades and M1 candles on one market connection.
// It returns when the connection or event consumer fails, or the context is cancelled.
func (c *Client) Stream(ctx context.Context, instruments []model.Instrument, depth int, emit func(model.Event) error) error {
	if len(instruments) == 0 || !supportedMarket(instruments[0].Key.Market) {
		return errors.New("stream requires instruments from a supported Bybit market")
	}
	if depth != 1 && depth != 50 && depth != 200 && depth != 1000 {
		return fmt.Errorf("unsupported Bybit stream depth %d", depth)
	}
	if emit == nil {
		return errors.New("stream requires a Bybit event consumer")
	}

	market := instruments[0].Key.Market
	routes, batches, err := subscriptions(instruments, depth, market)
	if err != nil {
		return err
	}
	conn, _, err := websocket.Dial(ctx, c.wsURL+"/v5/public/"+string(market), &websocket.DialOptions{HTTPClient: c.http}) //nolint:bodyclose // The dialer owns the handshake response body.
	if err != nil {
		return fmt.Errorf("connect to Bybit %s: %w", market, err)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(2 << 20)

	streamCtx, cancel := context.WithCancelCause(ctx)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		heartbeat(streamCtx, conn, cancel)
	}()
	defer func() {
		cancel(nil)
		<-heartbeatDone
	}()

	for _, batch := range batches {
		subCtx, done := context.WithTimeout(streamCtx, requestTimeout)
		err := subscribe(subCtx, conn, batch, routes, depth, emit)
		done()
		if err != nil {
			return streamError(streamCtx, market, err)
		}
	}
	for {
		readCtx, done := context.WithTimeout(streamCtx, streamIdleTimeout)
		_, err := readStream(readCtx, conn, routes, depth, emit)
		done()
		if err != nil {
			return streamError(streamCtx, market, err)
		}
	}
}

func streamError(ctx context.Context, market model.Market, err error) error {
	if cause := context.Cause(ctx); cause != nil {
		err = cause
	}

	return fmt.Errorf("stream from Bybit %s: %w", market, err)
}

func subscriptions(instruments []model.Instrument, depth int, market model.Market) (map[string]streamRoute, []subscription, error) {
	routes := make(map[string]streamRoute, len(instruments)*3)
	var args []string
	bookPrefix := "orderbook." + strconv.Itoa(depth) + "."
	for _, instrument := range instruments {
		if instrument.Key.Market != market || !validTopicSymbol(instrument.Key.Symbol) {
			return nil, nil, fmt.Errorf("invalid Bybit stream instrument %+v", instrument.Key)
		}
		tickText := instrument.TickSizeText
		if tickText == "" {
			tickText = strconv.FormatFloat(instrument.TickSize, 'f', -1, 64)
		}
		tick, err := parseDecimal(tickText)
		if err != nil || tick.mantissa == 0 {
			return nil, nil, fmt.Errorf("invalid Bybit %s tick size %q", instrument.Key.Symbol, tickText)
		}

		symbol := &streamSymbol{key: instrument.Key, tick: tick}
		topics := [...]struct {
			prefix string
			kind   string
		}{
			{prefix: bookPrefix, kind: "orderbook"},
			{prefix: "publicTrade.", kind: "publicTrade"},
			{prefix: "kline.1.", kind: "kline"},
		}
		for _, item := range topics {
			topic := item.prefix + instrument.Key.Symbol
			if _, exists := routes[topic]; exists {
				return nil, nil, fmt.Errorf("duplicate Bybit stream topic %q", topic)
			}
			routes[topic] = streamRoute{symbol: symbol, kind: item.kind}
			args = append(args, topic)
		}
	}

	encoded, err := json.Marshal(args)
	if err != nil {
		return nil, nil, fmt.Errorf("encode Bybit stream topics: %w", err)
	}
	if len(encoded) > topicBytesLimit {
		return nil, nil, fmt.Errorf("market topics exceed Bybit limit of %d encoded bytes per connection", topicBytesLimit)
	}

	var batches []subscription
	for len(args) > 0 {
		count := len(args)
		if market == model.Spot {
			count = min(count, spotTopicLimit)
		}
		batches = append(batches, subscription{
			ID: strconv.Itoa(len(batches) + 1), Op: "subscribe", Args: args[:count],
		})
		args = args[count:]
	}

	return routes, batches, nil
}

func subscribe(ctx context.Context, conn *websocket.Conn, batch subscription, routes map[string]streamRoute, depth int, emit func(model.Event) error) error {
	if err := wsjson.Write(ctx, conn, batch); err != nil {
		return fmt.Errorf("write subscribe request: %w", err)
	}
	for {
		message, err := readStream(ctx, conn, routes, depth, emit)
		if err != nil {
			return err
		}
		if message.Op != "subscribe" {
			continue
		}
		if message.ID != "" && message.ID != batch.ID {
			return fmt.Errorf("subscribe returned unexpected request ID %q", message.ID)
		}
		if message.Success == nil || !*message.Success {
			return fmt.Errorf("subscription rejected: %s", message.Text)
		}

		for _, topic := range batch.Args {
			route := routes[topic]
			if route.kind == "publicTrade" && !route.symbol.tradeReady {
				if err := emit(model.Event{Key: route.symbol.key, TradeReady: true}); err != nil {
					return err
				}
				route.symbol.tradeReady = true
			}
		}

		return nil
	}
}

func readStream(ctx context.Context, conn *websocket.Conn, routes map[string]streamRoute, depth int, emit func(model.Event) error) (streamMessage, error) {
	kind, frame, err := conn.Read(ctx)
	if err != nil {
		return streamMessage{}, err
	}
	received := time.Now()
	if kind != websocket.MessageText {
		return streamMessage{}, errors.New("unexpected Bybit binary frame")
	}

	var message streamMessage
	if err := json.Unmarshal(frame, &message); err != nil {
		return streamMessage{}, fmt.Errorf("decode exchange frame: %w", err)
	}
	route, subscribed := routes[message.Topic]
	if !subscribed {
		return message, nil
	}
	if err := route.apply(message, received, depth, emit); err != nil {
		return streamMessage{}, fmt.Errorf("topic %s: %w", message.Topic, err)
	}

	return message, nil
}

func heartbeat(ctx context.Context, conn *websocket.Conn, cancel context.CancelCauseFunc) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			writeCtx, done := context.WithTimeout(ctx, requestTimeout)
			err := conn.Write(writeCtx, websocket.MessageText, []byte(`{"op":"ping"}`))
			done()
			if err != nil {
				cancel(fmt.Errorf("write Bybit heartbeat: %w", err))

				return
			}
		}
	}
}

func (r streamRoute) apply(message streamMessage, received time.Time, depth int, emit func(model.Event) error) error {
	event := model.Event{Key: r.symbol.key}
	switch r.kind {
	case "orderbook":
		book, err := r.symbol.book.apply(r.symbol.key.Symbol, r.symbol.tick, depth, message, received)
		if err != nil || book == nil {
			return err
		}
		event.Book = book
	case "publicTrade":
		trades, err := r.symbol.trades(message.Data)
		if err != nil || len(trades) == 0 {
			return err
		}
		event.Trades = trades
		event.TradeReady = !r.symbol.tradeReady
		r.symbol.tradeReady = true
	case "kline":
		candles, err := r.symbol.candles(message.Data)
		if err != nil {
			return err
		}
		for i := range candles {
			event.Candle = &candles[i]
			if err := emit(event); err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("unsupported Bybit stream topic %q", r.kind)
	}

	return emit(event)
}

func (s *streamSymbol) trades(data json.RawMessage) ([]model.Trade, error) {
	var records []struct {
		ID     string `json:"i"`
		Symbol string `json:"s"`
		Side   string `json:"S"` // Bybit distinguishes symbol and taker side by case.
		Price  string `json:"p"`
		Qty    string `json:"v"`
		At     millis `json:"T"`
	}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode trades: %w", err)
	}
	trades := make([]model.Trade, 0, len(records))
	for _, record := range records {
		if record.Symbol != s.key.Symbol || record.ID == "" || record.At <= 0 {
			return nil, errors.New("invalid public trade identity or timestamp")
		}
		if record.Side != "Buy" && record.Side != "Sell" {
			return nil, fmt.Errorf("invalid public trade taker side %q", record.Side)
		}
		ticks, err := s.tick.ticks(record.Price)
		if err != nil {
			return nil, err
		}
		qty, err := nonnegative(record.Qty)
		if err != nil || qty <= 0 {
			return nil, fmt.Errorf("invalid trade quantity %q", record.Qty)
		}
		trades = append(trades, model.Trade{ID: record.ID, Ticks: ticks, Qty: qty, At: time.UnixMilli(int64(record.At))})
	}

	return trades, nil
}

func (s *streamSymbol) candles(data json.RawMessage) ([]model.Candle, error) {
	var records []struct {
		Start    millis `json:"start"`
		End      millis `json:"end"`
		Interval string `json:"interval"`
		Open     string `json:"open"`
		High     string `json:"high"`
		Low      string `json:"low"`
		Close    string `json:"close"`
		Confirm  bool   `json:"confirm"`
	}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode candles: %w", err)
	}
	var candles []model.Candle
	for _, record := range records {
		if !record.Confirm {
			continue
		}
		const minute = millis(time.Minute / time.Millisecond)
		if record.Interval != "1" || record.Start <= 0 || record.Start%minute != 0 || record.End-record.Start != minute-1 {
			return nil, errors.New("invalid exchange M1 candle interval")
		}

		prices := [...]string{record.Open, record.High, record.Low, record.Close}
		var ticks [len(prices)]int64
		for i, price := range prices {
			value, err := s.tick.ticks(price)
			if err != nil {
				return nil, err
			}
			ticks[i] = value
		}
		open, high, low, closeTicks := ticks[0], ticks[1], ticks[2], ticks[3]
		if high < max(open, closeTicks) || low > min(open, closeTicks) {
			return nil, errors.New("invalid exchange M1 candle price range")
		}
		candles = append(candles, model.Candle{
			Start: time.UnixMilli(int64(record.Start)),
			End:   time.UnixMilli(int64(record.End) + 1),
			Open:  open,
			High:  high,
			Low:   low,
			Close: closeTicks,
		})
	}

	return candles, nil
}
