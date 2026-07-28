package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// msk is the timezone every session rule is anchored to.
var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// dailyBars builds `days` consecutive calendar days ending the day BEFORE `before` (MSK),
// oldest-first. Every bar closes at 100; a weekday bar spans `w` and a weekend bar spans
// `we`, so a test can prove weekend sessions never reach the ATR. With a flat close the true
// range of each bar equals its own width, so ATR over N equal weekday bars is exactly `w`.
func dailyBars(before time.Time, days int, w, we float64) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(msk)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, msk).AddDate(0, 0, -days)
	const price = 100.0
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		width := w
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			width = we
		}
		highs = append(highs, price+width/2)
		lows = append(lows, price-width/2)
		closes = append(closes, price)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// barSeries builds MarketData from closes, stamping bars every 30 minutes starting at start.
// Highs/Lows are derived from the close with a fixed 0.3% envelope so ATR is always positive.
func barSeries(closes []float64, start time.Time) strategy.MarketData {
	n := len(closes)
	md := strategy.MarketData{
		Highs:   make([]float64, n),
		Lows:    make([]float64, n),
		Closes:  append([]float64(nil), closes...),
		Volumes: make([]int64, n),
		Times:   make([]time.Time, n),
	}
	for i, c := range closes {
		md.Highs[i] = c * 1.003
		md.Lows[i] = c * 0.997
		md.Volumes[i] = 1000
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
	md.Price = closes[n-1]
	return md
}

// pullbackCloses builds a long uptrend (so the fast EMA sits above the slow one) followed by a
// sharp multi-bar drop that pushes a short RSI below the lower band on the LAST bar.
func pullbackCloses() []float64 {
	const rise = 400
	out := make([]float64, 0, rise+6)
	p := 100.0
	for i := 0; i < rise; i++ {
		p *= 1.0008
		out = append(out, p)
	}
	// Five consecutive down bars: RSI(4) crosses below the lower band (15) only on the last one.
	for i := 0; i < 5; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// withDay attaches a completed weekday daily series (40 days, weekday width `atrWidth`) plus
// an explicit intraday extent to md, so entry tests can drive the day gate deterministically.
// The daily ATR of the attached series is exactly atrWidth.
func withDay(md strategy.MarketData, atrWidth, todayHigh, todayLow float64) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	h, l, c, ts := dailyBars(last, 40, atrWidth, atrWidth/10)
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = h, l, c, ts
	md.TodayHigh, md.TodayLow = todayHigh, todayLow
	return md
}

// entryFixture returns market data whose LAST bar is a valid entry bar at 12:00 MSK Monday.
func entryFixture() strategy.MarketData {
	closes := pullbackCloses()
	// 2026-06-01 is a Monday. Place the last bar at 12:00 MSK.
	last := time.Date(2026, 6, 1, 12, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

// downtrendCloses builds a LONG DECLINE (so the fast EMA ends up AT OR BELOW the slow one)
// followed by the same shape of multi-bar shock as pullbackCloses, calibrated so RSI(4) crosses
// below the lower band (15) only on the LAST bar. This isolates the trend gate in
// TestEnterGates's "downtrend" case: the RSI gate passes cleanly (same fresh-cross shape as the
// buy fixture), so a rejection can only come from fast[i] <= slow[i].
//
// A pure monotonic decline will NOT work here: Wilder's avgGain sits at exactly 0 when every bar
// is a loss (see pkg/indicators/rsi.go), which pins RSI(4) at its 0 floor from the very first
// down bar — there is no ">= 15" bar left to cross down FROM. The baseline below alternates a
// smaller up-tick with a larger down-tick (net down) to keep avgGain > 0 throughout, so RSI has
// room to sit above 15 until the shock. The exact multipliers were found by a small offline
// search over indicators.RSISeries/ema.Compute output (not eyeballed): with this shape,
// rsi[n-2]=15.45 (>=15) and rsi[n-1]=11.40 (<15), while fast(10)=99.88 <= slow(100)=99.96.
func downtrendCloses() []float64 {
	const baseLen = 400
	out := make([]float64, 0, baseLen+6)
	p := 100.0
	for i := 0; i < baseLen; i++ {
		if i%2 == 0 {
			p *= 0.9996
		} else {
			p *= 1.0004
		}
		out = append(out, p)
	}
	// Five consecutive down bars: same role as pullbackCloses' shock, calibrated so the RSI(4)
	// cross below 15 lands on the LAST bar (not earlier).
	for i := 0; i < 5; i++ {
		p *= 0.9995
		out = append(out, p)
	}
	return out
}

// downtrendFixture returns market data whose LAST bar clears the RSI entry gate (fresh cross
// below RSILower) but sits inside a downtrend, so fast EMA <= slow EMA — used to prove the trend
// gate alone blocks the entry.
func downtrendFixture() strategy.MarketData {
	closes := downtrendCloses()
	// 2026-06-01 is a Monday. Place the last bar at 12:00 MSK (same slot as entryFixture).
	last := time.Date(2026, 6, 1, 12, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	want := Params{
		RSIPeriod: 4, RSILower: 15, RSIUpper: 70,
		EMAFast: 10, EMASlow: 100,
		DailyATRPeriod: 14,
		UseDayATRGate:  1, FreshDayATR: 0.3, SpentDayATR: 0.8,
		StopDailyATR: 1.0, TPDailyATR: 0.6,
		MaxHoldBars:     8,
		SessionStartMin: 420, SessionEndMin: 1020, DayEndMin: 1380,
	}
	if p != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", p, want)
	}
}

func TestLookbackCoversSlowEMA(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	if got := s.Lookback(); got < 220 {
		t.Fatalf("Lookback() = %d, want >= 220 (2*EMASlow+20)", got)
	}
	p := DefaultParams()
	p.EMASlow = 200
	if got := NewWithParams("T", p).Lookback(); got != 420 {
		t.Fatalf("Lookback() with EMASlow=200 = %d, want 420", got)
	}
}

func TestEnterBuysThePullback(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := withDay(entryFixture(), 10.0, 101, 100)
	got := s.Decide(md)
	if got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy (EntryReason %q)", got.Kind, got.EntryReason)
	}
	i := len(md.Closes) - 1
	wantStop := md.Closes[i] - DefaultParams().StopDailyATR*got.ATR
	if got.ATR <= 0 {
		t.Fatalf("ATR = %v, want > 0", got.ATR)
	}
	if math.Abs(got.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", got.StopLoss, wantStop)
	}
	if got.EntryReason == "" {
		t.Fatal("EntryReason is empty")
	}
}

// TestEnterGates breaks exactly one precondition at a time and requires no entry.
func TestEnterGates(t *testing.T) {
	base := DefaultParams()
	tests := []struct {
		name  string
		tweak func(p *Params, md *strategy.MarketData)
	}{
		{"before the session opens", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 1, 6, 30, 0, 0, msk))
		}},
		{"after the entry window closes", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 1, 17, 0, 0, 0, msk))
		}},
		{"weekend", func(_ *Params, md *strategy.MarketData) {
			// 2026-06-06 is a Saturday.
			shiftTo(md, time.Date(2026, 6, 6, 12, 0, 0, 0, msk))
		}},
		{"downtrend: fast EMA below slow", func(_ *Params, md *strategy.MarketData) {
			// Swap in a fixture whose RSI gate passes cleanly (same fresh-cross shape as the
			// buy fixture) but whose price path is a genuine downtrend, so a rejection here can
			// only be explained by the trend gate — not by an earlier RSI-gate failure.
			*md = downtrendFixture()
		}},
		{"EMAFast == EMASlow: equality is not \"above\"", func(p *Params, _ *strategy.MarketData) {
			// Reuse the untouched buy fixture (its RSI gate is already proven to pass) and force
			// EMAFast and EMASlow to the SAME period. ema.Compute is a pure function of
			// (closes, period), so two calls with an identical period produce BIT-IDENTICAL
			// series — fast[i] == slow[i] exactly, no floating-point luck required. This isolates
			// the "<=" in enter()'s trend check: a gate written as "< slow" instead of "<= slow"
			// would let this exact-equality case through as a Buy.
			p.EMAFast = p.EMASlow
		}},
		{"RSI stays above the lower band", func(p *Params, _ *strategy.MarketData) {
			p.RSILower = 0.5
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			md := entryFixture()
			tc.tweak(&p, &md)
			if got := NewWithParams("T", p).Decide(md); got.Kind != model.SignalNone {
				t.Fatalf("Kind = %v, want None (reason %q)", got.Kind, got.EntryReason)
			}
		})
	}
}

