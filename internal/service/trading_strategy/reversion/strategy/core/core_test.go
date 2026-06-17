package core

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// mskDay builds noon of a specific MSK calendar date for weekend-aware tests.
func mskDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, mskLoc)
}

func TestAverageVolumeExcludesWeekends(t *testing.T) {
	// Jun 13/14 2026 are Sat/Sun (MSK); Jun 15 is the entry bar (excluded from its avg).
	vols := []int64{100, 200, 300, 9999, 9999, 50}
	times := []time.Time{
		mskDay(2026, 6, 10), // Wed
		mskDay(2026, 6, 11), // Thu
		mskDay(2026, 6, 12), // Fri
		mskDay(2026, 6, 13), // Sat (excluded)
		mskDay(2026, 6, 14), // Sun (excluded)
		mskDay(2026, 6, 15), // Mon (entry bar, excluded from its own average)
	}
	avg, ok := averageVolumeExcludingWeekends(vols, times, 5)
	if !ok {
		t.Fatalf("want ok=true")
	}
	if avg != 200 { // mean(100,200,300)
		t.Fatalf("want avg 200 (weekends + entry bar excluded), got %v", avg)
	}
}

func TestAverageVolumeNoTimesKeepsAllBars(t *testing.T) {
	vols := []int64{100, 200, 300, 9999, 9999, 50}
	avg, ok := averageVolumeExcludingWeekends(vols, nil, 5)
	if !ok {
		t.Fatalf("want ok=true")
	}
	want := float64(100+200+300+9999+9999) / 5 // entry bar still excluded; no weekend drop
	if avg != want {
		t.Fatalf("no-times: want %v, got %v", want, avg)
	}
}

func TestAverageVolumeNoSamplesNotOK(t *testing.T) {
	// Window is entirely weekend bars -> no surviving sample.
	vols := []int64{9999, 9999, 50}
	times := []time.Time{
		mskDay(2026, 6, 13), // Sat
		mskDay(2026, 6, 14), // Sun
		mskDay(2026, 6, 15), // Mon (entry)
	}
	if _, ok := averageVolumeExcludingWeekends(vols, times, 2); ok {
		t.Fatalf("all-weekend window: want ok=false")
	}
}

// defaultParams returns valid, entry-capable params: trend on, RSI/Stoch oversold 20.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
	}
}

// passingInput clears every entry gate: uptrend; RSI crosses DOWN into oversold while
// Stoch %D is already in the oversold zone. EMA prev == now so no spurious cross.
func passingInput() decideInput {
	return decideInput{
		price:   100,
		emaFast: 95, emaFastPrev: 95, emaSlow: 90, emaSlowPrev: 90,
		rsiPrev: 25, rsiNow: 15, rsiOK: true, // crossDown through 20 (RSI enters oversold)
		stochPrev: 10, stochNow: 8, stochOK: true, // already < 20 (Stoch already in zone)
	}
}

// openInput is an open position holding, with neutral signals (no exit): RSI above 50
// and rising, fast EMA above slow on both bars.
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100}
	in.rsiPrev, in.rsiNow = 60, 62
	in.stochPrev, in.stochNow = 50, 55
	in.emaFast, in.emaFastPrev = 95, 95
	in.emaSlow, in.emaSlowPrev = 90, 90
	in.emaOK = true
	return in
}

func TestEntryBlockedWhenOscillatorInvalid(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// Stoch warm-up: no valid %D reading. The sentinel 0 < oversold(20) would otherwise
	// masquerade as "already deep in zone" and degrade the dual gate to RSI-only.
	in := passingInput()
	in.stochOK = false
	in.stochNow, in.stochPrev = 0, 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("invalid stoch reading: want no Buy (dual confirmation must require a real reading)")
	}

	// Symmetric: RSI warm-up also blocks.
	in = passingInput()
	in.rsiOK = false
	in.rsiNow, in.rsiPrev = 0, 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("invalid rsi reading: want no Buy")
	}
}

func TestEntryAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.decide(passingInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if !strings.Contains(sig.EntryReason, "RSI(14)") || !strings.Contains(sig.EntryReason, "Stoch") {
		t.Fatalf("EntryReason missing dual detail: %q", sig.EntryReason)
	}
}

func TestEntryRequiresBothIndicators(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses in but Stoch NOT in zone -> no buy.
	in := passingInput()
	in.stochPrev, in.stochNow = 50, 55
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("RSI cross but Stoch out of zone: want no Buy")
	}

	// Stoch crosses in while RSI already in zone -> buy (the mirror branch).
	in = passingInput()
	in.rsiPrev, in.rsiNow = 15, 12     // already < 20, no fresh cross
	in.stochPrev, in.stochNow = 25, 15 // crossDown through 20
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("Stoch cross + RSI already in: want Buy, got %v", sig.Kind)
	}
}

func TestSimultaneousEntry(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.rsiPrev, in.rsiNow = 25, 15     // RSI crosses in
	in.stochPrev, in.stochNow = 25, 15 // Stoch crosses in same bar
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("both cross into zone same bar: want Buy, got %v", sig.Kind)
	}
}

func TestTrendFilterToggles(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.emaFast = 85 // fast < slow
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, fast<slow: want no Buy")
	}
	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, price<slowEMA: want no Buy")
	}

	p := defaultParams()
	p.UseTrend = 0
	s0 := NewWithParams("TEST", p)
	in = passingInput()
	in.emaFast = 85
	if sig := s0.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseTrend=0: want Buy regardless of trend, got %v", sig.Kind)
	}
}

func TestExitRSI50(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses 50 downward -> sell RSI50.
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI50" {
		t.Fatalf("RSI down-cross 50: want RSI50 sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// RSI stays above 50 -> no sell.
	in = openInput()
	in.rsiPrev, in.rsiNow = 55, 52
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("RSI above 50: should NOT sell")
	}
}

func TestExitEMACross(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// Fast EMA drops below slow EMA, RSI neutral (no 50 cross) -> sell EMAX.
	in := openInput()
	in.rsiPrev, in.rsiNow = 60, 58 // no 50 cross
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // fast now below slow
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "EMAX" {
		t.Fatalf("bearish EMA cross: want EMAX sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// Fast stays above slow -> no EMA exit.
	in = openInput()
	in.rsiPrev, in.rsiNow = 60, 58
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 94, 90
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("fast above slow: should NOT sell")
	}
}

func TestExitPrecedenceRSIWhenBoth(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45 // RSI50 fires
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("both fire: want RSI50 precedence, got %q", sig.Reason)
	}
}

func TestNoExitWhenHolding(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	if sig := s.decide(openInput()); sig.Kind == model.SignalSell {
		t.Fatalf("neutral signals: should hold, got sell %q", sig.Reason)
	}
}

func TestExitNoFalseEMACrossAtWarmup(t *testing.T) {
	p := defaultParams()
	p.UseTrend = 0
	s := NewWithParams("TEST", p)

	// Exactly SlowEMA(200) closes: the slow EMA is valid only on the last bar, so
	// emaSlowPrev is the warm-up zero sentinel. A monotone decline would make the
	// fast-minus-slow diff cross below zero purely from that sentinel; the emaOK guard
	// must suppress the spurious EMAX exit.
	n := 200
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = float64(300 - i) // strictly declining
	}
	md := strategy.MarketData{
		Price:    closes[n-1],
		Highs:    closes,
		Lows:     closes,
		Closes:   closes,
		Volumes:  make([]int64, n),
		Position: &strategy.Position{PurchasePrice: closes[n-1]},
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell && sig.Reason == "EMAX" {
		t.Fatalf("warm-up slow EMA sentinel produced a false EMAX exit")
	}
}

