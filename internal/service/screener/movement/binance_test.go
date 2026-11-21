package movement

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHourlyBinanceDigest(t *testing.T) {
	end := time.Unix(3600, 0).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/exchangeInfo":
			fmt.Fprint(w, `{"symbols":[{"symbol":"ABCUSDT","quoteAsset":"USDT","status":"TRADING","isSpotTradingAllowed":true},{"symbol":"ABCBTC","quoteAsset":"BTC","status":"TRADING","isSpotTradingAllowed":true}]}`)
		case "/klines":
			q := r.URL.Query()
			if q.Get("interval") != "5m" || q.Get("symbol") != "ABCUSDT" || q.Get("startTime") != "0" || q.Get("endTime") != "3599999" {
				t.Errorf("unexpected request: %v", q)
			}
			var rows []string
			for i := 0; i < 12; i++ {
				high := "100"
				if i == 0 {
					high = "106"
				}
				rows = append(rows, fmt.Sprintf(`[%d,"100","%s","100"]`, int64(i)*step.Milliseconds(), high))
			}
			fmt.Fprintf(w, "[%s]", strings.Join(rows, ","))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var messages []string
	w := NewBinance(0.5, func(s string) error { messages = append(messages, s); return nil })
	w.feed = binance{server.Client(), server.URL + "/"}
	w.delay = time.Millisecond
	if err := w.scan(context.Background(), end); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || strings.Contains(messages[0], "ABCUSDT") || !strings.Contains(messages[1], "ABCUSDT    0.50%    1/12") {
		t.Fatal(messages)
	}
}