// shiftTo re-stamps the series so its LAST bar opens at last.
func shiftTo(md *strategy.MarketData, last time.Time) {
	start := last.Add(-time.Duration(len(md.Times)-1) * 30 * time.Minute)
	for i := range md.Times {
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
}

// TestCrossHelpersBoundaries pins the comparison operators of crossedDown/crossedUp directly on
// synthetic series. The gate tests above only feed extreme values, which survive an off-by-one
// shift of any single comparison; the strategy's whole trade population hangs on these four
// operators, so each one gets a case that flips when it is loosened or tightened by one step.
// Verified mutationally: flipping `series[i] < level` to `<=`, `series[i-1] >= level` to `>`, or
// `series[i-1] <= level` to `<` makes this test fail.
func TestCrossHelpersBoundaries(t *testing.T) {
	const lvl = 15.0
	// justBelow/justAbove are one representable step off the level: the tightest possible probe
	// of a strict comparison.
	justBelow := math.Nextafter(lvl, 0)
	justAbove := math.Nextafter(lvl, 100)

	down := []struct {
		name   string
		series []float64
		want   bool
	}{
		{"previous exactly on the level, current below: a cross", []float64{lvl, 14}, true},
		{"previous exactly on the level, current one step below: a cross", []float64{lvl, justBelow}, true},
		{"current exactly on the level: touching is not crossing", []float64{20, lvl}, false},
		{"current one step above the level: not yet", []float64{20, justAbove}, false},
		{"previous already below the level: no fresh cross", []float64{14, 13}, false},
		{"previous below, current back on the level: no cross", []float64{14, lvl}, false},
		{"previous is a warm-up zero: ignored", []float64{0, 14}, false},
		{"single bar: no previous to cross from", []float64{14}, false},
	}
	for _, tc := range down {
		t.Run("crossedDown/"+tc.name, func(t *testing.T) {
			if got := crossedDown(tc.series, len(tc.series)-1, lvl); got != tc.want {
				t.Fatalf("crossedDown(%v, %.20g) = %v, want %v", tc.series, lvl, got, tc.want)
			}
		})
	}

	const up = 70.0
	upJustBelow := math.Nextafter(up, 0)
	upJustAbove := math.Nextafter(up, 100)
	ups := []struct {
		name   string
		series []float64
		want   bool
	}{
		{"previous exactly on the level, current above: a cross", []float64{up, 71}, true},
		{"previous exactly on the level, current one step above: a cross", []float64{up, upJustAbove}, true},
		{"current exactly on the level: touching is not crossing", []float64{60, up}, false},
		{"current one step below the level: not yet", []float64{60, upJustBelow}, false},
		{"previous already above the level: no fresh cross", []float64{71, 72}, false},
		{"previous above, current back on the level: no cross", []float64{71, up}, false},
		// Documents the asymmetric warm-up guard (see crossedUp's comment): a leading 0 is
		// treated as un-warmed even though Wilder's RSI can genuinely read 0, so the exit is
		// suppressed rather than fired off an ambiguous value.
		{"previous is zero: guarded, no cross", []float64{0, 71}, false},
		{"single bar: no previous to cross from", []float64{71}, false},
	}
	for _, tc := range ups {
		t.Run("crossedUp/"+tc.name, func(t *testing.T) {
			if got := crossedUp(tc.series, len(tc.series)-1, up); got != tc.want {
				t.Fatalf("crossedUp(%v, %.20g) = %v, want %v", tc.series, up, got, tc.want)
			}
		})
	}
}

// TestEnterAtSessionOpenBoundary pins the `m >= SessionStartMin` comparison: a bar opening
// EXACTLY at 07:00 MSK is inside the entry window. Real GAZP 30m series carry that bar, and
// TestEnterGates only probes 06:30, which survives a shift to a strict `>`.
func TestEnterAtSessionOpenBoundary(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	// 2026-06-01 is a Monday; 07:00 is SessionStartMin (420) to the minute.
	shiftTo(&md, time.Date(2026, 6, 1, 7, 0, 0, 0, msk))
	md = withDay(md, 10.0, 101, 100)
	if got := s.Decide(md); got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy on the bar opening exactly at SessionStartMin", got.Kind)
	}
}

