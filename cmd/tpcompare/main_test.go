package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	domain "tinvest/internal/domain/backtest"
	live "tinvest/internal/service/trading_strategy/rsi_pullback/live"
	core "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func TestLoadWindow(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	candles := []domain.Candle{{Time: start.Add(-time.Hour), Open: 1, High: 1, Low: 1, Close: 1}, {Time: start, Open: 1, High: 1, Low: 1, Close: 1}, {Time: end, Open: 1, High: 1, Low: 1, Close: 1}}
	raw, _ := json.Marshal(candles)
	if err := os.WriteFile(filepath.Join(dir, "TEST.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := loadWindow(filepath.Join(dir, "TEST.json"), start, end)
	if err != nil || len(got) != 1 || !got[0].Time.Equal(start) {
		t.Fatalf("window=%+v err=%v", got, err)
	}
}

func TestPrepareRejectsUnknownTickerAndMissingDaily(t *testing.T) {
	now := time.Now()
	if _, err := prepare(t.TempDir(), "UNKNOWN", now.AddDate(-2, 0, 0), now); err == nil {
		t.Fatal("unknown ticker accepted")
	}
	dir := t.TempDir()
	c := []domain.Candle{{Time: now.Add(-time.Hour), Open: 100, High: 100, Low: 100, Close: 100}}
	raw, _ := json.Marshal(c)
	if err := os.WriteFile(filepath.Join(dir, "GAZP_Minutes30.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(dir, "GAZP", now.AddDate(-2, 0, 0), now); err == nil {
		t.Fatal("missing dailies accepted")
	}
}

func TestPrepareRejectsInsufficientDailyWarmup(t *testing.T) {
	dir := t.TempDir()
	end := time.Date(2026, 10, 14, 0, 0, 0, 0, time.FixedZone("MSK", 3*3600))
	p, _ := live.ParamsFor("GAZP")
	n := core.NewWithParams("GAZP", p).Lookback() + 2
	candles := make([]domain.Candle, n)
	for i := range candles {
		candles[i] = domain.Candle{Time: end.Add(time.Duration(i-n) * 30 * time.Minute), Open: 100, High: 101, Low: 99, Close: 100}
	}
	raw, _ := json.Marshal(candles)
	if err := os.WriteFile(filepath.Join(dir, "GAZP_Minutes30.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal([]domain.Candle{{Time: end.AddDate(0, 0, -1), Open: 100, High: 101, Low: 99, Close: 100}})
	if err := os.WriteFile(filepath.Join(dir, "GAZP_Day1.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(dir, "GAZP", end.AddDate(-2, 0, 0), end); err == nil || !strings.Contains(err.Error(), "daily warmup") {
		t.Fatalf("err=%v want daily warmup rejection", err)
	}
}
