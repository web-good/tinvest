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
// Volume is flat at 1000 except the LAST bar, which is boosted 10x: with DefaultParams()'s
// volume-background gate armed (UseVolume=1), a flat series never beats its own slot average, so
// every entry fixture built from this helper would be blocked by a gate it isn't testing. Tests
// that exercise the volume gate itself build their own series via volSeries instead.
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
	md.Volumes[n-1] = 10000
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
		RSIPeriod: 4, RSILower: 30, RSIUpper: 70,
		EMAFast: 10, EMASlow: 100,
		DailyATRPeriod: 14,
		UseDayATRGate:  1, FreshDayATR: 0, SpentDayATR: 0.8,
		StopDailyATR: 0.5, TPDailyATR: 0.6,
		UseVolume: 0, VolBaseDays: 14, VolLookbackBars: 3, VolMult: 1.2,
		UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0,
	}
	if p != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", p, want)
	}
}

// entryParams returns the parameters the entry fixtures were BUILT for, not whatever ships in
// DefaultParams today. entryFixture()'s RSI drops from ~15.6 to ~11.1 on the last bar, and
// withDay(md, 10, 101, 100) reports a day that has used 0.1 of its daily ATR — so the fixtures
// only reach enter()'s later gates while the lower band sits at 15 and the "day barely started"
// branch is armed. Calibration moves both numbers in the shipped defaults; inheriting them here
// would silently turn every entry test into a vacuous "None for some other reason" — exactly the
// failure a mutation run caught in these tests once already. Tests that are ABOUT the defaults
// (TestDefaultParams) or about a gate's own mechanics set their values themselves.
func entryParams() Params {
	p := DefaultParams()
	p.RSILower = 15
	p.FreshDayATR = 0.3
	return p
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
	s := NewWithParams("T", entryParams())
	md := withDay(entryFixture(), 10.0, 101, 100)
	got := s.Decide(md)
	if got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy (EntryReason %q)", got.Kind, got.EntryReason)
	}
	i := len(md.Closes) - 1
	wantStop := md.Closes[i] - entryParams().StopDailyATR*got.ATR
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