// TestEnterCrossIsAnEventNotAState: once RSI already sits below the band on the PREVIOUS bar,
// there is no fresh cross and therefore no entry.
func TestEnterCrossIsAnEventNotAState(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	// Append one more down bar: now bar i-1 is already below the band.
	last := md.Closes[len(md.Closes)-1] * 0.994
	md = barSeries(append(md.Closes, last), md.Times[0])
	shiftTo(&md, time.Date(2026, 6, 1, 12, 30, 0, 0, msk))
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None: RSI was already below the band on the previous bar", got.Kind)
	}
}

func TestEnterRejectsShortHistory(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := barSeries([]float64{100, 99}, time.Date(2026, 6, 1, 11, 0, 0, 0, msk))
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None on a two-bar series", got.Kind)
	}
}

func TestDayStateGateBothBranches(t *testing.T) {
	p := DefaultParams() // FreshDayATR 0.3, SpentDayATR 0.8
	s := NewWithParams("TEST", p)
	const atr = 10.0
	cases := []struct {
		name string
		used float64
		want bool
	}{
		{"день только начался", 1.0, true},
		{"ровно на границе свежести", 3.0, true},
		{"чуть выше границы свежести", 3.0001, false},
		{"мёртвая зона", 5.0, false},
		{"чуть ниже границы исчерпания", 7.9999, false},
		{"ровно на границе исчерпания", 8.0, true},
		{"день исчерпан", 12.0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := strategy.MarketData{TodayHigh: 100 + tc.used, TodayLow: 100}
			if got := s.dayStateOK(md, atr); got != tc.want {
				t.Fatalf("used=%.4f: dayStateOK = %v, want %v", tc.used, got, tc.want)
			}
		})
	}
}