func TestExitRSIOversoldBreakdown(t *testing.T) {
	s := NewWithParams("T", Params{FastEMA: 50, SlowEMA: 200, RSIPeriod: 14, RSIOversold: 30, StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20})
	in := openInput()
	in.rsiOK = true
	in.rsiPrev = 32 // above the oversold zone
	in.rsiNow = 28  // crossed down through 30
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "RSIOS" {
		t.Fatalf("expected RSIOS sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoRSIOSExitJustAfterEntry(t *testing.T) {
	// On/after the entry bar RSI is already below the oversold zone, so prev < level
	// and crossDown must not fire.
	s := NewWithParams("T", Params{FastEMA: 50, SlowEMA: 200, RSIPeriod: 14, RSIOversold: 30, StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20})
	in := openInput()
	in.rsiOK = true
	in.rsiPrev = 25 // already inside the zone
	in.rsiNow = 22  // still falling, but no fresh down-cross of 30
	sig := s.decide(in)
	if sig.Kind == model.SignalSell && sig.Reason == "RSIOS" {
		t.Fatalf("RSIOS must not fire when prev already below oversold (prev=%.0f now=%.0f)", in.rsiPrev, in.rsiNow)
	}
}

func TestExplainBlocksOnTrend(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 1, Highs: []float64{1}, Lows: []float64{1},
		Closes: []float64{1}, Volumes: []int64{1}, // no EMA history -> emaSlow 0 -> not uptrend
	}
	out := s.Explain(md)
	if !strings.Contains(out, "Тренд") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should block on trend: %q", out)
	}
}

func TestExplainTrendOffSkipsGate(t *testing.T) {
	p := defaultParams()
	p.UseTrend = 0
	s := NewWithParams("TEST", p)
	md := strategy.MarketData{
		Price: 1, Highs: []float64{1}, Lows: []float64{1},
		Closes: []float64{1}, Volumes: []int64{1},
	}
	if out := s.Explain(md); strings.Contains(out, "Тренд:") {
		t.Fatalf("UseTrend=0: Explain should not show trend gate: %q", out)
	}
}

func TestExplainPositionOpen(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 100, Highs: []float64{100}, Lows: []float64{100},
		Closes: []float64{100}, Volumes: []int64{1},
		Position: &strategy.Position{PurchasePrice: 100},
	}
	if out := s.Explain(md); !strings.Contains(out, "позиция уже открыта") {
		t.Fatalf("Explain with open position: %q", out)
	}
}

func TestEntryStampsEntryATR(t *testing.T) {
	p := defaultParams()
	p.UseATRStop = 1
	p.ATRPeriod = 14
	p.StopATRMult = 1.0
	s := NewWithParams("TEST", p)

	in := passingInput()
	in.atr = 2.5
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if sig.ATR != 2.5 {
		t.Fatalf("entry must stamp sig.ATR for freeze; want 2.5 got %v", sig.ATR)
	}
}

func TestLookbackIncludesATRWhenStopOn(t *testing.T) {
	p := defaultParams()
	p.FastEMA, p.SlowEMA = 50, 200
	p.RSIPeriod, p.StochKPeriod, p.StochDSmooth = 14, 14, 3
	p.ATRPeriod = 300 // dominates SlowEMA when the stop is on

	p.UseATRStop = 1
	if got := NewWithParams("T", p).Lookback(); got != 306 {
		t.Fatalf("ATRPeriod=300 dominates: want 306, got %d", got)
	}

	p.UseATRStop = 0
	if got := NewWithParams("T", p).Lookback(); got != 205 {
		t.Fatalf("UseATRStop=0: ATRPeriod ignored, want SlowEMA+5=205, got %d", got)
	}
}

func TestBuildInputATRGate(t *testing.T) {
	n := 60
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100 + float64(i)
		highs[i] = closes[i] + 1
		lows[i] = closes[i] - 1
	}
	md := strategy.MarketData{Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: make([]int64, n)}

	p := defaultParams()
	p.ATRPeriod = 14

	p.UseATRStop = 1
	if in := NewWithParams("T", p).buildInput(md); in.atr <= 0 {
		t.Fatalf("UseATRStop=1: want atr>0, got %v", in.atr)
	}

	p.UseATRStop = 0
	if in := NewWithParams("T", p).buildInput(md); in.atr != 0 {
		t.Fatalf("UseATRStop=0: ATR must not be computed, want 0, got %v", in.atr)
	}
}