// TestEnterGates breaks exactly one precondition at a time and requires no entry. Every case
// attaches a daily series via withDay (atr=10, TodayHigh/TodayLow=101/100, the same "day just
// started" shape TestEnterBuysThePullback uses) AFTER the tweak runs, so the daily-ATR gate (4)
// always has what it needs and cannot mask whichever earlier gate the case is meant to isolate.
// Without that, entryFixture() alone carries no daily series, dailyATR() is 0, and gate 4 rejects
// every case regardless of what the tweak broke — verified mutationally: before this fix, removing
// the weekday gate, the RSI-cross gate, the trend gate or the volume gate from enter() did not
// fail this test.
func TestEnterGates(t *testing.T) {
	base := entryParams()
	tests := []struct {
		name  string
		tweak func(p *Params, md *strategy.MarketData)
	}{
		{"weekend", func(p *Params, md *strategy.MarketData) {
			// 2026-06-06 is a Saturday.
			shiftTo(md, time.Date(2026, 6, 6, 12, 0, 0, 0, msk))
			// volumeOK only ever inspects WEEKDAY bars: on a Saturday last bar it skips straight
			// past barSeries' boosted last-bar volume (it is a weekend bar) and falls back to the
			// flat weekday volume behind it, which would ALSO fail the volume gate — confounding
			// this case with gate 6 instead of isolating gate 1's weekend check. Disable the
			// volume gate explicitly so only the weekday gate can block this entry.
			p.UseVolume = 0
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
		{"tape not busier than usual: volume gate", func(p *Params, md *strategy.MarketData) {
			// The gate ships disabled (UseVolume=0 in DefaultParams), so arm it here — this case
			// exists to pin that enter() actually calls volumeOK, which only matters when the
			// gate is on. barSeries always boosts the LAST bar's volume 10x so every other case
			// in this table would clear the gate anyway; flatten it back to the series' flat
			// background so gate 6 is the only thing that can reject this entry.
			p.UseVolume = 1
			md.Volumes[len(md.Volumes)-1] = 1000
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			md := entryFixture()
			tc.tweak(&p, &md)
			md = withDay(md, 10.0, 101, 100)
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

// TestEnterHasNoTimeOfDayWindow pins the removal of the entry window: the hour of a weekday bar
// must not influence the entry decision at all. The three probes are the times the retired
// 07:00-17:00 window used to reject — before the old open, exactly at the old close (which the
// old `< SessionEndMin` excluded), and deep into the evening session. A reintroduced window of
// any plausible shape fails at least one of them.
func TestEnterHasNoTimeOfDayWindow(t *testing.T) {
	// 2026-06-01 is a Monday, so every probe below is a weekday bar.
	for _, at := range []time.Time{
		time.Date(2026, 6, 1, 6, 30, 0, 0, msk),
		time.Date(2026, 6, 1, 17, 0, 0, 0, msk),
		time.Date(2026, 6, 1, 23, 30, 0, 0, msk),
	} {
		t.Run(at.Format("15:04"), func(t *testing.T) {
			s := NewWithParams("T", entryParams())
			md := entryFixture()
			shiftTo(&md, at)
			md = withDay(md, 10.0, 101, 100)
			if got := s.Decide(md); got.Kind != model.SignalBuy {
				t.Fatalf("Kind = %v, want Buy: the hour of a weekday bar must not gate the entry",
					got.Kind)
			}
		})
	}
}

// TestEnterCrossIsAnEventNotAState: once RSI already sits below the band on the PREVIOUS bar,
// there is no fresh cross and therefore no entry. withDay attaches a daily series so the daily-ATR
// gate (4) cannot mask the RSI gate (2) this case is meant to isolate — without it, dailyATR()
// is 0 and gate 4 alone would explain the None verdict regardless of the RSI-cross check.
func TestEnterCrossIsAnEventNotAState(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	// Append one more down bar: now bar i-1 is already below the band.
	last := md.Closes[len(md.Closes)-1] * 0.994
	md = barSeries(append(md.Closes, last), md.Times[0])
	shiftTo(&md, time.Date(2026, 6, 1, 12, 30, 0, 0, msk))
	md = withDay(md, 10.0, 101, 100)
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
	p := DefaultParams()
	// Both branches are armed here explicitly: this test is about the gate's geometry — two
	// windows and the dead band between them — not about which branch the shipped defaults
	// happen to enable. Collapsing a branch is a legitimate calibration choice (FreshDayATR = 0
	// disables the early one); TestDayStateGateDegradations covers those cases.
	p.UseDayATRGate, p.FreshDayATR, p.SpentDayATR = 1, 0.3, 0.8
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

	// Схлопывание одной ветки — рабочий приём калибровки (data/params/rsi_pullback/cal_day_spent.json),
	// а не вырожденный случай: FreshDayATR = 0 оставляет только «день исчерпан». Держится это на
	// условии `fresh > 0 &&` в dayStateOK: без него `used <= 0*atr` пропускало бы день с нулевым
	// диапазоном, то есть ровно самое начало дня — тот вход, который отключением и убирали.
	spentOnly := base
	spentOnly.UseDayATRGate, spentOnly.FreshDayATR, spentOnly.SpentDayATR = 1, 0, 0.8
	sSpent := NewWithParams("TEST", spentOnly)
	for _, used := range []float64{0, 1, 5, 7.9999} {
		md := strategy.MarketData{TodayHigh: 100 + used, TodayLow: 100}
		if sSpent.dayStateOK(md, 10) {
			t.Fatalf("FreshDayATR=0: день, прошедший %.4f ATR, обязан быть отклонён", used)
		}
	}
	if !sSpent.dayStateOK(strategy.MarketData{TodayHigh: 109, TodayLow: 100}, 10) {
		t.Fatal("FreshDayATR=0: исчерпанный день обязан проходить")
	}

	// Симметрично: SpentDayATR = 0 оставляет только ветку «день только начался».
	freshOnly := base
	freshOnly.UseDayATRGate, freshOnly.FreshDayATR, freshOnly.SpentDayATR = 1, 0.3, 0
	sFresh := NewWithParams("TEST", freshOnly)
	if !sFresh.dayStateOK(strategy.MarketData{TodayHigh: 101, TodayLow: 100}, 10) {
		t.Fatal("SpentDayATR=0: свежий день обязан проходить")
	}
	if sFresh.dayStateOK(strategy.MarketData{TodayHigh: 120, TodayLow: 100}, 10) {
		t.Fatal("SpentDayATR=0: исчерпанный день обязан быть отклонён")
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
// on a WEEKDAY — the only calendar condition enter() still imposes. pullbackCloses() has a fixed
// length, so the offset from start to the last bar is always exactly 404*30min = 8 days 10 hours:
// this Monday start puts the last bar on the Tuesday of the following week at 16:00. The hour no
// longer matters (see TestEnterHasNoTimeOfDayWindow); the weekday does.
var entryDailyATRStart = time.Date(2026, 3, 2, 6, 0, 0, 0, msk)

// shallowPullbackCloses is pullbackCloses with a THREE-bar shock instead of five: RSI(4) then
// reads 33.9 on the previous bar and 22.6 on the last, a clean downward cross of the shipped
// RSILower = 30 (the five-bar shock overshoots — it is already at 15.6 before the last bar, so
// no cross of 30 lands on bar i). Multipliers come from a sweep over indicators.RSISeries, not
// from eyeballing.
func shallowPullbackCloses() []float64 {
	const rise = 400
	out := make([]float64, 0, rise+3)
	p := 100.0
	for i := 0; i < rise; i++ {
		p *= 1.0008
		out = append(out, p)
	}
	for i := 0; i < 3; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// TestShippedDefaultsCanEnter is the counterweight to entryParams(): every other entry test pins
// its own band and day branch, so none of them would notice if DefaultParams drifted into a
// combination that can never fire at all (e.g. a day gate whose only armed branch contradicts the
// entry conditions). This one runs the defaults exactly as they ship — RSILower = 30, the
// spent-day branch, volume gate off — against a series and a day shaped to satisfy them, and
// requires an actual Buy.
func TestShippedDefaultsCanEnter(t *testing.T) {
	p := DefaultParams()
	md := barSeries(shallowPullbackCloses(), entryDailyATRStart)
	// used = 9 >= SpentDayATR(0.8) * 10 -> the "day is spent" branch, the only one armed by default.
	md = withDay(md, 10.0, 109, 100)
	if got := NewWithParams("T", p).Decide(md); got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy: the shipped defaults must be able to enter at all", got.Kind)
	}
}

func TestEnterSetsStopAndTargetFromDailyATR(t *testing.T) {
	p := entryParams()
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

// TestEnterRefusesNonPositiveStop: when StopDailyATR*dailyATR eats through the entire entry
// price, the computed stop lands at or below zero. manage() only ever checks pos.StopLoss > 0,
// so a non-positive stop here would silently hold a multi-day, multi-night position with no
// protective exit at all — the entry must be refused instead.
func TestEnterRefusesNonPositiveStop(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 1.0 // pinned here, not inherited: the case is about the stop maths, not the default
	s := NewWithParams("TEST", p)
	// entryFixture's last close is ~136 (see pullbackCloses); a daily ATR of 200 at
	// StopDailyATR=1.0 makes the stop distance dwarf the entry price.
	md := withDay(entryFixture(), 200.0, 101, 100)
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatalf("вход с неположительным стопом (ATR=200 » цена входа) должен быть запрещён, StopLoss=%.4f", sig.StopLoss)
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
		// EntryATR=0 гасит защитный уровень целиком: фикстура обслуживает тесты о ДРУГИХ
		// выходах, и пересчитанный стоп там только мешал бы. Тесты про сам стоп задают
		// EntryATR явно.
		EntryATR:              0,
		MaxFavorablePrice:     entryPrice,
		PrevMaxFavorablePrice: entryPrice,
		EntryTime:             last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}

// TestExitStopLoss covers the stop, including the exact-touch boundary: `low <= level` must
// fire when the low lands ON the level. A test that only puts the stop strictly inside the bar
// survives a shift of that comparison to `<`.
func TestExitStopLoss(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	s := NewWithParams("T", p)
	tests := []struct {
		name  string
		lowAt func(level float64) float64
	}{
		{"low pierces the stop", func(level float64) float64 { return level * 0.999 }},
		{"low touches the stop exactly", func(level float64) float64 { return level }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
			md := barSeries([]float64{100, 100, 100, 100}, start)
			i := len(md.Closes) - 1
			// entry 100, EntryATR 10, StopDailyATR 0.5 -> уровень 95.
			const level = 95.0
			md.Lows[i] = tc.lowAt(level)
			md.Position = &strategy.Position{
				PurchasePrice: 100, Quantity: 1, StopLoss: level,
				EntryATR: 10, MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100,
				EntryTime: md.Times[0],
			}
			got := s.Decide(md)
			if got.Kind != model.SignalSell || got.Reason != "SL" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/SL", got.Kind, got.Reason)
			}
			if math.Abs(got.StopLoss-level) > 1e-9 {
				t.Fatalf("StopLoss = %v, want %v", got.StopLoss, level)
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
	// The stop argument is inert here: withPosition zeroes EntryATR, which disables the
	// protective level in desiredStop regardless of this value, so only the RSI exit can fire.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI", got.Kind, got.Reason)
	}
}

func TestExitStopWinsOverRSIOnTheSameBar(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	s := NewWithParams("T", p)
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// entry близко к текущей цене (не 0.97, как в других фикстурах этого файла): при 3%
	// дисконте entry падает НИЖЕ low бара (у upperCrossFixture low = close*0.997), и тогда
	// формула ниже требует отрицательный ATR, который desiredStop трактует как «стоп
	// выключен» — SL молчал бы. Уровень при этом всё равно алгебраически равен low*1.0001
	// независимо от entry; меняется только знак ATR.
	entry := md.Closes[i] * 0.999
	// EntryATR подобран так, чтобы уровень стопа лёг внутрь бара: level = entry - 0.5*atr.
	atr := (entry - md.Lows[i]*1.0001) / 0.5
	md = withPosition(md, entry, 0, 2)
	md.Position.EntryATR = atr
	md.Position.MaxFavorablePrice, md.Position.PrevMaxFavorablePrice = entry, entry
	if got := s.Decide(md); got.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL to take precedence over RSI", got.Reason)
	}
}

func TestExitTakeProfit(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	i := len(md.Closes) - 1
	md.Highs[i] = 110
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 106,
		EntryTime: md.Times[0],
	}
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 106 {
		t.Fatalf("TakeProfit = %.4f, want 106 (уровень из позиции)", sig.TakeProfit)
	}
}

func TestExitStopWinsOverTakeProfitOnTheSameBar(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	i := len(md.Closes) - 1
	md.Highs[i], md.Lows[i] = 110, 90 // бар задевает и цель, и стоп
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 95, TakeProfit: 105,
		// entry 100, EntryATR 10, StopDailyATR 0.5 -> уровень 95, low 90 его пробивает.
		EntryATR: 10, MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL: внутрибарный порядок неизвестен, побеждает худший исход", sig.Reason)
	}
}

func TestExitTakeProfitDisabledAtZero(t *testing.T) {
	p := DefaultParams()
	p.TPDailyATR = 0
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	md.Highs[len(md.Highs)-1] = 500
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 0,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell && sig.Reason == "TP" {
		t.Fatal("цель выключена (TakeProfit=0), выхода по TP быть не должно")
	}
}

func TestPositionSurvivesOvernight(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	// Понедельник 22:30 -> вторник 07:00: смена календарного дня и разрыв в серии.
	closes := []float64{100, 100, 100, 100}
	md := barSeries(closes, time.Date(2026, 3, 2, 21, 30, 0, 0, msk))
	md.Times[len(md.Times)-1] = time.Date(2026, 3, 3, 7, 0, 0, 0, msk)
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 110,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("позиция закрыта на переходе через ночь (%q), а перенос разрешён", sig.Reason)
	}
}

func TestPositionSurvivesWeekend(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	// Пятница -> понедельник.
	md := barSeries([]float64{100, 100, 100, 100}, time.Date(2026, 3, 6, 21, 30, 0, 0, msk))
	md.Times[len(md.Times)-1] = time.Date(2026, 3, 9, 7, 0, 0, 0, msk)
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 110,
		EntryTime: md.Times[0],
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("позиция закрыта на переходе через выходные (%q)", sig.Reason)
	}
}

func TestExplainMentionsEveryGate(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := withDay(barSeries(pullbackCloses(), start), 10.0, 101, 100)
	got := s.Explain(md)
	for _, want := range []string{"день", "RSI", "EMA", "дневной ATR", "состояние дня", "фон объёмов", "стоп", "цель"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Explain не упоминает %q:\n%s", want, got)
		}
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

// sliceMDFull is sliceMD plus a cut of the daily series: it drops the oldest dailyFrom entries
// from DailyHighs/DailyLows/DailyCloses/DailyTimes. TodayHigh/TodayLow are left untouched — they
// describe the CURRENT day's extent and do not shrink when older completed-day history is
// trimmed away.
func sliceMDFull(md strategy.MarketData, from, to, dailyFrom int) strategy.MarketData {
	out := sliceMD(md, from, to)
	if dailyFrom > 0 && dailyFrom <= len(md.DailyCloses) {
		out.DailyHighs = md.DailyHighs[dailyFrom:]
		out.DailyLows = md.DailyLows[dailyFrom:]
		out.DailyCloses = md.DailyCloses[dailyFrom:]
		out.DailyTimes = md.DailyTimes[dailyFrom:]
	}
	return out
}

// TestNoLookaheadAcrossWindowCuts is the load-bearing safety net: the decision on bar i must not
// depend on how much history precedes it. Cuts stay far enough from bar i that every indicator
// is warmed in both windows. The stop/target are still compared with a relative tolerance for
// robustness, even though the daily ATR they derive from is untouched by sliceMD (it lives in
// DailyHighs/DailyLows/DailyCloses, none of which sliceMD trims) and is therefore identical
// across every cut.
func TestNoLookaheadAcrossWindowCuts(t *testing.T) {
	s := NewWithParams("T", entryParams())
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

	// The daily series feeds the daily ATR, which sizes the stop/target and both day-gate
	// thresholds. Cutting it alongside the intraday window (dropping older completed weekday
	// days, TodayHigh/TodayLow held fixed) must not move the verdict on the same last bar. This
	// is NOT because indicators.ATR only reads a fixed-size tail of its input: it seeds at index
	// `period` of WHATEVER slice it is given and Wilder-smooths across the full remaining length,
	// so in general a longer daily history does shift the value. It is invariant here only
	// because withDay's fixture (dailyBars) gives every weekday day the exact same true range —
	// with a constant input, Wilder's EWMA has already converged on the very first smoothing
	// step, so feeding it more identical days changes nothing. This is not lookahead either way:
	// the engine always feeds the same daily history for a given bar regardless of how the test
	// slices it.
	for _, dailyFrom := range []int{0, 5, 10} {
		got := s.Decide(sliceMDFull(full, 160, n, dailyFrom))
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("daily cut at %d gave %v/%q, full window gave %v/%q",
				dailyFrom, got.Kind, got.Reason, want.Kind, want.Reason)
		}
	}
}

// TestNoLookaheadWithOpenPosition covers the manage() path the same way.
func TestNoLookaheadWithOpenPosition(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := upperCrossFixture()
	n := len(full.Closes)
	i := n - 1
	// The stop argument is inert: withPosition zeroes EntryATR, which disables the protective
	// level regardless of this value.
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

// volSeries builds `days` weekday days of `perDay` 30-minute bars starting at 07:00 MSK, with
// a U-shaped volume profile: the first bar of each day is `openVol`, the rest are `midVol`.
// The last bar of the last day is the "current" bar.
func volSeries(firstDay time.Time, days, perDay int, openVol, midVol int64) strategy.MarketData {
	var md strategy.MarketData
	d := firstDay
	for added := 0; added < days; {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			d = d.AddDate(0, 0, 1)
			continue
		}
		for b := 0; b < perDay; b++ {
			t := time.Date(d.Year(), d.Month(), d.Day(), 7, 0, 0, 0, msk).
				Add(time.Duration(b) * 30 * time.Minute)
			v := midVol
			if b == 0 {
				v = openVol
			}
			md.Times = append(md.Times, t)
			md.Volumes = append(md.Volumes, v)
			md.Closes = append(md.Closes, 100)
			md.Highs = append(md.Highs, 100.3)
			md.Lows = append(md.Lows, 99.7)
		}
		added++
		d = d.AddDate(0, 0, 1)
	}
	md.Price = 100
	return md
}

func TestVolumeGateComparesAgainstItsOwnSlot(t *testing.T) {
	p := DefaultParams()
	// The gate is armed explicitly: these cases test its mechanics, not whether it ships on.
	p.UseVolume, p.VolBaseDays, p.VolLookbackBars, p.VolMult = 1, 5, 3, 1.5
	s := NewWithParams("TEST", p)

	// Профиль: открытие 10000, середина дня 1000. Текущий бар — середина дня с объёмом 2000:
	// это вдвое выше своего слота (1000), но впятеро ниже утреннего.
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)
	last := len(md.Volumes) - 1
	md.Volumes[last] = 2000
	md.Volumes[last-1], md.Volumes[last-2] = 1000, 1000

	// Слотовая база для 10:30 равна 1000, значит 2000 — это 2.0x, гейт открыт.
	// На плоской базе тот же бар НЕ прошёл бы: она равна (10000 + 7*1000)/8 = 2125,
	// и 2000/2125 = 0.94 < 1.5. Именно это различает две реализации.
	if !s.volumeOK(md) {
		t.Fatal("бар вдвое выше своего слота обязан открывать гейт (плоская база дала бы отказ)")
	}

	md.Volumes[last] = 1000
	if s.volumeOK(md) {
		t.Fatal("бар ровно на уровне своего слота не должен открывать гейт при VolMult=1.5")
	}
}

func TestVolumeGateAnyOfTheLastThreeBars(t *testing.T) {
	p := DefaultParams()
	// The gate is armed explicitly: these cases test its mechanics, not whether it ships on.
	p.UseVolume, p.VolBaseDays, p.VolLookbackBars, p.VolMult = 1, 5, 3, 1.5
	s := NewWithParams("TEST", p)
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)
	last := len(md.Volumes) - 1

	md.Volumes[last], md.Volumes[last-1], md.Volumes[last-2] = 1000, 1000, 1000
	if s.volumeOK(md) {
		t.Fatal("три тихих бара не должны открывать гейт")
	}
	md.Volumes[last-2] = 2000 // всплеск на третьем баре назад
	if !s.volumeOK(md) {
		t.Fatal("всплеск на любом из последних трёх баров обязан открывать гейт")
	}
	md.Volumes[last-2] = 1000
	md.Volumes[last-3] = 5000 // четвёртый бар назад — уже вне окна
	if s.volumeOK(md) {
		t.Fatal("всплеск за пределами VolLookbackBars не должен открывать гейт")
	}
}

func TestVolumeGateIgnoresWeekendBars(t *testing.T) {
	p := DefaultParams()
	// The gate is armed explicitly: these cases test its mechanics, not whether it ships on.
	p.UseVolume, p.VolBaseDays, p.VolLookbackBars, p.VolMult = 1, 5, 3, 1.5
	s := NewWithParams("TEST", p)
	const perDay = 8
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, perDay, 10000, 1000)

	// Текущий бар — 1.4x своего слота: ниже порога 1.5, гейт закрыт.
	last := len(md.Volumes) - 1
	md.Volumes[last] = 1400
	md.Volumes[last-1], md.Volumes[last-2] = 1000, 1000
	if s.volumeOK(md) {
		t.Fatal("1.4x от слотовой базы не должно открывать гейт при VolMult=1.5")
	}

	// Вклеиваем тонкую субботнюю сессию (те же слоты, объём 50) перед последним днём.
	// Если бы выходные попадали в базу, суббота заняла бы один из VolBaseDays=5 слотов
	// дневного счётчика и вытеснила бы будний день: слотовая база упала бы с 1000 до
	// (4*1000+50)/5 = 810, и тот же бар дал бы 1400/810 ≈ 1.73 ≥ 1.5 — гейт бы открылся.
	// Он открыться не должен.
	sat := time.Date(2026, 3, 7, 7, 0, 0, 0, msk)
	insertAt := len(md.Times) - perDay
	for b := 0; b < perDay; b++ {
		bt := sat.Add(time.Duration(b) * 30 * time.Minute)
		md.Times = append(md.Times[:insertAt], append([]time.Time{bt}, md.Times[insertAt:]...)...)
		md.Volumes = append(md.Volumes[:insertAt], append([]int64{50}, md.Volumes[insertAt:]...)...)
		md.Closes = append(md.Closes[:insertAt], append([]float64{100}, md.Closes[insertAt:]...)...)
		md.Highs = append(md.Highs[:insertAt], append([]float64{100.3}, md.Highs[insertAt:]...)...)
		md.Lows = append(md.Lows[:insertAt], append([]float64{99.7}, md.Lows[insertAt:]...)...)
		insertAt++
	}
	if s.volumeOK(md) {
		t.Fatal("выходная сессия занизила базу: выходные обязаны выпадать из расчёта")
	}

	// И выходные не должны занимать места в окне последних трёх баров: всплеск на третьем
	// БУДНЕМ баре назад обязан открывать гейт.
	md.Volumes[len(md.Volumes)-1] = 1000
	md.Volumes[len(md.Volumes)-3] = 2000
	if !s.volumeOK(md) {
		t.Fatal("выходные бары не должны вытеснять будние из окна VolLookbackBars")
	}
}

// TestVolumeGateSkipsBrokenReadingsInsteadOfBlocking: bars with non-positive volume are missing
// data, not "checked and quiet" bars. Before the fix, each such bar still consumed a slot of
// VolLookbackBars, so a run of broken readings could exhaust the window and block the entry
// outright even with a genuinely busy bar sitting right behind them — violating volumeOK's own
// "missing volume must never block an entry" contract.
func TestVolumeGateSkipsBrokenReadingsInsteadOfBlocking(t *testing.T) {
	p := DefaultParams()
	// The gate is armed explicitly: these cases test its mechanics, not whether it ships on.
	p.UseVolume, p.VolBaseDays, p.VolLookbackBars, p.VolMult = 1, 5, 3, 1.5
	s := NewWithParams("TEST", p)
	md := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)
	last := len(md.Volumes) - 1

	md.Volumes[last] = 0
	md.Volumes[last-1] = -5
	md.Volumes[last-2] = 0
	md.Volumes[last-3] = 2000 // 2x its own slot base (1000): would open the gate if reached.
	if !s.volumeOK(md) {
		t.Fatal("нулевые/битые объёмы не должны занимать место в окне и блокировать вход")
	}
}

func TestVolumeGateDegradations(t *testing.T) {
	base := DefaultParams()
	// Armed explicitly: each case below turns exactly one knob off, so the baseline must be on.
	base.UseVolume, base.VolBaseDays, base.VolLookbackBars, base.VolMult = 1, 5, 3, 1.5
	quiet := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 6, 8, 10000, 1000)

	off := base
	off.UseVolume = 0
	if !NewWithParams("TEST", off).volumeOK(quiet) {
		t.Fatal("выключенный гейт обязан пропускать")
	}

	for name, mutate := range map[string]func(p *Params){
		"VolBaseDays=0":     func(p *Params) { p.VolBaseDays = 0 },
		"VolLookbackBars=0": func(p *Params) { p.VolLookbackBars = 0 },
		"VolMult=0":         func(p *Params) { p.VolMult = 0 },
	} {
		p := base
		mutate(&p)
		if !NewWithParams("TEST", p).volumeOK(quiet) {
			t.Fatalf("%s: сломанная конфигурация обязана пропускать", name)
		}
	}

	s := NewWithParams("TEST", base)
	if !s.volumeOK(strategy.MarketData{}) {
		t.Fatal("без объёмов гейт обязан пропускать")
	}
	noTimes := quiet
	noTimes.Times = nil
	if !s.volumeOK(noTimes) {
		t.Fatal("без времён гейт обязан пропускать")
	}
	// Только один день истории: базы нет вовсе.
	oneDay := volSeries(time.Date(2026, 3, 2, 0, 0, 0, 0, msk), 1, 8, 10000, 1000)
	if !s.volumeOK(oneDay) {
		t.Fatal("без завершённых дней в базе гейт обязан пропускать")
	}
}

