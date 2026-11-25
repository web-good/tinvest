package inplay

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHourlyBinanceDigest(t *testing.T) {
	end := time.Unix(86400*20+3600, 0).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/exchangeInfo":
			fmt.Fprint(w, `{"symbols":[{"symbol":"ABCUSDT","quoteAsset":"USDT","status":"TRADING","isSpotTradingAllowed":true},{"symbol":"ABCBTC","quoteAsset":"BTC","status":"TRADING","isSpotTradingAllowed":true}]}`)
		case "/klines":
			q := r.URL.Query()
			if q.Get("interval") != "5m" || q.Get("symbol") != "ABCUSDT" {
				t.Errorf("unexpected request: %v", q)
			}
			start, err := strconv.ParseInt(q.Get("startTime"), 10, 64)
			if err != nil {
				t.Error(err)
				return
			}
			volume := 10
			if start == end.Add(-time.Hour).UnixMilli() {
				volume = 20
			}
			var rows []string
			stop, _ := strconv.ParseInt(q.Get("endTime"), 10, 64)
			for i := 0; start+int64(i)*step.Milliseconds() <= stop; i++ {
				high := "100"
				if i == 0 {
					high = "106"
				}
				rows = append(rows, fmt.Sprintf(`[%d,"100","%s","100","100","%d"]`, start+int64(i)*step.Milliseconds(), high, volume))
			}
			fmt.Fprintf(w, "[%s]", strings.Join(rows, ","))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var messages []string
	w := NewBinance(0.5, 2, func(s string) error { messages = append(messages, s); return nil })
	w.feed = binance{server.Client(), server.URL + "/"}
	w.delay = time.Millisecond
	if err := w.scan(context.Background(), end); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || strings.Contains(messages[0], "ABCUSDT") || !strings.Contains(messages[1], "ABCUSDT    0.50%    1/12    RVOL 2.00×") {
		t.Fatal(messages)
	}
	messages = nil
	w.minRVOL = 2.1
	if err := w.scan(context.Background(), end); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || strings.Contains(messages[1], "ABCUSDT") {
		t.Fatal(messages)
	}
}
