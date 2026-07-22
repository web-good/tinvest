package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// mskAt builds an MSK bar open-time for the given date and clock.
func mskAt(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, mskLoc)
}

func TestInSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		// 2026-07-20 is a Monday, 2026-07-24 a Friday, 2026-07-25 a Saturday.
		{"mon before open", mskAt(2026, 7, 20, 7, 55), false},
		{"mon at open", mskAt(2026, 7, 20, 8, 0), true},
		{"mon midday", mskAt(2026, 7, 20, 12, 30), true},
		{"mon last bar", mskAt(2026, 7, 20, 16, 55), true},
		{"mon at close", mskAt(2026, 7, 20, 17, 0), false},
		{"mon after close", mskAt(2026, 7, 20, 17, 5), false},
		{"fri before friday close", mskAt(2026, 7, 24, 13, 55), true},
		{"fri at friday close", mskAt(2026, 7, 24, 14, 0), false},
		{"fri after friday close", mskAt(2026, 7, 24, 16, 0), false},
		{"saturday", mskAt(2026, 7, 25, 12, 0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.inSession(tc.at); got != tc.want {
				t.Fatalf("inSession(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestInSessionZeroTimeIsPermissive(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if !s.inSession(time.Time{}) {
		t.Fatalf("zero time must not block the entry gate")
	}
}

func TestIsDayEnd(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"mon midday", mskAt(2026, 7, 20, 12, 0), false},
		{"mon 16:50", mskAt(2026, 7, 20, 16, 50), false},
		{"mon 16:55 last bar closes at 17:00", mskAt(2026, 7, 20, 16, 55), true},
		{"mon 17:05", mskAt(2026, 7, 20, 17, 5), true},
		{"fri 13:50", mskAt(2026, 7, 24, 13, 50), false},
		{"fri 13:55 last bar closes at 14:00", mskAt(2026, 7, 24, 13, 55), true},
		{"saturday always day end", mskAt(2026, 7, 25, 10, 0), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isDayEnd(tc.at); got != tc.want {
				t.Fatalf("isDayEnd(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestIsDayEndZeroTimeIsNoOp(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if s.isDayEnd(time.Time{}) {
		t.Fatalf("zero time must degrade the EOD exit to a no-op")
	}
}

func TestBarTimeRequiresAlignedTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{
		Closes: []float64{1, 2, 3},
		Times:  []time.Time{mskAt(2026, 7, 20, 10, 0)}, // misaligned on purpose
	}
	if got := s.barTime(md); !got.IsZero() {
		t.Fatalf("misaligned Times must yield zero time, got %v", got)
	}
	md.Times = []time.Time{
		mskAt(2026, 7, 20, 10, 0),
		mskAt(2026, 7, 20, 10, 5),
		mskAt(2026, 7, 20, 10, 10),
	}
	if got := s.barTime(md); !got.Equal(mskAt(2026, 7, 20, 10, 10)) {
		t.Fatalf("barTime = %v want the latest bar time", got)
	}
}

func TestLookbackCoversIndicatorWarmup(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if got := s.Lookback(); got != 120 {
		t.Fatalf("Lookback = %d want 120", got)
	}
}

func TestTickerRoundTrip(t *testing.T) {
	if got := NewWithParams("SBER", DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q want SBER", got)
	}
}

// testParams are permissive defaults for entry tests: the risk sanity bounds are opened
// up so synthetic series never trip them, and the stop sits exactly on the trigger low.
func testParams() Params {
	p := DefaultParams()
	p.MinRiskATR = 0
	p.MaxRiskATR = 100
	p.StopBufferATR = 0
	return p
}

// declineThenRally builds a synthetic 5m series: `down` bars falling by step from start,
// then `up` bars rising by 2*step. Each bar's high/low is close +/- wick, so lows rise
// monotonically during the rally (the trigger is never invalidated by the rally itself).
//
// Diagnosed while making TestEntryFiresOnRSIThenMACDCross pass (see task-2 report): a
// perfectly monotonic decline forces Wilder's RSI to exactly 0 for every bar past warm-up
// (avgGain never leaves 0), so the bar immediately before the reversal always has rsi==0.
// lastRSITrigger's rsi[j-1] > 0 guard then can't tell that real (but degenerate) zero apart
// from a warm-up zero and refuses to trigger — exactly the behavior TestNoEntryFromWarmupZeros
// pins down. A small counter-trend tick every 7th decline bar keeps RSI inside (0, oversold)
// throughout the decline without ever crossing back above it, so the guard sees a genuine
// non-zero "was oversold" reading on the bar before the rally. RSIOversold/MACDConfirmBars
// are untouched; only this synthetic-series shape changed.
func declineThenRally(down, up int, start, step, wick float64) (highs, lows, closes []float64) {
	price := start
	for i := 0; i < down; i++ {
		if i%7 == 6 {
			price += step * 0.25
		} else {
			price -= step
		}
		closes = append(closes, price)
	}
	for i := 0; i < up; i++ {
		price += 2 * step
		closes = append(closes, price)
	}
	for _, c := range closes {
		highs = append(highs, c+wick)
		lows = append(lows, c-wick)
	}
	return highs, lows, closes
}

// sessionTimes returns len(n) MSK bar open-times, 5 minutes apart, starting Mon 10:00.
func sessionTimes(n int) []time.Time {
	out := make([]time.Time, n)
	base := mskAt(2026, 7, 20, 10, 0)
	for i := range out {
		out[i] = base.Add(time.Duration(i*barSpanMin) * time.Minute)
	}
	return out
}

// mdPrefix builds MarketData over bars [0, i] of the given series (flat position).
func mdPrefix(highs, lows, closes []float64, times []time.Time, i int) strategy.MarketData {
	return strategy.MarketData{
		Price:  closes[i],
		Highs:  highs[:i+1],
		Lows:   lows[:i+1],
		Closes: closes[:i+1],
		Times:  times[:i+1],
	}
}

// firstBuy walks the series bar by bar and returns the first Buy signal and its index.
func firstBuy(s *Strategy, highs, lows, closes []float64, times []time.Time) (model.Signal, int, bool) {
	for i := 40; i < len(closes); i++ {
		sig := s.Decide(mdPrefix(highs, lows, closes, times, i))
		if sig.Kind == model.SignalBuy {
			return sig, i, true
		}
	}
	return model.Signal{}, 0, false
}

func TestEntryFiresOnRSIThenMACDCross(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	sig, i, ok := firstBuy(s, highs, lows, closes, times)
	if !ok {
		t.Fatalf("no entry fired on a decline-then-rally series")
	}
	if i < 60 || i > 66 {
		t.Fatalf("entry at bar %d, want it inside the rally (60..66)", i)
	}

	// The MACD cross must be on the entry bar and both lines below zero.
	macd, signal := indicators.MACD(closes[:i+1], 3, 6, 9)
	if !(macd[i-1] <= signal[i-1] && macd[i] > signal[i]) {
		t.Fatalf("entry bar %d is not a MACD bullish cross", i)
	}
	if macd[i] >= 0 || signal[i] >= 0 {
		t.Fatalf("MACD lines must be below zero at entry: macd=%v signal=%v", macd[i], signal[i])
	}

	// The stop is the low of the RSI-cross bar, which lies within the confirm window.
	rsi := indicators.RSISeries(closes[:i+1], s.p.RSIPeriod)
	trig, found := s.lastRSITrigger(rsi, i+1)
	if !found {
		t.Fatalf("no RSI trigger found for entry bar %d", i)
	}
	if math.Abs(sig.StopLoss-lows[trig]) > 1e-9 {
		t.Fatalf("stop = %v want low of the RSI bar %v", sig.StopLoss, lows[trig])
	}
	wantTP := closes[i] + s.p.RR*(closes[i]-sig.StopLoss)
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("tp = %v want %v", sig.TakeProfit, wantTP)
	}
	if sig.EntryReason == "" {
		t.Fatalf("EntryReason must be filled for the trade journal")
	}
}

func TestNoEntryOutsideSession(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	// Same series, but every bar sits on a Saturday.
	times := make([]time.Time, len(closes))
	base := mskAt(2026, 7, 25, 10, 0)
	for i := range times {
		times[i] = base.Add(time.Duration(i*barSpanMin) * time.Minute)
	}
	s := NewWithParams("TEST", testParams())
	if _, _, ok := firstBuy(s, highs, lows, closes, times); ok {
		t.Fatalf("entry fired on a Saturday")
	}
}

func TestEveryEntryHasBothMACDLinesBelowZero(t *testing.T) {
	// A long rally eventually drags MACD above zero; no Buy may fire on such a cross.
	highs, lows, closes := declineThenRally(60, 40, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	var buys int
	for i := 40; i < len(closes); i++ {
		sig := s.Decide(mdPrefix(highs, lows, closes, times, i))
		if sig.Kind != model.SignalBuy {
			continue
		}
		buys++
		macd, signal := indicators.MACD(closes[:i+1], 3, 6, 9)
		if macd[i] >= 0 || signal[i] >= 0 {
			t.Fatalf("Buy at bar %d with MACD above zero: macd=%v signal=%v", i, macd[i], signal[i])
		}
	}
	if buys == 0 {
		t.Fatalf("no Buy fired at all; the assertion above would be vacuous")
	}
}

func TestLastRSITriggerWindowBoundary(t *testing.T) {
	p := testParams()
	p.RSIOversold = 30
	p.MACDConfirmBars = 3
	s := NewWithParams("TEST", p)

	// rsi[3] crosses up through 30 (25 -> 40). n-1 is the current bar.
	rsi := []float64{20, 22, 25, 40, 45, 50, 55, 60}
	if idx, ok := s.lastRSITrigger(rsi, 7); !ok || idx != 3 { // current bar 6 -> t=3 is the edge
		t.Fatalf("cross at the window edge: idx=%d ok=%v want 3,true", idx, ok)
	}
	if _, ok := s.lastRSITrigger(rsi, 8); ok { // current bar 7 -> t=3 is one bar too old
		t.Fatalf("cross one bar past the window must not be accepted")
	}
}

func TestLastRSITriggerPicksMostRecent(t *testing.T) {
	p := testParams()
	p.RSIOversold = 30
	p.MACDConfirmBars = 3
	s := NewWithParams("TEST", p)
	// Two crosses: at index 1 and at index 3. The most recent one wins.
	rsi := []float64{20, 40, 25, 45, 50}
	if idx, ok := s.lastRSITrigger(rsi, 5); !ok || idx != 3 {
		t.Fatalf("idx=%d ok=%v want 3,true", idx, ok)
	}
}

func TestNoEntryFromWarmupZeros(t *testing.T) {
	s := NewWithParams("TEST", testParams())
	// RSISeries fills warm-up positions with 0. A zero is below any oversold level, so a
	// naive comparison would read bar 1 as "crossed up out of the zone".
	rsi := []float64{0, 55, 60, 62}
	if idx, ok := s.lastRSITrigger(rsi, 4); ok {
		t.Fatalf("warm-up zero produced a phantom trigger at %d", idx)
	}
}

func TestTriggerInvalidatedByStopBreak(t *testing.T) {
	s := NewWithParams("TEST", testParams())
	md := strategy.MarketData{
		Lows:   []float64{10, 9, 8.5, 9.5},
		Closes: []float64{10.5, 9.5, 9.0, 10.0},
	}
	// Trigger at bar 1 with stop 9: bar 2's low 8.5 breaks it before the entry.
	if s.triggerAlive(md, 1, 4, 9) {
		t.Fatalf("a low below the stop between the trigger and the entry must invalidate it")
	}
	// Trigger at bar 2 with stop 8.5: nothing after it breaks the level.
	if !s.triggerAlive(md, 2, 4, 8.5) {
		t.Fatalf("intact trigger reported as invalidated")
	}
	// Entry close at or below the stop is never valid.
	if s.triggerAlive(md, 2, 4, 10.0) {
		t.Fatalf("close at the stop level must invalidate the entry")
	}
}

func TestRiskSanityBounds(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))

	// Baseline: the permissive params do produce an entry.
	if _, _, ok := firstBuy(NewWithParams("TEST", testParams()), highs, lows, closes, times); !ok {
		t.Fatalf("baseline entry missing; the other assertions would be vacuous")
	}

	tight := testParams()
	tight.MinRiskATR = 100 // no realistic risk can clear this
	if _, _, ok := firstBuy(NewWithParams("TEST", tight), highs, lows, closes, times); ok {
		t.Fatalf("entry fired despite risk below MinRiskATR")
	}

	wide := testParams()
	wide.MaxRiskATR = 0.001 // any realistic risk exceeds this
	if _, _, ok := firstBuy(NewWithParams("TEST", wide), highs, lows, closes, times); ok {
		t.Fatalf("entry fired despite risk above MaxRiskATR")
	}
}