func TestLookbackCoversVolumeBaseline(t *testing.T) {
	p := DefaultParams()
	p.UseVolume, p.VolBaseDays = 1, 10
	if got := NewWithParams("TEST", p).Lookback(); got < 11*maxBarsPerDay {
		t.Fatalf("Lookback = %d, want >= %d (11 дней по %d баров)", got, 11*maxBarsPerDay, maxBarsPerDay)
	}
	p.UseVolume = 0
	if got := NewWithParams("TEST", p).Lookback(); got >= 11*maxBarsPerDay {
		t.Fatalf("Lookback = %d: с выключенным гейтом окно не должно раздуваться", got)
	}
}

// TestDesiredStopBindsTheNearestLevel фиксирует главное правило уровня: среди активных
// компонентов связывает ЧИСЛЕННО БОЛЬШИЙ — он ближе к цене, и падающая цена коснётся его
// первым. Таблица заодно пинует каждое условие отключения по отдельности: тест, который
// проверяет только «трейл выше SL», переживает удаление любой из guard-веток.
func TestDesiredStopBindsTheNearestLevel(t *testing.T) {
	base := DefaultParams()
	base.StopDailyATR = 0.5
	base.UseTrail = 1
	base.TrailDailyATR = 1.0

	tests := []struct {
		name       string
		mutate     func(p *Params)
		entry      float64
		dailyATR   float64
		maxFav     float64
		wantLevel  float64
		wantReason string
	}{
		{
			name:       "трейл ещё ниже SL — связывает SL",
			entry:      100,
			dailyATR:   10,
			maxFav:     100, // трейл = 100-10 = 90, SL = 100-5 = 95
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "цена ушла вверх — трейл перехватывает",
			entry:      100,
			dailyATR:   10,
			maxFav:     112, // трейл = 112-10 = 102 > SL 95
			wantLevel:  102,
			wantReason: "TRAIL",
		},
		{
			name:       "трейл ровно на уровне SL — SL удерживает связь",
			entry:      100,
			dailyATR:   10,
			maxFav:     105, // трейл = 95 == SL 95, строгое > не срабатывает
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "UseTrail выключен",
			mutate:     func(p *Params) { p.UseTrail = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "TrailDailyATR=0 при включённом UseTrail",
			mutate:     func(p *Params) { p.TrailDailyATR = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "maxFav=0 — трейлить не от чего",
			entry:      100,
			dailyATR:   10,
			maxFav:     0,
			wantLevel:  95,
			wantReason: "SL",
		},
		{
			name:       "дневной ATR не посчитан — стопов нет вовсе",
			entry:      100,
			dailyATR:   0,
			maxFav:     112,
			wantLevel:  0,
			wantReason: "",
		},
		{
			name:       "SL отключён, трейл несёт уровень один",
			mutate:     func(p *Params) { p.StopDailyATR = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  102,
			wantReason: "TRAIL",
		},
		{
			name:       "оба отключены",
			mutate:     func(p *Params) { p.StopDailyATR = 0; p.UseTrail = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     112,
			wantLevel:  0,
			wantReason: "",
		},
		{
			name:       "уровень провалился в ноль — не пол, а голый лонг",
			mutate:     func(p *Params) { p.StopDailyATR = 20; p.UseTrail = 0 },
			entry:      100,
			dailyATR:   10,
			maxFav:     100,
			wantLevel:  0,
			wantReason: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			if tc.mutate != nil {
				tc.mutate(&p)
			}
			level, reason := desiredStop(p, tc.entry, tc.dailyATR, tc.maxFav)
			if math.Abs(level-tc.wantLevel) > 1e-9 || reason != tc.wantReason {
				t.Fatalf("desiredStop = (%.4f, %q), want (%.4f, %q)", level, reason, tc.wantLevel, tc.wantReason)
			}
		})
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

// trailFixture строит позицию с включённым трейлом на плоской серии, где всё, кроме
// low последнего бара и двух максимумов позиции, зафиксировано. entry=100, EntryATR=10.
func trailFixture(p Params, low, maxFav, prevMaxFav float64) (*Strategy, strategy.MarketData) {
	s := NewWithParams("TEST", p)
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 105}, start)
	i := len(md.Closes) - 1
	md.Lows[i], md.Highs[i], md.Closes[i] = low, 105, 105
	md.Price = 105
	md.Position = &strategy.Position{
		PurchasePrice:         100,
		Quantity:              1,
		StopLoss:              0, // фиксированный SL выключен: изолируем трейл
		TakeProfit:            0,
		EntryATR:              10,
		MaxFavorablePrice:     maxFav,
		PrevMaxFavorablePrice: prevMaxFav,
		EntryTime:             md.Times[0],
	}
	return s, md
}

// TestTrailReadsPrevMaxFavorable — регресс на lookahead. Движок марк-ту-маркетит позицию
// ДО Decide, поэтому MaxFavorablePrice уже знает close текущего бара, а стоп проверяется по
// low того же бара. Отсчёт трейла от MaxFavorablePrice выдал бы уровень, которого биржевой
// ордер в момент прохода low знать не мог. Тест падает при подмене поля.
func TestTrailReadsPrevMaxFavorable(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseTrail = 1
	p.TrailDailyATR = 0.5 // трейл = maxFav - 5
	p.UseRSIExit = 0      // изолируем стоп от RSI-ветки

	// Бар: low 96, close 105. maxFav после mark = 105 -> уровень 100, low 96 его пробил бы.
	// prevMaxFav = 100 -> уровень 95, low 96 его НЕ достаёт.
	s, md := trailFixture(p, 96, 105, 100)
	got := s.Decide(md)
	if got.Kind == model.SignalSell {
		t.Fatalf("выход %q по уровню от MaxFavorablePrice: трейл обязан считаться от "+
			"PrevMaxFavorablePrice (95), а low 96 его не достаёт", got.Reason)
	}
}

// TestExitTrailFiresOnLow проверяет само срабатывание, включая точное касание: `low <= level`
// должно сработать, когда low ложится РОВНО на уровень. Тест только со строгим пробоем
// пережил бы замену сравнения на `<`.
func TestExitTrailFiresOnLow(t *testing.T) {
	tests := []struct {
		name string
		low  float64
	}{
		{"low пробил трейл", 94.9},
		{"low коснулся трейла ровно", 95},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.StopDailyATR = 0
			p.UseTrail = 1
			p.TrailDailyATR = 0.5
			p.UseRSIExit = 0

			// prevMaxFav = 100, EntryATR = 10 -> уровень 100 - 0.5*10 = 95.
			s, md := trailFixture(p, tc.low, 100, 100)
			got := s.Decide(md)
			if got.Kind != model.SignalSell || got.Reason != "TRAIL" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/TRAIL", got.Kind, got.Reason)
			}
			if math.Abs(got.StopLoss-95) > 1e-9 {
				t.Fatalf("StopLoss = %v, want 95: движок заливает по min(level, open), "+
					"и без уровня в сигнале он зальёт по close", got.StopLoss)
			}
		})
	}
}