// atrStopParams: ATR-stop branch on, multiplier 1.0, RSI oversold 30.
func atrStopParams() Params {
	p := defaultParams()
	p.UseATRStop = 1
	p.ATRPeriod = 14
	p.StopATRMult = 1.0
	p.RSIOversold = 30
	return p
}

func TestExitATRStopFires(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput() // neutral RSI/EMA: no RSI50, no EMAX
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 94 // <= 100 - 1.0*5 = 95
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "ATRSL" {
		t.Fatalf("price below entry-ATR: want ATRSL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoATRStopAboveThreshold(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 96 // > 95 threshold
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("price above threshold: should hold, got sell %q", sig.Reason)
	}
}

func TestATRStopSkippedWhenEntryATRZero(t *testing.T) {
	// Live-trading guard: EntryATR is not persisted (0), so the stop must never fire,
	// even though price (1) is far below PurchasePrice (100).
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 0}
	in.price = 1
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "ATRSL" {
		t.Fatalf("EntryATR=0 must skip ATRSL (live-trading guard)")
	}
}

func TestRSIOSInertWhenATRStopOn(t *testing.T) {
	// With the ATR stop on, the RSIOS branch is disabled: an RSI break of the oversold
	// zone must NOT produce a sell (price is above the ATR threshold).
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 99 // above the 95 ATR threshold
	in.rsiOK = true
	in.rsiPrev, in.rsiNow = 32, 28 // would be an RSIOS down-cross of 30
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("UseATRStop=1: RSIOS must be inert, got sell %q", sig.Reason)
	}
}

func TestExitPrecedenceRSI50OverATR(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 94                  // ATRSL would fire
	in.rsiPrev, in.rsiNow = 55, 45 // RSI50 also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("RSI50 must win over ATRSL, got %q", sig.Reason)
	}
}

func TestExitPrecedenceATROverEMA(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 94                  // ATRSL fires
	in.rsiPrev, in.rsiNow = 60, 58 // no RSI50
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX also fires
	if sig := s.decide(in); sig.Reason != "ATRSL" {
		t.Fatalf("ATRSL must win over EMAX, got %q", sig.Reason)
	}
}

func TestATRStopSkippedWhenMultZero(t *testing.T) {
	// Zero-multiplier guard: a 0 StopATRMult would put the threshold at PurchasePrice
	// (a break-even stop), so the guard must keep ATRSL inert even below entry.
	p := atrStopParams()
	p.StopATRMult = 0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price = 99 // below PurchasePrice but ATRSL must not fire
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "ATRSL" {
		t.Fatalf("StopATRMult=0 must skip ATRSL (zero-multiplier guard)")
	}
}

func TestBuildInputVolumeGate(t *testing.T) {
	n := 60
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	vols := make([]int64, n)
	times := make([]time.Time, n)
	base := mskDay(2026, 1, 1)
	for i := 0; i < n; i++ {
		closes[i] = 100 + float64(i)
		highs[i] = closes[i] + 1
		lows[i] = closes[i] - 1
		vols[i] = 1000
		times[i] = base.AddDate(0, 0, i)
	}
	md := strategy.MarketData{Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Times: times}

	p := defaultParams()
	p.VolAvgPeriod = 20

	p.UseVolume = 1
	if in := NewWithParams("T", p).buildInput(md); !in.volOK || in.avgVol <= 0 || in.entryVol != 1000 {
		t.Fatalf("UseVolume=1: want volOK, avgVol>0, entryVol=1000; got volOK=%v avg=%v entry=%v", in.volOK, in.avgVol, in.entryVol)
	}

	p.UseVolume = 0
	if in := NewWithParams("T", p).buildInput(md); in.volOK || in.avgVol != 0 || in.entryVol != 0 {
		t.Fatalf("UseVolume=0: volume inputs must stay zero, got volOK=%v avg=%v entry=%v", in.volOK, in.avgVol, in.entryVol)
	}
}

