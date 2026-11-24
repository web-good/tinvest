package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestSpotClosesReturnsCompleteCandlesChronologically(t *testing.T) {
	from := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	const interval = 30 * time.Minute
	to := from.Add(1001 * interval)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/v5/market/kline" || q.Get("category") != "spot" || q.Get("symbol") != "SOLUSDT" || q.Get("interval") != "30" || q.Get("limit") != "1000" {
			t.Error("unexpected candle query:", r.URL)
		}
		start, _ := strconv.ParseInt(q.Get("start"), 10, 64)
		end, _ := strconv.ParseInt(q.Get("end"), 10, 64)
		rows := make([][]string, 0, 1000)
		for stamp := end + 1 - interval.Milliseconds(); stamp >= start; stamp -= interval.Milliseconds() {
			rows = append(rows, []string{strconv.FormatInt(stamp, 10), "1", "1", "1", "1", "0", "0"})
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"retCode": 0, "result": map[string]any{"category": "spot", "symbol": "SOLUSDT", "list": rows},
		}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	key := model.Key{Market: model.Spot, Symbol: "SOLUSDT"}
	closes, err := NewClient(server.URL, "").SpotCloses(context.Background(), key, interval, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(closes) != 1001 || !closes[0].End.Equal(from.Add(interval)) || !closes[1000].End.Equal(to) || closes[1000].Price != 1 {
		t.Fatalf("unexpected candle closes: %d", len(closes))
	}
}

func TestDayHighUsesAllHoursOfMoscowSunday(t *testing.T) {
	from := time.Date(2026, 11, 21, 21, 0, 0, 0, time.UTC)
	end := from.Add(24 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("interval") != "60" || q.Get("start") != strconv.FormatInt(from.UnixMilli(), 10) || q.Get("end") != strconv.FormatInt(end.UnixMilli()-1, 10) {
			t.Errorf("unexpected hourly window: %s", r.URL)
		}
		var rows [][]string
		for i := 23; i >= 0; i-- {
			high := "100"
			if i == 11 {
				high = "123.45"
			}
			rows = append(rows, []string{strconv.FormatInt(from.Add(time.Duration(i)*time.Hour).UnixMilli(), 10), "90", high, "80", "95"})
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"retCode": 0, "result": map[string]any{"category": "spot", "symbol": "SOLUSDT", "list": rows}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	got, err := NewClient(server.URL, "").DayHigh(t.Context(), model.Key{Market: model.Spot, Symbol: "SOLUSDT"}, from)
	if err != nil || got != 123.45 {
		t.Fatalf("Sunday high = %v, %v", got, err)
	}
}

func TestSpotM5UsesClosedRangeAndQuoteTurnover(t *testing.T) {
	from := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	to := from.Add(10 * time.Minute)
	key := model.Key{Market: model.Spot, Symbol: "PIEVERSEUSDT"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/v5/market/kline" || q.Get("interval") != "5" || q.Get("end") != strconv.FormatInt(to.UnixMilli()-1, 10) {
			t.Errorf("unexpected candle query: %s", r.URL)
		}
		fmt.Fprintf(w, `{"retCode":0,"result":{"category":"spot","symbol":"PIEVERSEUSDT","list":[["%d","100","105","99","102","5","506"],["%d","99","103","98","100","7","710"]]}}`, from.Add(5*time.Minute).UnixMilli(), from.UnixMilli())
	}))
	defer server.Close()
	bars, err := NewClient(server.URL, "").SpotM5(t.Context(), key, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 2 || bars[0] != (model.M5Candle{End: from.Add(5 * time.Minute), Open: 99, High: 103, Low: 98, Turnover: 710}) ||
		bars[1] != (model.M5Candle{End: to, Open: 100, High: 105, Low: 99, Turnover: 506}) {
		t.Fatalf("unexpected bars: %+v", bars)
	}
}

func TestSpotM5PreservesMissingCandle(t *testing.T) {
	from := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Minute)
	key := model.Key{Market: model.Spot, Symbol: "PIEVERSEUSDT"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"retCode":0,"result":{"category":"spot","symbol":"PIEVERSEUSDT","list":[["%d","100","105","99","102","5","506"],["%d","99","103","98","100","7","710"]]}}`, from.Add(10*time.Minute).UnixMilli(), from.UnixMilli())
	}))
	defer server.Close()
	bars, err := NewClient(server.URL, "").SpotM5(t.Context(), key, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 2 || !bars[0].End.Equal(from.Add(5*time.Minute)) || !bars[1].End.Equal(to) {
		t.Fatalf("expected two candles surrounding the gap: %+v", bars)
	}
}
