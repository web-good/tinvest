package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	svc "tinvest/internal/service/backtest"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func okCfg() runCfg {
	return runCfg{tickers: []string{"GAZP"}, months: 36, trainMonths: 12, testMonths: 3, minTrades: 30, workers: 8, commission: 0.0005}
}

func TestValidate(t *testing.T) {
	if err := validate(okCfg()); err != nil {
		t.Fatalf("valid cfg rejected: %v", err)
	}
	bad := map[string]func(*runCfg){
		"empty universe":   func(c *runCfg) { c.tickers = nil },
		"zero train":       func(c *runCfg) { c.trainMonths = 0 },
		"zero test":        func(c *runCfg) { c.testMonths = 0 },
		"window too short": func(c *runCfg) { c.months = 14 },
		"min-trades":       func(c *runCfg) { c.minTrades = 0 },
		"commission":       func(c *runCfg) { c.commission = -0.001 },
		"workers":          func(c *runCfg) { c.workers = 0 },
	}
	for name, mut := range bad {
		c := okCfg()
		mut(&c)
		if err := validate(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestDefaultTickers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prod.env")
	if err := os.WriteFile(path, []byte("RSI_PULLBACK_TICKERS=UGLD\nGAP_FADE_TICKERS=SBER, T ,GAZP\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := defaultTickers(path)
	if err != nil || !reflect.DeepEqual(got, []string{"SBER", "T", "GAZP"}) {
		t.Fatalf("got %v, %v", got, err)
	}
	// No fallback to the rsi_pullback universe: that set was picked for another strategy.
	pullbackOnly := filepath.Join(dir, "pullback.env")
	_ = os.WriteFile(pullbackOnly, []byte("RSI_PULLBACK_TICKERS=UGLD,T\n"), 0o600)
	_, err = defaultTickers(pullbackOnly)
	if err == nil || !strings.Contains(err.Error(), "GAP_FADE_TICKERS") || !strings.Contains(err.Error(), "cmd/gapscreen") {
		t.Fatalf("missing key: err = %v, want GAP_FADE_TICKERS and a cmd/gapscreen hint", err)
	}
}

func TestMissingTickers(t *testing.T) {
	got := missingTickers([]string{"gazp", "XXX", "T"}, []string{"GAZP", "T"})
	if !reflect.DeepEqual(got, []string{"XXX: нет в списке акций"}) {
		t.Fatalf("got %v", got)
	}
}

func TestDedupeTickers(t *testing.T) {
	got := dedupeTickers([]string{"GAZP", "ugld", "gazp", "UGLD", "T"})
	if !reflect.DeepEqual(got, []string{"GAZP", "ugld", "T"}) {
		t.Fatalf("got %v, want first spelling of each ticker kept in order", got)
	}
}

func TestDividendExDays(t *testing.T) {
	// Bars on 2025-06-11 and 2025-06-13: 2025-06-12 is a holiday.
	bars := append(flatBars(mskDate("2025-06-11"), 14, 100), flatBars(mskDate("2025-06-13"), 14, 100)...)
	divs := []*pkgmodel.Dividend{
		{LastBuyDate: time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC), DividendType: "Regular Cash"},
		{RecordDate: time.Date(2025, 7, 22, 0, 0, 0, 0, time.UTC)}, // no last buy date: the record date is the ex-day (T+1)
		{LastBuyDate: time.Date(2025, 5, 5, 0, 0, 0, 0, time.UTC), DividendType: "Cancelled"},
	}
	barTimes := make([]time.Time, len(bars))
	for i, b := range bars {
		barTimes[i] = b.Time
	}
	got := svc.GapDividendExDays(divs, barTimes)
	want := map[string]bool{"2025-06-13": true, "2025-07-22": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ex-days = %v, want %v", got, want)
	}
}