func TestDayStateGateDegradations(t *testing.T) {
	base := DefaultParams()
	dead := strategy.MarketData{TodayHigh: 105, TodayLow: 100} // ровно мёртвая зона при atr=10

	off := base
	off.UseDayATRGate = 0
	if !NewWithParams("TEST", off).dayStateOK(dead, 10) {
		t.Fatal("выключенный гейт обязан пропускать")
	}

	noThresholds := base
	noThresholds.FreshDayATR, noThresholds.SpentDayATR = 0, 0
	if !NewWithParams("TEST", noThresholds).dayStateOK(dead, 10) {
		t.Fatal("гейт без порогов обязан пропускать")
	}

	degenerate := base
	degenerate.FreshDayATR, degenerate.SpentDayATR = 0.8, 0.3 // ветки перекрываются
	if !NewWithParams("TEST", degenerate).dayStateOK(dead, 10) {
		t.Fatal("вырожденная конфигурация обязана пропускать всё")
	}

	s := NewWithParams("TEST", base)
	if !s.dayStateOK(strategy.MarketData{}, 10) {
		t.Fatal("без TodayHigh/TodayLow гейт обязан пропускать")
	}
	if !s.dayStateOK(dead, 0) {
		t.Fatal("без ATR гейт обязан пропускать (вход отсекается отдельной проверкой)")
	}
}

// entryDailyATRStart anchors barSeries(pullbackCloses(), ...) so the LAST of its 405 bars lands
// well inside the entry session. pullbackCloses() has a fixed length, so the offset from start to
// the last bar is always exactly 404*30min = 8 days 10 hours: starting at 07:00 (the session
// open) would land the last bar at exactly 17:00 the following Tuesday — SessionEndMin to the
// minute, which inSession's `< SessionEndMin` excludes. Starting an hour earlier keeps the last
// bar at 16:00, comfortably inside the window, without changing which bar is "last" (the RSI
// cross this fixture relies on is a property of bar order, not of the clock).
var entryDailyATRStart = time.Date(2026, 3, 2, 6, 0, 0, 0, msk)

func TestEnterSetsStopAndTargetFromDailyATR(t *testing.T) {
	p := DefaultParams()
	s := NewWithParams("TEST", p)
	md := barSeries(pullbackCloses(), entryDailyATRStart)
	md = withDay(md, 10.0, 101, 100) // used = 1 <= 0.3*10 -> ветка «день только начался»

	sig := s.Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	entry := md.Closes[len(md.Closes)-1]
	wantStop := entry - p.StopDailyATR*10.0
	wantTP := entry + p.TPDailyATR*10.0
	if math.Abs(sig.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %.6f, want %.6f", sig.StopLoss, wantStop)
	}
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("TakeProfit = %.6f, want %.6f", sig.TakeProfit, wantTP)
	}
	if math.Abs(sig.ATR-10.0) > 1e-9 {
		t.Fatalf("ATR = %.6f, want 10.0 (дневной)", sig.ATR)
	}
}