func TestLookbackIncludesVolumeWindow(t *testing.T) {
	p := defaultParams()
	p.FastEMA, p.SlowEMA = 50, 200
	p.RSIPeriod, p.StochKPeriod, p.StochDSmooth = 14, 14, 3
	p.VolAvgPeriod = 300 // dominates SlowEMA when the filter is on

	p.UseVolume = 1
	if got := NewWithParams("T", p).Lookback(); got != 306 {
		t.Fatalf("VolAvgPeriod=300 dominates: want 306, got %d", got)
	}

	p.UseVolume = 0
	if got := NewWithParams("T", p).Lookback(); got != 205 {
		t.Fatalf("UseVolume=0: VolAvgPeriod ignored, want SlowEMA+5=205, got %d", got)
	}
}

func TestVolumeGateBlocksBelowAverage(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 800 // below average -> blocked
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("entry volume below average: want no Buy")
	}
}

func TestVolumeGateAllowsAtOrAboveAverage(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 1000 // exactly at average -> allowed
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("entry volume at average: want Buy, got %v", sig.Kind)
	}
}

func TestVolumeGateMultiplierRaisesBar(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.5
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 1200 // passes at 1.0, fails at 1.5 (threshold 1500)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("VolMult=1.5: 1200 < 1500 threshold, want no Buy")
	}
}

func TestVolumeGateOffIgnoresVolume(t *testing.T) {
	p := defaultParams() // UseVolume defaults to 0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 1 // would be blocked if the gate were on
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseVolume=0: volume must be ignored, want Buy, got %v", sig.Kind)
	}
}

func TestVolumeGateSkippedWhenNotOK(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = false // baseline unavailable -> gate must not block
	in.avgVol = 0
	in.entryVol = 0
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("volOK=false: gate must be skipped, want Buy, got %v", sig.Kind)
	}
}

func TestExplainBlocksOnVolume(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 800
	out := s.explainFrom(in)
	if !strings.Contains(out, "Объём") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should block on volume, got: %q", out)
	}
}

func TestExitOverbought(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	// Both oscillators in their overbought zones -> sell OB.
	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75     // RSI >= 70
	in.stochPrev, in.stochNow = 82, 85 // Stoch >= 80
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "OB" {
		t.Fatalf("both overbought: want OB sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoOverboughtExitWhenOnlyOneZone(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	// Only RSI in zone -> hold.
	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 50, 55 // below 80
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("only RSI overbought: should NOT sell, got %q", sig.Reason)
	}

	// Only Stoch in zone -> hold.
	in = openInput()
	in.rsiPrev, in.rsiNow = 60, 62 // below 70
	in.stochPrev, in.stochNow = 82, 85
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("only Stoch overbought: should NOT sell, got %q", sig.Reason)
	}
}

func TestOverboughtExitOffByFlag(t *testing.T) {
	p := defaultParams() // UseOverbought defaults to 0
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("UseOverbought=0: should NOT sell, got %q", sig.Reason)
	}
}

func TestOverboughtExitTakesPrecedence(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	// Both overbought AND a bearish EMA cross also fires; OB must win.
	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX would fire
	if sig := s.decide(in); sig.Reason != "OB" {
		t.Fatalf("both OB and EMAX fire: want OB precedence, got %q", sig.Reason)
	}
}

func TestOverboughtExitSkippedAtWarmup(t *testing.T) {
	p := defaultParams()
	p.UseOverbought = 1
	p.RSIOverbought, p.StochOverbought = 70, 80
	s := NewWithParams("TEST", p)

	in := openInput()
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	in.rsiOK = false // warm-up: no valid RSI reading
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "OB" {
		t.Fatalf("rsiOK=false: OB must not fire")
	}
}

// htfPassingInput clears every gate AND has a confirmed up HTF trend.
func htfPassingInput() decideInput {
	in := passingInput()
	in.htfOK = true
	in.htfClose = 110
	in.htfEMA = 100 // close > EMA -> uptrend
	return in
}

