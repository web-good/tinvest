// legacy_golden_test.go pins the trades the rsi_zone entry produced BEFORE the stochastic gate
// existed, replayed through the real backtest engine on a seeded synthetic history for the core
// baseline and every calibrated ticker literal. With UseStoch left at zero the gate must be
// invisible: any change to entries, exits, prices or journal text fails here. Regenerate only
// deliberately: go test ./internal/service/trading_strategy/rsi_zone/strategy/core/ -run TestLegacyTradesGolden -update
package core_test

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden from the current code")

// syntheticHistory builds a seeded random walk of 30-minute weekday bars (16 per day,
// 10:00–17:30 MSK) with a mild upward drift, plus one daily candle per weekday aggregated
// from them. The drift keeps the close above long EMAs often enough for entries; the noise
// makes RSI(4) cross its bands and the stop fire.
func syntheticHistory(seed int64, days int) (bars, daily []bt.Candle) {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	r := rand.New(rand.NewSource(seed))
	p := 100.0
	first := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	for d := 0; len(daily) < days; d++ {
		day := first.AddDate(0, 0, d)
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		var dc bt.Candle
		for b := 0; b < 16; b++ {
			o := p
			p *= 1 + 0.0002 + 0.004*r.NormFloat64()
			hi := math.Max(o, p) * (1 + 0.001*r.Float64())
			lo := math.Min(o, p) * (1 - 0.001*r.Float64())
			c := bt.Candle{
				Time: day.Add(10*time.Hour + time.Duration(b)*30*time.Minute),
				Open: o, High: hi, Low: lo, Close: p, Volume: 1000,
			}
			bars = append(bars, c)
			if b == 0 {
				dc = bt.Candle{Time: day.Add(10 * time.Hour), Open: o, High: hi, Low: lo}
			}
			dc.High = math.Max(dc.High, hi)
			dc.Low = math.Min(dc.Low, lo)
			dc.Close = p
		}
		daily = append(daily, dc)
	}
	return bars, daily
}

// legacyCases are the parameter sets whose trades are pinned: the core baseline and every
// ticker literal registered today.
func legacyCases() []struct {
	name string
	p    core.Params
} {
	return []struct {
		name string
		p    core.Params
	}{
		{"core", core.DefaultParams()},
		{afks.Ticker, afks.DefaultParams()},
		{baza.Ticker, baza.DefaultParams()},
		{dias.Ticker, dias.DefaultParams()},
		{domrf.Ticker, domrf.DefaultParams()},
		{lent.Ticker, lent.DefaultParams()},
	}
}

// renderTrades formats every trade on one line: times, prices, exit code and both journal texts.
func renderTrades(name string, trades []bt.Trade) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "== %s: %d trades\n", name, len(trades))
	for _, tr := range trades {
		fmt.Fprintf(&sb, "%s %.6f -> %s %.6f %s | %s | %s\n",
			tr.EntryTime.Format(time.RFC3339), tr.EntryPrice,
			tr.ExitTime.Format(time.RFC3339), tr.ExitPrice, tr.Reason,
			tr.EntryReason, tr.ExitReason)
	}
	return sb.String()
}

func TestLegacyTradesGolden(t *testing.T) {
	bars, daily := syntheticHistory(20260930, 400)
	cfg := bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Commission: 0.0005, Lot: 1}

	var sb strings.Builder
	for _, c := range legacyCases() {
		res := bt.Run(core.NewWithParams(c.name, c.p), bars, daily, nil, cfg)
		// The fixture must actually exercise the entry, or the golden would pin nothing.
		if len(res.Trades) < 10 {
			t.Fatalf("%s: %d trades on the synthetic history, want >= 10 — the fixture no longer exercises the entry", c.name, len(res.Trades))
		}
		sb.WriteString(renderTrades(c.name, res.Trades))
	}
	got := sb.String()

	path := filepath.Join("testdata", "legacy_trades.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update once on the pre-change code): %v", err)
	}
	if got != string(want) {
		t.Fatalf("rsi_zone trades drifted from %s.\nThe stochastic gate must be invisible with UseStoch=0.\n--- got ---\n%s", path, got)
	}
}