func TestEnterRefusedWithoutDailyATR(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := barSeries(pullbackCloses(), entryDailyATRStart) // дневных серий нет вовсе
	md.TodayHigh, md.TodayLow = 101, 100
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("вход без дневного ATR запрещён: нечем выставить стоп и цель")
	}
}

func TestEnterBlockedInTheDeadBand(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := withDay(barSeries(pullbackCloses(), entryDailyATRStart), 10.0, 105, 100) // used = 5, мёртвая зона
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("вход в мёртвой зоне гейта запрещён")
	}
}

// withPosition attaches an open long entered heldBars bars before the last bar of md.
func withPosition(md strategy.MarketData, entryPrice, stop float64, heldBars int) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	md.Position = &strategy.Position{
		PurchasePrice: entryPrice,
		StopLoss:      stop,
		EntryATR:      entryPrice * 0.003,
		EntryTime:     last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}

// TestExitStopLoss covers the stop, including the exact-touch boundary: `low <= StopLoss` must
// fire when the low lands ON the stop. A test that only puts the stop strictly inside the bar
// survives a shift of that comparison to `<`.
func TestExitStopLoss(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	tests := []struct {
		name string
		stop func(low float64) float64
	}{
		{"low pierces the stop", func(low float64) float64 { return low * 1.0001 }},
		{"low touches the stop exactly", func(low float64) float64 { return low }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			md := entryFixture()
			i := len(md.Closes) - 1
			stop := tc.stop(md.Lows[i])
			md = withPosition(md, md.Closes[i]*1.02, stop, 1)
			got := s.Decide(md)
			if got.Kind != model.SignalSell || got.Reason != "SL" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/SL", got.Kind, got.Reason)
			}
			if math.Abs(got.StopLoss-stop) > 1e-9 {
				t.Fatalf("StopLoss = %v, want the frozen position stop %v", got.StopLoss, stop)
			}
		})
	}
}

// upperCrossFixture builds an uptrend whose LAST bar pushes RSI(4) above the upper band.
//
// The 1.010 up-tick from the plan's literal fixture only reaches RSI(4)=69.95 on the last bar —
// just short of the 70 upper band, so no cross fires. Verified by an offline run of
// indicators.RSISeries over this exact shape (not eyeballed): 1.012 clears it cleanly, with
// rsi[n-2]=55.32 (<=70, not yet crossed) and rsi[n-1]=73.53 (>70, crossed).
func upperCrossFixture() strategy.MarketData {
	closes := make([]float64, 0, 410)
	p := 100.0
	for i := 0; i < 400; i++ {
		p *= 1.0008
		closes = append(closes, p)
	}
	// Three down bars pull RSI below the upper band, then two sharp up bars push it back above.
	for i := 0; i < 3; i++ {
		p *= 0.994
		closes = append(closes, p)
	}
	for i := 0; i < 2; i++ {
		p *= 1.012
		closes = append(closes, p)
	}
	last := time.Date(2026, 6, 1, 14, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

func TestExitOnRSIEnteringUpperBand(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// Stop far below so only the RSI exit can fire.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI", got.Kind, got.Reason)
	}
}

func TestExitStopWinsOverRSIOnTheSameBar(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// Stop inside the bar AND the RSI cross on the same bar: SL must win.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*1.0001, 2)
	got := s.Decide(md)
	if got.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL to take precedence over RSI", got.Reason)
	}
}

func TestExitTimeStop(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 3
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 3)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "TIME" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/TIME after 3 bars", got.Kind, got.Reason)
	}
}

func TestExitTimeStopDisabledAtZero(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	// 10 bars back from the 12:00 bar is 07:00 of the SAME MSK day: a long hold that still keeps
	// the EOD backstop (isDayEnd / crossedIntoNewDay) silent, so only the time stop is under test.
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 10)
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v (%q), want None: the time stop is disabled at MaxHoldBars=0", got.Kind, got.Reason)
	}
}

func TestExitTimeStopSilentOnUnknownEntryTime(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 1
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 5)
	md.Position.EntryTime = time.Time{}
	if got := s.Decide(md); got.Reason == "TIME" {
		t.Fatal("TIME fired with an unknown entry time; it must stay silent")
	}
}

func TestExitEndOfDay(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	shiftTo(&md, time.Date(2026, 6, 1, 22, 30, 0, 0, msk))
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "EOD" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/EOD on the 22:30 bar", got.Kind, got.Reason)
	}
}