func TestStopBufferWidensTheStop(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))

	plain, _, ok := firstBuy(NewWithParams("TEST", testParams()), highs, lows, closes, times)
	if !ok {
		t.Fatalf("baseline entry missing")
	}
	p := testParams()
	p.StopBufferATR = 0.5
	buffered, _, ok := firstBuy(NewWithParams("TEST", p), highs, lows, closes, times)
	if !ok {
		t.Fatalf("buffered entry missing")
	}
	if !(buffered.StopLoss < plain.StopLoss) {
		t.Fatalf("buffered stop %v must sit below the plain stop %v", buffered.StopLoss, plain.StopLoss)
	}
}

func TestNoEntryWhenAlreadyInPosition(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())
	for i := 40; i < len(closes); i++ {
		md := mdPrefix(highs, lows, closes, times, i)
		md.Position = &strategy.Position{PurchasePrice: closes[i], Quantity: 1}
		if sig := s.Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("Buy emitted while a position is open (bar %d)", i)
		}
	}
}

// openPos builds MarketData for an open long on a single trailing bar.
func openPos(high, low, close float64, at time.Time, stop, tp float64) strategy.MarketData {
	md := strategy.MarketData{
		Price:  close,
		Highs:  []float64{high},
		Lows:   []float64{low},
		Closes: []float64{close},
		Times:  []time.Time{at},
		Position: &strategy.Position{
			PurchasePrice: 100,
			Quantity:      1,
			StopLoss:      stop,
			TakeProfit:    tp,
		},
	}
	return md
}

