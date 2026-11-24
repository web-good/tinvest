package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestInstrumentsRanksActiveUSDTPairsAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		category := r.URL.Query().Get("category")
		if category != "linear" {
			t.Errorf("unexpected category %q", category)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		switch r.URL.Path {
		case "/v5/market/tickers":
			fmt.Fprint(w, `{"retCode":0,"result":{"category":"linear","list":[{"symbol":"AAAUSDT","turnover24h":"40.1"},{"symbol":"BBBUSDT","turnover24h":"50.2"},{"symbol":"FUTUSDT","turnover24h":"99"},{"symbol":"OTHERUSDC","turnover24h":"200"},{"symbol":"PENDUSDT","turnover24h":"300"}]}}`)
		case "/v5/market/instruments-info":
			switch r.URL.Query().Get("cursor") {
			case "":
				fmt.Fprint(w, `{"retCode":0,"result":{"category":"linear","nextPageCursor":"next+token","list":[`+instrumentJSON("AAAUSDT", "LinearPerpetual", "Trading", "USDT")+","+instrumentJSON("FUTUSDT", "LinearFutures", "Trading", "USDT")+`]}}`)
			case "next+token":
				fmt.Fprint(w, `{"retCode":0,"result":{"category":"linear","list":[`+instrumentJSON("BBBUSDT", "LinearPerpetual", "Trading", "USDT")+","+instrumentJSON("OTHERUSDC", "LinearPerpetual", "Trading", "USDC")+","+instrumentJSON("PENDUSDT", "LinearPerpetual", "PendingOpen", "USDT")+`]}}`)
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	ctx := t.Context()
	ranked, err := client.Instruments(ctx, model.Linear, nil, 1)
	require.NoError(t, err)
	require.Equal(t, []model.Instrument{{
		Key: model.Key{Market: model.Linear, Symbol: "BBBUSDT"}, TickSize: 0.05, TickSizeText: "0.0500",
		MinPrice: 0.05, MaxPrice: 1000, QtyStep: 0.001, MinQty: 0.01, MaxQty: 20, MinNotional: 5, Turnover: 50.2,
	}}, ranked)

	explicit, err := client.Instruments(ctx, model.Linear, []string{"AAAUSDT", "BBBUSDT"}, 1)
	require.NoError(t, err)
	require.Len(t, explicit, 2)
	require.Equal(t, "AAAUSDT", explicit[0].Key.Symbol)

	_, err = client.Instruments(ctx, model.Linear, []string{"FUTUSDT"}, 1)
	require.ErrorIs(t, err, ErrInvalidSymbols)
}

func TestInstrumentsUsesSpotQuantityAndAmountFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/market/tickers":
			fmt.Fprint(w, `{"retCode":0,"result":{"category":"spot","list":[{"symbol":"AAAUSDT","turnover24h":"4"}]}}`)
		case "/v5/market/instruments-info":
			if r.URL.Query().Has("limit") || r.URL.Query().Has("cursor") {
				t.Errorf("spot market request used pagination: %s", r.URL)
			}
			fmt.Fprint(w, `{"retCode":0,"result":{"category":"spot","list":[{"symbol":"AAAUSDT","status":"Trading","quoteCoin":"USDT","priceFilter":{"tickSize":"0.05"},"lotSizeFilter":{"basePrecision":"0.002","minOrderAmt":"3","maxLimitOrderQty":"25","maxMarketOrderQty":"10"}}]}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	instruments, err := client.Instruments(t.Context(), model.Spot, nil, 20)
	require.NoError(t, err)
	require.Equal(t, []model.Instrument{{
		Key: model.Key{Market: model.Spot, Symbol: "AAAUSDT"}, TickSize: 0.05, TickSizeText: "0.05",
		QtyStep: 0.002, MaxQty: 10, MinNotional: 3, Turnover: 4,
	}}, instruments)
}

func instrumentJSON(symbol, contractType, status, coin string) string {
	return fmt.Sprintf(`{"symbol":%q,"contractType":%q,"status":%q,"quoteCoin":%q,"settleCoin":%q,"priceFilter":{"tickSize":"0.0500","minPrice":"0.05","maxPrice":"1000"},"lotSizeFilter":{"qtyStep":"0.001","minOrderQty":"0.01","maxMktOrderQty":"20","minNotionalValue":"5"}}`,
		symbol, contractType, status, coin, coin)
}

type request struct {
	ID   string   `json:"req_id"`
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

func mockStream(t *testing.T, market model.Market, records []string) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/public/"+string(market) {
			t.Errorf("unexpected stream path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)

			return
		}
		defer conn.CloseNow()

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		var message request
		if err := wsjson.Read(ctx, conn, &message); err != nil {
			t.Error(err)

			return
		}
		if message.Op != "subscribe" || strings.Join(message.Args, ",") != "orderbook.50.AAAUSDT,publicTrade.AAAUSDT,kline.1.AAAUSDT" {
			t.Errorf("unexpected subscription %+v", message)
		}
		ack := map[string]any{"req_id": message.ID, "op": "subscribe", "success": true}
		if err := wsjson.Write(ctx, conn, ack); err != nil {
			t.Error(err)

			return
		}
		for _, record := range records {
			if err := conn.Write(ctx, websocket.MessageText, []byte(record)); err != nil {
				t.Error(err)

				return
			}
		}
		conn.Close(websocket.StatusNormalClosure, "")
	}))
	t.Cleanup(server.Close)

	return NewClient("", "ws"+strings.TrimPrefix(server.URL, "http"))
}

func TestStreamAppliesDeltasAndReplacesAnAuthoritativeSnapshot(t *testing.T) {
	client := mockStream(t, model.Spot, []string{
		`{"topic":"orderbook.50.AAAUSDT","type":"delta","ts":1699999999999,"data":{"s":"AAAUSDT","u":3,"b":[],"a":[]}}`,
		`{"topic":"orderbook.50.AAAUSDT","type":"snapshot","ts":"1700000000000","cts":1699999999999,"data":{"s":"AAAUSDT","u":4,"b":[["9.90","0.3"],["10.00","1.25"]],"a":[["10.05","2.0"],["10.20","1.75"]]}}`,
		`{"topic":"orderbook.50.AAAUSDT","type":"delta","ts":1700000001000,"data":{"s":"AAAUSDT","u":14,"b":[["10.00","0"],["9.95","0.75"]],"a":[["10.05","2.25"]]}}`,
		`{"topic":"orderbook.50.AAAUSDT","type":"snapshot","ts":1700000002000,"data":{"s":"AAAUSDT","u":1,"b":[["9.85","3.5"]],"a":[["10.25","4.75"]]}}`,
	})
	instrument := model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "AAAUSDT"}, TickSize: 0.05}
	var events []model.Event
	err := client.Stream(t.Context(), []model.Instrument{instrument}, 50, func(event model.Event) error {
		events = append(events, event)

		return nil
	})

	require.Equal(t, websocket.StatusNormalClosure, websocket.CloseStatus(err))
	require.Len(t, events, 4)
	require.True(t, events[0].TradeReady)
	require.Nil(t, events[0].Book)
	require.Equal(t, instrument.Key, events[0].Key)
	require.True(t, events[1].Book.Snapshot)
	require.Equal(t, []model.Level{{Ticks: 200, Qty: 1.25}, {Ticks: 198, Qty: 0.3}}, events[1].Book.Bids)
	require.Equal(t, time.UnixMilli(1699999999999), events[1].Book.At)
	require.WithinDuration(t, time.Now(), events[1].Book.Received, time.Second)
	require.False(t, events[2].Book.Snapshot)
	require.Equal(t, []model.Level{{Ticks: 199, Qty: 0.75}, {Ticks: 198, Qty: 0.3}}, events[2].Book.Bids)
	require.Equal(t, []model.Level{{Ticks: 201, Qty: 2.25}, {Ticks: 204, Qty: 1.75}}, events[2].Book.Asks)
	require.True(t, events[3].Book.Snapshot)
	require.Equal(t, []model.Level{{Ticks: 197, Qty: 3.5}}, events[3].Book.Bids)
	require.Equal(t, []model.Level{{Ticks: 205, Qty: 4.75}}, events[3].Book.Asks)
}

func TestStreamStopsOnMisalignedDeltaWithoutPublishingIt(t *testing.T) {
	client := mockStream(t, model.Linear, []string{
		`{"topic":"orderbook.50.AAAUSDT","type":"snapshot","ts":1700000000000,"data":{"s":"AAAUSDT","u":4,"b":[["10.00","1.25"]],"a":[["10.05","2.0"]]}}`,
		`{"topic":"orderbook.50.AAAUSDT","type":"delta","ts":1700000001000,"data":{"s":"AAAUSDT","u":5,"b":[["10.001","0"]],"a":[]}}`,
	})
	var events []model.Event
	instrument := model.Instrument{Key: model.Key{Market: model.Linear, Symbol: "AAAUSDT"}, TickSize: 0.05}
	err := client.Stream(t.Context(), []model.Instrument{instrument}, 50, func(event model.Event) error {
		events = append(events, event)

		return nil
	})

	require.ErrorContains(t, err, "not aligned")
	require.Len(t, events, 2)
	require.Equal(t, model.Linear, events[1].Key.Market)
	require.Equal(t, []model.Level{{Ticks: 200, Qty: 1.25}}, events[1].Book.Bids)
}

func TestStreamNormalizesTradesAndConfirmedMinutes(t *testing.T) {
	client := mockStream(t, model.Linear, []string{
		`{"topic":"publicTrade.AAAUSDT","type":"snapshot","ts":1700000001000,"data":[{"T":1700000000999,"s":"AAAUSDT","S":"Buy","v":"0.0125","p":"10.050","i":"trade-1"}]}`,
		`{"topic":"kline.1.AAAUSDT","type":"snapshot","ts":1700000001000,"data":[{"start":1699999980000,"end":1700000039999,"interval":"1","confirm":false}]}`,
		`{"topic":"kline.1.AAAUSDT","type":"snapshot","ts":1700000040000,"data":[{"start":1699999980000,"end":1700000039999,"interval":"1","open":"10.00","high":"10.05","low":"9.90","close":"9.95","confirm":true}]}`,
	})
	instrument := model.Instrument{Key: model.Key{Market: model.Linear, Symbol: "AAAUSDT"}, TickSize: 0.05}
	var events []model.Event
	err := client.Stream(t.Context(), []model.Instrument{instrument}, 50, func(event model.Event) error {
		events = append(events, event)

		return nil
	})

	require.Equal(t, websocket.StatusNormalClosure, websocket.CloseStatus(err))
	require.Len(t, events, 3)
	require.Equal(t, []model.Trade{{ID: "trade-1", Ticks: 201, Qty: 0.0125, At: time.UnixMilli(1700000000999)}}, events[1].Trades)
	require.Equal(t, &model.Candle{
		Start: time.UnixMilli(1699999980000), End: time.UnixMilli(1700000040000), Open: 200, High: 201, Low: 198, Close: 199,
	}, events[2].Candle)
}

func TestStreamPreservesCallbackError(t *testing.T) {
	client := mockStream(t, model.Linear, nil)
	instrument := model.Instrument{Key: model.Key{Market: model.Linear, Symbol: "AAAUSDT"}, TickSize: 0.05}
	callbackError := errors.New("consumer stopped")
	err := client.Stream(t.Context(), []model.Instrument{instrument}, 50, func(model.Event) error {
		return callbackError
	})

	require.ErrorIs(t, err, callbackError)
}

func TestStreamStopsWhenAnOrderbookSideIsLost(t *testing.T) {
	client := mockStream(t, model.Linear, []string{
		`{"topic":"orderbook.50.AAAUSDT","type":"snapshot","ts":1700000000000,"data":{"s":"AAAUSDT","u":4,"b":[["10.00","1.25"]],"a":[["10.05","2.0"]]}}`,
		`{"topic":"orderbook.50.AAAUSDT","type":"delta","ts":1700000001000,"data":{"s":"AAAUSDT","u":5,"b":[["10.00","0"]],"a":[]}}`,
	})
	var events []model.Event
	instrument := model.Instrument{Key: model.Key{Market: model.Linear, Symbol: "AAAUSDT"}, TickSize: 0.05}
	err := client.Stream(t.Context(), []model.Instrument{instrument}, 50, func(event model.Event) error {
		events = append(events, event)

		return nil
	})

	require.ErrorContains(t, err, "one-sided orderbook")
	require.Len(t, events, 2)
	require.Equal(t, []model.Level{{Ticks: 200, Qty: 1.25}}, events[1].Book.Bids)
}

func TestLinearMetadataRejectsInvalidPriceAndQuantityBounds(t *testing.T) {
	for _, field := range []string{"minPrice", "maxPrice", "minOrderQty"} {
		t.Run(field, func(t *testing.T) {
			var info instrumentInfo
			require.NoError(t, json.Unmarshal([]byte(instrumentJSON("AAAUSDT", "LinearPerpetual", "Trading", "USDT")), &info))
			switch field {
			case "minPrice":
				info.Price.Min = ""
			case "maxPrice":
				info.Price.Max = "malformed"
			case "minOrderQty":
				info.Lot.Min = "-1"
			}

			_, err := (candidate{info: info}).instrument(model.Linear)
			require.Error(t, err)
		})
	}
}

func TestTickConversionKeepsConsecutivePricesAboveFloatPrecision(t *testing.T) {
	tick, err := parseDecimal("0.01")
	require.NoError(t, err)

	lower, err := tick.ticks("90071992547409.92")
	require.NoError(t, err)
	higher, err := tick.ticks("90071992547409.93")
	require.NoError(t, err)

	require.EqualValues(t, 9007199254740992, lower)
	require.EqualValues(t, 9007199254740993, higher)
}

func TestSpotSubscriptionsRespectBybitTenTopicRequestLimit(t *testing.T) {
	var instruments []model.Instrument
	for i := range 20 {
		instruments = append(instruments, model.Instrument{
			Key: model.Key{Market: model.Spot, Symbol: fmt.Sprintf("TOKEN%02dUSDT", i)}, TickSize: 0.01,
		})
	}
	routes, batches, err := subscriptions(instruments, 200, model.Spot)
	require.NoError(t, err)
	require.Len(t, routes, 60)
	require.Len(t, batches, 6)
	for _, batch := range batches {
		require.Len(t, batch.Args, 10)
	}
}

func TestAverageVolumeUsesSpotBaseQuantityAndSession(t *testing.T) {
	from := time.Date(2026, 11, 18, 6, 0, 0, 0, time.UTC)
	to := from.Add(9 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/v5/market/kline" || query.Get("category") != "spot" || query.Get("symbol") != "BTCUSDT" ||
			query.Get("interval") != "5" || query.Get("limit") != "108" ||
			query.Get("start") != fmt.Sprint(from.UnixMilli()) || query.Get("end") != fmt.Sprint(to.UnixMilli()-1) {
			t.Errorf("unexpected request %s", r.URL)
		}
		fmt.Fprintf(w, `{"retCode":0,"result":{"symbol":"BTCUSDT","category":"spot","list":[
			["%d","1","1","1","1","999","999"],
			["%d","1","1","1","1","3.5","350"],
			["%d","1","1","1","1","1.5","150"]
		]}}`, to.UnixMilli(), from.Add(5*time.Minute).UnixMilli(), from.UnixMilli())
	}))
	defer server.Close()

	got, err := NewClient(server.URL, "").AverageVolume(t.Context(), model.Key{Market: model.Spot, Symbol: "BTCUSDT"}, from, to)
	require.NoError(t, err)
	require.Equal(t, 2.5, got)
}
