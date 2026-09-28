package screenrun

import (
	"context"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/model"
)

func TestPauseAfterTickerSleepsForTheRequestedDelay(t *testing.T) {
	start := time.Now()
	if !PauseAfterTicker(context.Background(), 30*time.Millisecond) {
		t.Fatal("PauseAfterTicker returned false on a live context, want true")
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("slept %v, want at least 30ms — the whole point of -pause is to idle the CPU", elapsed)
	}
}

func TestPauseAfterTickerIsFreeWhenDisabled(t *testing.T) {
	// -pause 0 is the default: no timer, no allocation, no measurable delay.
	start := time.Now()
	if !PauseAfterTicker(context.Background(), 0) {
		t.Fatal("PauseAfterTicker returned false for a zero delay, want true")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("slept %v on a zero delay, want an immediate return", elapsed)
	}
}

func TestPauseAfterTickerAbandonsTheSleepOnCancel(t *testing.T) {
	// Ctrl+C during a paced overnight run must not wait out the pause on every
	// in-flight worker before the pool unwinds.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if PauseAfterTicker(ctx, time.Hour) {
		t.Fatal("PauseAfterTicker returned true on a cancelled context, want false")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("took %v to notice cancellation, want an immediate return", elapsed)
	}
}

func TestEffectiveWorkersCapsOnRefresh(t *testing.T) {
	// 8 workers against the market-data API is roughly 26 req/s, well above what the
	// candle provider's cache writer can survive: Load(refresh=true) logs a failed
	// chunk and keeps going, then writes whatever it got over the existing cache file —
	// silently punching holes in the 540MB local cache the whole screener depends on.
	// -refresh must therefore run gentle regardless of what -workers asked for.
	got, capped := EffectiveWorkers(8, true)
	if got != 2 || !capped {
		t.Fatalf("EffectiveWorkers(8, true) = %d,%v, want 2,true", got, capped)
	}
}

func TestEffectiveWorkersLeavesLowRequestAlone(t *testing.T) {
	got, capped := EffectiveWorkers(1, true)
	if got != 1 || capped {
		t.Fatalf("EffectiveWorkers(1, true) = %d,%v, want 1,false — already at or below the refresh cap", got, capped)
	}
}

func TestEffectiveWorkersIgnoresCapWithoutRefresh(t *testing.T) {
	got, capped := EffectiveWorkers(8, false)
	if got != 8 || capped {
		t.Fatalf("EffectiveWorkers(8, false) = %d,%v, want 8,false — the cache-corruption risk is specific to -refresh", got, capped)
	}
}

func TestFilterSharesKeepsTradableRubAndCarriesTick(t *testing.T) {
	shares := []*model.Share{
		{Ticker: "SBER", Name: "Сбербанк", ID: "id-sber", Lot: 10, Currency: "rub", Trading: true, MinPriceIncrement: 0.01},
		{Ticker: "USDX", Name: "Dollar", ID: "id-usd", Lot: 1, Currency: "usd", Trading: true, MinPriceIncrement: 0.01},
		{Ticker: "DEAD", Name: "Halted", ID: "id-dead", Lot: 1, Currency: "RUB", Trading: false, MinPriceIncrement: 0.1},
		nil,
	}
	got := FilterShares(shares, nil)
	want := []ShareInfo{{Ticker: "SBER", Name: "Сбербанк", ID: "id-sber", Lot: 10, MinPriceIncrement: 0.01}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterShares = %+v, want %+v", got, want)
	}
}

func TestFilterSharesExplicitListIgnoresCurrencyAndCase(t *testing.T) {
	// -tickers is a diagnostics escape hatch: it must reach a ticker the universe
	// filter would drop, matched case-insensitively.
	shares := []*model.Share{
		{Ticker: "SBER", ID: "id-sber", Currency: "rub", Trading: true},
		{Ticker: "DEAD", ID: "id-dead", Currency: "rub", Trading: false, MinPriceIncrement: 0.1},
	}
	got := FilterShares(shares, []string{"dead"})
	if len(got) != 1 || got[0].Ticker != "DEAD" || got[0].MinPriceIncrement != 0.1 {
		t.Fatalf("FilterShares(only=dead) = %+v, want only DEAD with its tick", got)
	}
}

func TestSplitCSV(t *testing.T) {
	if got := SplitCSV("  "); got != nil {
		t.Fatalf("SplitCSV(blank) = %v, want nil", got)
	}
	got := SplitCSV(" SBER, ,AFKS ,")
	if !reflect.DeepEqual(got, []string{"SBER", "AFKS"}) {
		t.Fatalf("SplitCSV = %v, want [SBER AFKS]", got)
	}
}