func TestExitStopLoss(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := openPos(101, 97, 98, mskAt(2026, 7, 20, 12, 0), 98.5, 110)
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind=%v reason=%q want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 98.5 {
		t.Fatalf("StopLoss=%v want 98.5 (engine fills stops at min(level, open))", sig.StopLoss)
	}
}

func TestExitTakeProfit(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := openPos(110.5, 104, 110, mskAt(2026, 7, 20, 12, 0), 95, 110)
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 110 {
		t.Fatalf("TakeProfit=%v want 110", sig.TakeProfit)
	}
}

func TestStopLossWinsWhenBothTouch(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	// The bar spans both levels; the conservative assumption is that the stop hit first.
	md := openPos(112, 94, 105, mskAt(2026, 7, 20, 12, 0), 95, 110)
	sig := s.Decide(md)
	if sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL to win a same-bar SL/TP touch", sig.Reason)
	}
}

// stochSeries builds a series whose %K sits above 80 on the second-to-last bar and drops
// below 80 on the last one: a long flat-high range, then a close near the top, then a
// close near the bottom of the same range.
func stochSeries() (highs, lows, closes []float64) {
	const n = 20
	for i := 0; i < n; i++ {
		highs = append(highs, 110)
		lows = append(lows, 100)
		closes = append(closes, 105)
	}
	closes[n-2] = 109.5 // %K = 95
	closes[n-1] = 102   // %K = 20
	return highs, lows, closes
}