// TestExitEndOfDayOnTruncatedSession covers the case isDayEnd alone cannot: the session ended
// early (data gap, halt, short trading day), so NO bar ever reaches DayEndMin and the day-end
// force close never gets its trigger bar. With MaxHoldBars=0 — a value the calibration grid
// really does offer — nothing else would close the position either, and it would ride overnight
// under a frozen stop, against the "never hold into the next day" invariant. The first bar of the
// next MSK day must close it as EOD.
func TestExitEndOfDayOnTruncatedSession(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0 // the time stop must not be what saves us here
	s := NewWithParams("T", p)

	tests := []struct {
		name       string
		lastBar    time.Time
		entryTime  time.Time
		wantKind   model.SignalKind
		wantReason string
	}{
		{
			name: "first bar of the next trading day closes a position left over from a truncated session",
			// 2026-06-01 is a Monday whose session is cut short at 20:30 (no 22:30 bar exists);
			// the next bar the engine feeds is Tuesday's open.
			lastBar:    time.Date(2026, 6, 2, 7, 0, 0, 0, msk),
			entryTime:  time.Date(2026, 6, 1, 18, 0, 0, 0, msk),
			wantKind:   model.SignalSell,
			wantReason: "EOD",
		},
		{
			name:      "same MSK day, still inside the evening session: nothing fires",
			lastBar:   time.Date(2026, 6, 1, 19, 0, 0, 0, msk),
			entryTime: time.Date(2026, 6, 1, 12, 0, 0, 0, msk),
			wantKind:  model.SignalNone,
		},
		{
			name:      "unknown entry time keeps the new-day check silent",
			lastBar:   time.Date(2026, 6, 2, 7, 0, 0, 0, msk),
			entryTime: time.Time{},
			wantKind:  model.SignalNone,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			md := entryFixture()
			shiftTo(&md, tc.lastBar)
			i := len(md.Closes) - 1
			md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
			md.Position.EntryTime = tc.entryTime
			got := s.Decide(md)
			if got.Kind != tc.wantKind || got.Reason != tc.wantReason {
				t.Fatalf("Kind/Reason = %v/%q, want %v/%q", got.Kind, got.Reason, tc.wantKind, tc.wantReason)
			}
		})
	}
}

func TestExitHoldsThroughTheEveningSession(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0
	s := NewWithParams("T", p)
	md := entryFixture()
	shiftTo(&md, time.Date(2026, 6, 1, 19, 0, 0, 0, msk))
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v (%q), want None: 19:00 is past the entry window but before DayEndMin",
			got.Kind, got.Reason)
	}
}

func TestExplainMentionsEveryGate(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	out := s.Explain(entryFixture())
	for _, want := range []string{"сессия", "RSI", "EMA", "стоп", "удержание"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain() is missing %q; got:\n%s", want, out)
		}
	}
}

// TestExplainLabelsUnknownHold: the holdUnknown sentinel must never reach the diagnostics the
// owner reads — "-1 баров" is not a hold.
func TestExplainLabelsUnknownHold(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	md.Position.EntryTime = time.Time{}
	out := s.Explain(md)
	if strings.Contains(out, "-1 баров") {
		t.Fatalf("Explain() leaked the holdUnknown sentinel; got:\n%s", out)
	}
	if !strings.Contains(out, "удержание: неизвестно") {
		t.Fatalf("Explain() must label an unknown hold; got:\n%s", out)
	}
}

// sliceMD returns md restricted to bars [from:to).
func sliceMD(md strategy.MarketData, from, to int) strategy.MarketData {
	out := md
	out.Highs = md.Highs[from:to]
	out.Lows = md.Lows[from:to]
	out.Closes = md.Closes[from:to]
	out.Volumes = md.Volumes[from:to]
	out.Times = md.Times[from:to]
	out.Price = out.Closes[len(out.Closes)-1]
	return out
}