func TestHTFGateOffWhenZero(t *testing.T) {
	// HTFTrendEMA=0: gate не читает HTF, вход определяется остальными фильтрами.
	s := NewWithParams("TEST", defaultParams()) // HTFTrendEMA=0
	in := passingInput()                        // htfOK=false, но gate выключен
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("HTFTrendEMA=0: want Buy, got %v", sig.Kind)
	}
}

func TestHTFGatePassesUptrend(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	if sig := s.decide(htfPassingInput()); sig.Kind != model.SignalBuy {
		t.Fatalf("HTF up + все фильтры пройдены: want Buy, got %v", sig.Kind)
	}
}

func TestHTFGateBlocksDowntrend(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := htfPassingInput()
	in.htfClose, in.htfEMA = 90, 100 // close < EMA -> downtrend
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("HTF down: want no Buy несмотря на пройденные hour1-фильтры")
	}
}

func TestHTFGateBlocksMissingData(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := htfPassingInput()
	in.htfOK = false // не хватило 4H-данных
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("htfOK=false при HTFTrendEMA>0: want no Buy (защитный фильтр блокирует)")
	}
}

func TestHTFGateDoesNotAffectExits(t *testing.T) {
	// Открытая позиция: путь manage не должен зависеть от HTF gate.
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := openInput()
	in.htfOK, in.htfClose, in.htfEMA = false, 0, 0 // HTF "не подтверждён"
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("в позиции gate входа не применяется")
	}
	// Никакого ложного выхода от gate: openInput нейтрален -> SignalNone.
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("HTF gate не должен порождать выход; got %v/%v", sig.Kind, sig.Reason)
	}
	// Позитивное доказательство: нормальный выход (RSI50) срабатывает даже при htfOK=false —
	// путь manage полностью независим от HTF gate.
	in.rsiPrev, in.rsiNow = 55, 45 // RSI пересекает 50 сверху вниз -> выход RSI50
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI50" {
		t.Fatalf("выход manage должен сработать независимо от HTF gate; got %v/%v", sig.Kind, sig.Reason)
	}
}

func TestExplainHTFBlock(t *testing.T) {
	p := defaultParams()
	p.HTFTrendEMA = 20
	s := NewWithParams("TEST", p)
	in := htfPassingInput()
	in.htfClose, in.htfEMA = 90, 100 // HTF вниз
	out := s.explainFrom(in)
	if !strings.Contains(out, "HTF") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain должен показать блок по HTF: %q", out)
	}
}

func TestBuildInputATRForBreakeven(t *testing.T) {
	n := 60
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = 100 + float64(i)
		highs[i] = closes[i] + 1
		lows[i] = closes[i] - 1
	}
	md := strategy.MarketData{Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: make([]int64, n)}

	p := defaultParams()
	p.ATRPeriod = 14
	p.UseATRStop = 0

	p.UseBreakeven = 1
	if in := NewWithParams("T", p).buildInput(md); in.atr <= 0 {
		t.Fatalf("UseBreakeven=1: want atr>0 (BE needs EntryATR), got %v", in.atr)
	}

	p.UseBreakeven = 0
	if in := NewWithParams("T", p).buildInput(md); in.atr != 0 {
		t.Fatalf("UseBreakeven=0 & UseATRStop=0: ATR must not be computed, got %v", in.atr)
	}
}

func TestLookbackIncludesATRForBreakeven(t *testing.T) {
	p := defaultParams()
	p.FastEMA, p.SlowEMA = 50, 200
	p.RSIPeriod, p.StochKPeriod, p.StochDSmooth = 14, 14, 3
	p.ATRPeriod = 300 // dominates SlowEMA when ATR is needed
	p.UseATRStop = 0

	p.UseBreakeven = 1
	if got := NewWithParams("T", p).Lookback(); got != 306 {
		t.Fatalf("UseBreakeven=1: ATRPeriod=300 dominates, want 306, got %d", got)
	}

	p.UseBreakeven = 0
	if got := NewWithParams("T", p).Lookback(); got != 205 {
		t.Fatalf("UseBreakeven=0: ATRPeriod ignored, want SlowEMA+5=205, got %d", got)
	}
}