// TestTrailNeverGoesBelowSL: широкий трейл не должен ослаблять фиксированный стоп.
func TestTrailNeverGoesBelowSL(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0.5 // SL = 100 - 5 = 95
	p.UseTrail = 1
	p.TrailDailyATR = 3.0 // трейл = 100 - 30 = 70, сильно ниже
	p.UseRSIExit = 0

	s, md := trailFixture(p, 94.9, 100, 100)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/SL: широкий трейл не ослабляет стоп",
			got.Kind, got.Reason)
	}
}

// TestTrailDisabledWithoutEntryATR — live-guard: без ATR на входе трейл не считается.
func TestTrailDisabledWithoutEntryATR(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseTrail = 1
	p.TrailDailyATR = 0.5
	p.UseRSIExit = 0

	s, md := trailFixture(p, 50, 100, 100)
	md.Position.EntryATR = 0
	if got := s.Decide(md); got.Kind == model.SignalSell {
		t.Fatalf("выход %q без EntryATR: трейл обязан молчать, когда линейка не известна",
			got.Reason)
	}
}

// TestTrailWinsOverTakeProfitOnTheSameBar: приоритет стопа над целью держится и для трейла.
func TestTrailWinsOverTakeProfitOnTheSameBar(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseTrail = 1
	p.TrailDailyATR = 0.5
	p.UseRSIExit = 0

	s, md := trailFixture(p, 94.9, 100, 100)
	i := len(md.Closes) - 1
	md.Highs[i] = 130
	md.Position.TakeProfit = 120 // бар задевает и цель, и трейл
	if got := s.Decide(md); got.Reason != "TRAIL" {
		t.Fatalf("Reason = %q, want TRAIL: внутрибарный порядок неизвестен, побеждает худший исход",
			got.Reason)
	}
}