// TestNoLookaheadAcrossWindowCuts is the load-bearing safety net: the decision on bar i must not
// depend on how much history precedes it. Cuts stay far enough from bar i that every indicator
// is warmed in both windows. The stop/target are still compared with a relative tolerance for
// robustness, even though the daily ATR they derive from is untouched by sliceMD (it lives in
// DailyHighs/DailyLows/DailyCloses, none of which sliceMD trims) and is therefore identical
// across every cut.
func TestNoLookaheadAcrossWindowCuts(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := withDay(entryFixture(), 10.0, 101, 100)
	n := len(full.Closes)

	// Every cut must drop real history: `from = 0` would compare the full window with itself and
	// assert nothing. The deepest cut still leaves ~245 bars, well past the slow EMA's warm-up.
	want := s.Decide(full)
	if want.Kind != model.SignalBuy {
		t.Fatalf("fixture produced %v on the full window; the cuts would assert nothing", want.Kind)
	}
	for _, from := range []int{40, 80, 160} {
		got := s.Decide(sliceMD(full, from, n))
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("cut at %d gave %v/%q, full window gave %v/%q",
				from, got.Kind, got.Reason, want.Kind, want.Reason)
		}
		if math.Abs(got.RSI-want.RSI) > 1e-9 {
			t.Fatalf("cut at %d gave RSI %v, full window %v", from, got.RSI, want.RSI)
		}
		if want.StopLoss > 0 {
			if rel := math.Abs(got.StopLoss-want.StopLoss) / want.StopLoss; rel > 1e-3 {
				t.Fatalf("cut at %d gave stop %v, full window %v (relative %g)",
					from, got.StopLoss, want.StopLoss, rel)
			}
		}
	}
}

// TestNoLookaheadWithOpenPosition covers the manage() path the same way.
func TestNoLookaheadWithOpenPosition(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := upperCrossFixture()
	n := len(full.Closes)
	i := n - 1
	full = withPosition(full, full.Closes[i]*0.97, full.Lows[i]*0.5, 2)

	want := s.Decide(full)
	if want.Kind == model.SignalNone {
		t.Fatal("fixture produced no exit; the managed sub-case would assert nothing")
	}
	for _, from := range []int{40, 80} {
		cut := sliceMD(full, from, n)
		cut.Position = full.Position
		got := s.Decide(cut)
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("cut at %d gave %v/%q, full window gave %v/%q",
				from, got.Kind, got.Reason, want.Kind, want.Reason)
		}
	}
}

func TestDailyATRIgnoresWeekends(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, msk) // понедельник

	// 40 календарных дней: будни шириной 2.0, выходные — намеренно узкие 0.2.
	h, l, c, ts := dailyBars(now, 40, 2.0, 0.2)
	withWeekend := strategy.MarketData{DailyHighs: h, DailyLows: l, DailyCloses: c, DailyTimes: ts}

	// Та же серия, но выходные вырезаны заранее.
	var wh, wl, wc []float64
	var wt []time.Time
	for i := range c {
		if wd := ts[i].In(msk).Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		wh = append(wh, h[i])
		wl = append(wl, l[i])
		wc = append(wc, c[i])
		wt = append(wt, ts[i])
	}
	weekdaysOnly := strategy.MarketData{DailyHighs: wh, DailyLows: wl, DailyCloses: wc, DailyTimes: wt}

	got, want := s.dailyATR(withWeekend), s.dailyATR(weekdaysOnly)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("ATR с выходными %.6f != ATR без выходных %.6f", got, want)
	}
	if math.Abs(got-2.0) > 1e-6 {
		t.Fatalf("ATR = %.6f, want 2.0 (ширина буднего бара)", got)
	}
}

func TestDailyATRZeroWhenDataCannotSupportIt(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, msk)

	// Будних дней меньше, чем DailyATRPeriod+1.
	h, l, c, ts := dailyBars(now, 10, 2.0, 0.2)
	if got := s.dailyATR(strategy.MarketData{DailyHighs: h, DailyLows: l, DailyCloses: c, DailyTimes: ts}); got != 0 {
		t.Fatalf("ATR на короткой истории = %.6f, want 0", got)
	}
	if got := s.dailyATR(strategy.MarketData{}); got != 0 {
		t.Fatalf("ATR без дневных данных = %.6f, want 0", got)
	}
}

func TestDailyATRDegradesWhenTimesMisaligned(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, msk)
	h, l, c, ts := dailyBars(now, 40, 2.0, 0.2)

	// Времена короче ценовых серий: фильтровать нечем — серия должна пойти в ATR как есть,
	// а не обнулиться и не паниковать.
	md := strategy.MarketData{DailyHighs: h, DailyLows: l, DailyCloses: c, DailyTimes: ts[:5]}
	if got := s.dailyATR(md); got <= 0 {
		t.Fatalf("ATR при рассинхроне времён = %.6f, want > 0 (деградация, не отказ)", got)
	}
}