func TestBuildInputHTFGate(t *testing.T) {
	htf := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20} // 11 точек, растущие

	// Минимальный валидный набор hour1-свечей для прочих индикаторов.
	closes := make([]float64, 60)
	for i := range closes {
		closes[i] = 100
	}
	md := strategy.MarketData{
		Price: 100, Closes: closes, Highs: closes, Lows: closes,
		Volumes:   make([]int64, len(closes)),
		HTFCloses: htf, HTFHighs: htf, HTFLows: htf,
	}

	// Gate выключен (HTFTrendEMA=0): htf-значения не читаются.
	p := defaultParams()
	if in := NewWithParams("T", p).buildInput(md); in.htfOK || in.htfEMA != 0 || in.htfClose != 0 {
		t.Fatalf("HTFTrendEMA=0: want htfOK=false, htfEMA=0, htfClose=0; got %v/%v/%v", in.htfOK, in.htfEMA, in.htfClose)
	}

	// Gate включён: EMA посчитана, htfClose = последний 4H-close, htfOK=true.
	p.HTFTrendEMA = 5
	in := NewWithParams("T", p).buildInput(md)
	if !in.htfOK {
		t.Fatalf("HTFTrendEMA=5 с достаточными данными: want htfOK=true")
	}
	if in.htfClose != 20 {
		t.Fatalf("htfClose = %v, want 20 (последний HTFCloses)", in.htfClose)
	}
	if in.htfEMA <= 0 {
		t.Fatalf("htfEMA = %v, want >0", in.htfEMA)
	}

	// Недостаточно данных (len < HTFTrendEMA): htfOK=false.
	p.HTFTrendEMA = 50
	if in := NewWithParams("T", p).buildInput(md); in.htfOK {
		t.Fatalf("HTFTrendEMA=50 при 11 точках: want htfOK=false")
	}
}

// breakevenParams: breakeven exit on, arm at 1.0×ATR, RSI oversold 30. ATR stop OFF so
// the middle exit is RSIOS and cannot mask the BE branch.
func breakevenParams() Params {
	p := defaultParams()
	p.UseBreakeven = 1
	p.BreakevenArmATR = 1.0
	p.RSIOversold = 30
	return p
}

func TestExitBreakevenFiresAfterArm(t *testing.T) {
	s := NewWithParams("T", breakevenParams())
	in := openInput() // neutral RSI/EMA: no RSI50, no EMAX
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106}
	in.price = 99 // <= entry 100, armed (106 >= 100 + 1.0*5 = 105)
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "BE" {
		t.Fatalf("armed + price<=entry: want BE sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoBreakevenBeforeArm(t *testing.T) {
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 103} // 103 < 105: not armed
	in.price = 99
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("not armed: should hold, got sell %q", sig.Reason)
	}
}

func TestBreakevenInertOnEntryBar(t *testing.T) {
	// At entry the engine seeds MaxFavorablePrice to the entry price, so the armed
	// threshold (entry + arm×ATR, arm>0) cannot be met on the entry bar even though
	// price == entry satisfies the price<=entry leg. BE must not fire on the entry bar.
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 100} // seeded to entry, not armed
	in.price = 100
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "BE" {
		t.Fatalf("entry bar (MaxFavorablePrice==entry) must not arm BE")
	}
}

func TestBreakevenOffByFlag(t *testing.T) {
	p := breakevenParams()
	p.UseBreakeven = 0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106} // armed
	in.price = 99
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("UseBreakeven=0 must skip BE (and nothing else should fire here), got sell %q", sig.Reason)
	}
}

func TestBreakevenInertWhenEntryATRZero(t *testing.T) {
	// Live-trading guard: EntryATR not persisted (0). Without the guard the arm threshold
	// collapses to PurchasePrice and BE would fire on any pullback.
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 0, MaxFavorablePrice: 200}
	in.price = 1
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "BE" {
		t.Fatalf("EntryATR=0 must skip BE (live-trading guard)")
	}
}