func TestExitStochasticCrossDown(t *testing.T) {
	highs, lows, closes := stochSeries()
	n := len(closes)
	times := sessionTimes(n)
	s := NewWithParams("TEST", DefaultParams())

	md := strategy.MarketData{
		Price:  closes[n-1],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
		Times:  times,
		Position: &strategy.Position{
			PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 200,
		},
	}
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "STOCH" {
		t.Fatalf("kind=%v reason=%q want Sell/STOCH", sig.Kind, sig.Reason)
	}
	if model.IsStopReason(sig.Reason) {
		t.Fatalf("STOCH must not be a stop-style reason: it fills at the bar close")
	}
}

func TestStochasticExitDisabled(t *testing.T) {
	highs, lows, closes := stochSeries()
	n := len(closes)
	p := DefaultParams()
	p.EnableStochExit = 0
	s := NewWithParams("TEST", p)

	md := strategy.MarketData{
		Price:  closes[n-1],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
		Times:  sessionTimes(n),
		Position: &strategy.Position{
			PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 200,
		},
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("stochastic exit fired while EnableStochExit=0 (reason=%q)", sig.Reason)
	}
}

func TestExitEndOfDay(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"monday midday", mskAt(2026, 7, 20, 12, 0), false},
		{"monday last bar", mskAt(2026, 7, 20, 16, 55), true},
		{"friday midday", mskAt(2026, 7, 24, 12, 0), false},
		{"friday last bar", mskAt(2026, 7, 24, 13, 55), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := openPos(101, 99, 100, tc.at, 90, 200)
			sig := s.Decide(md)
			got := sig.Kind == model.SignalSell && sig.Reason == "EOD"
			if got != tc.want {
				t.Fatalf("EOD exit = %v want %v (reason=%q)", got, tc.want, sig.Reason)
			}
		})
	}
}

func TestNoExitWithoutTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := openPos(101, 99, 100, time.Time{}, 90, 200)
	md.Times = nil
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("missing Times must degrade the EOD exit to a no-op, got reason=%q", sig.Reason)
	}
}