// TestUseRSIExitZeroKeepsPosition: при UseRSIExit=0 крест RSI вверх позицию не закрывает.
func TestUseRSIExitZeroKeepsPosition(t *testing.T) {
	p := DefaultParams()
	p.UseRSIExit = 0
	s := NewWithParams("T", p)
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// The stop argument is inert: withPosition zeroes EntryATR, which disables the protective
	// level regardless of this value.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind == model.SignalSell {
		t.Fatalf("выход %q при UseRSIExit=0: RSI-выход должен быть отключён", got.Reason)
	}
}

// TestDefaultsPreserveLegacyExits: на дефолтах трейла нет, RSI-выход есть — то же поведение,
// что до появления обоих параметров. Это страховка совместимости уже откалиброванных наборов
// по тикерам, в которых новых ключей нет и которые поэтому получат дефолты.
func TestDefaultsPreserveLegacyExits(t *testing.T) {
	s := NewWithParams("T", DefaultParams())

	// RSI-выход работает.
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// The stop argument is inert: withPosition zeroes EntryATR, which disables the protective
	// level regardless of this value.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI на дефолтах", got.Kind, got.Reason)
	}

	// Трейла нет: позиция глубоко в прибыли, но проседание её не закрывает.
	p := DefaultParams()
	p.StopDailyATR = 0
	p.UseRSIExit = 0
	s2, md2 := trailFixture(p, 50, 200, 200)
	if got := s2.Decide(md2); got.Kind == model.SignalSell {
		t.Fatalf("выход %q на дефолтах: UseTrail=0, трейла быть не должно", got.Reason)
	}
}

// TestTrailIsAStopReason пинует неявную связь со движком: "TRAIL" обязан числиться стоповой
// причиной, иначе бэктест зальёт выход по close и потеряет модель гэпа min(level, open).
func TestTrailIsAStopReason(t *testing.T) {
	if !model.IsStopReason("TRAIL") {
		t.Fatal(`model.IsStopReason("TRAIL") = false: движок зальёт трейл по close`)
	}
}