func TestBreakevenInertWhenArmZero(t *testing.T) {
	// Zero arm: threshold equals PurchasePrice (armed immediately), turning BE into a plain
	// entry-price stop. Guard keeps it inert.
	p := breakevenParams()
	p.BreakevenArmATR = 0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106}
	in.price = 99
	if sig := s.decide(in); sig.Kind == model.SignalSell && sig.Reason == "BE" {
		t.Fatalf("BreakevenArmATR=0 must skip BE (zero-arm guard)")
	}
}

func TestExitPrecedenceBreakevenOverMiddle(t *testing.T) {
	// Armed, ATR stop also on. Price is below the ATR threshold (ATRSL would fire) AND
	// below entry (BE fires). BE has higher precedence -> reason BE.
	p := breakevenParams()
	p.UseATRStop = 1
	p.ATRPeriod = 14
	p.StopATRMult = 1.0
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106} // armed
	in.price = 94 // <= 95 ATR threshold AND <= 100 entry
	if sig := s.decide(in); sig.Reason != "BE" {
		t.Fatalf("BE must win over ATRSL, got %q", sig.Reason)
	}
}

func TestExitPrecedenceRSI50OverBreakeven(t *testing.T) {
	s := NewWithParams("T", breakevenParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5, MaxFavorablePrice: 106} // armed
	in.price = 99                  // BE would fire
	in.rsiPrev, in.rsiNow = 55, 45 // RSI50 also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("RSI50 must win over BE, got %q", sig.Reason)
	}
}

func TestBuyEmitsCatStop(t *testing.T) {
	p := defaultParams()
	p.CatStopATRMult = 2.0
	p.ATRPeriod = 14
	s := NewWithParams("TEST", p)
	in := passingInput() // price = 100, all entry gates pass
	in.atr = 3.0
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected Buy, got %v", sig.Kind)
	}
	want := in.price - 2.0*3.0 // 100 - 6 = 94
	if sig.StopLoss != want {
		t.Fatalf("StopLoss = %.4f, want %.4f", sig.StopLoss, want)
	}
}

func TestBuyNoCatStopWhenATRZero(t *testing.T) {
	p := defaultParams()
	p.CatStopATRMult = 2.0
	p.ATRPeriod = 14
	s := NewWithParams("TEST", p)
	in := passingInput()
	in.atr = 0 // ATR unavailable
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected Buy, got %v", sig.Kind)
	}
	if sig.StopLoss != 0 {
		t.Fatalf("StopLoss should be 0 when atr=0, got %.4f", sig.StopLoss)
	}
}

func TestRegimeGateBlocksTrend(t *testing.T) {
	p := defaultParams()
	p.UseRegime = 1
	p.ADXPeriod = 14
	p.ADXMax = 25
	s := NewWithParams("TEST", p)

	in := passingInput()
	in.adxOK = true
	in.adx = 30 // trending -> blocked
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("ADX 30 >= 25 should block the buy")
	}

	in = passingInput()
	in.adxOK = true
	in.adx = 15 // ranging -> allowed
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("ADX 15 < 25 should allow the buy")
	}
}

func TestRegimeGateBlocksWhenUnwarmed(t *testing.T) {
	p := defaultParams()
	p.UseRegime = 1
	p.ADXPeriod = 14
	p.ADXMax = 25
	s := NewWithParams("TEST", p)
	in := passingInput()
	in.adxOK = false // not warmed -> protective block
	in.adx = 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("un-warmed ADX must block the buy")
	}
}

func TestManageCatStopExit(t *testing.T) {
	p := defaultParams()
	p.CatStopATRMult = 2.0
	s := NewWithParams("TEST", p)
	in := openInput() // in.pos != nil, neutral signals -> no other exit fires
	in.pos.PurchasePrice = 100
	in.pos.EntryATR = 3.0
	in.price = 100 - 2.0*3.0 - 0.01 // 93.99 < stop 94
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("got Kind=%v Reason=%q, want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 94.0 {
		t.Fatalf("StopLoss = %.4f, want 94.0", sig.StopLoss)
	}
}
