package core

import (
	"math"
	"strings"
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
			if got := s.isDayEnd(tc.at, defaultBarSpanMin); got != tc.want {
				t.Fatalf("isDayEnd(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestIsDayEndZeroTimeIsNoOp(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if s.isDayEnd(time.Time{}, defaultBarSpanMin) {
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
// up so synthetic series never trip them, the stop sits exactly on the trigger low, and
// the RSI threshold gate is off unless a test turns it on.
func testParams() Params {
	p := DefaultParams()
	p.MinRiskATR = 0
	p.MaxRiskATR = 100
	p.StopBufferATR = 0
	p.RSIEntryMin = 0
	p.HTFTrendEMA = 0
	// The swing-low geometry is the baseline the older gate tests are written against; the ATR
	// stop/target get their own tests, which set StopATR/TPATR explicitly.
	p.StopATR = 0
	p.TPATR = 0
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

// sessionTimesFrom returns len(n) MSK bar open-times, 5 minutes apart, starting at base.
func sessionTimesFrom(n int, base time.Time) []time.Time {
	out := make([]time.Time, n)
	for i := range out {
		out[i] = base.Add(time.Duration(i*defaultBarSpanMin) * time.Minute)
	}
	return out
}

// sessionTimes returns len(n) MSK bar open-times, 5 minutes apart, starting Mon 10:00.
func sessionTimes(n int) []time.Time {
	return sessionTimesFrom(n, mskAt(2026, 7, 20, 10, 0))
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
		times[i] = base.Add(time.Duration(i*defaultBarSpanMin) * time.Minute)
	}
	s := NewWithParams("TEST", testParams())
	if _, _, ok := firstBuy(s, highs, lows, closes, times); ok {
		t.Fatalf("entry fired on a Saturday")
	}
}

func TestNoEntryOnDayEndBar(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	s := NewWithParams("TEST", testParams())

	// Mid-session baseline: the RSI/MACD math is time-independent, so bar i is the entry
	// opportunity regardless of which clock time it's stamped with.
	baseline := sessionTimes(len(closes))
	_, i, ok := firstBuy(s, highs, lows, closes, baseline)
	if !ok {
		t.Fatalf("baseline entry missing; the day-end assertions below would be vacuous")
	}

	// Shift the whole series so bar i opens at 16:55 Mon-Thu: in session (TestInSession's
	// "mon last bar" case) but also the day-end bar (TestIsDayEnd's matching case) — the
	// last bar that still ends inside the session. All earlier bars land between ~11:25 and
	// 16:55, still comfortably mid-session on the same Monday.
	monThu := sessionTimesFrom(len(closes), mskAt(2026, 7, 20, 16, 55).Add(-time.Duration(i*defaultBarSpanMin)*time.Minute))
	if !s.inSession(monThu[i]) || !s.isDayEnd(monThu[i], defaultBarSpanMin) {
		t.Fatalf("test setup broken: bar %d at %v must be in-session AND day-end", i, monThu[i])
	}
	if _, _, ok := firstBuy(s, highs, lows, closes, monThu); ok {
		t.Fatalf("entry fired on the Mon-Thu day-end bar (16:55)")
	}

	// Same shift onto Friday's early close (13:55): the equivalent Friday day-end bar.
	fri := sessionTimesFrom(len(closes), mskAt(2026, 7, 24, 13, 55).Add(-time.Duration(i*defaultBarSpanMin)*time.Minute))
	if !s.inSession(fri[i]) || !s.isDayEnd(fri[i], defaultBarSpanMin) {
		t.Fatalf("test setup broken: bar %d at %v must be in-session AND day-end", i, fri[i])
	}
	if _, _, ok := firstBuy(s, highs, lows, closes, fri); ok {
		t.Fatalf("entry fired on the Friday day-end bar (13:55)")
	}
}

// declineRallyPullbackRally builds a four-phase synthetic 5m series: decline(down1), rally
// (up1) — a below-zero MACD cross fires here, same as declineThenRally — then a second
// pullback decline(down2) and a second rally(up2). The second rally's residual momentum
// from the first pushes MACD to a FRESH bullish cross with both lines already at/above
// zero: the case the MACD-zero gate in enter() must reject. Without this second leg, the
// only crosses ever offered sit deep below zero, making a "no Buy on an above-zero cross"
// assertion vacuous (see task-5 review finding 2).
func declineRallyPullbackRally(down1, up1, down2, up2 int, start, step, wick float64) (highs, lows, closes []float64) {
	price := start
	for i := 0; i < down1; i++ {
		if i%7 == 6 {
			price += step * 0.25
		} else {
			price -= step
		}
		closes = append(closes, price)
	}
	for i := 0; i < up1; i++ {
		price += 2 * step
		closes = append(closes, price)
	}
	for i := 0; i < down2; i++ {
		if i%7 == 6 {
			price += step * 0.25
		} else {
			price -= step
		}
		closes = append(closes, price)
	}
	for i := 0; i < up2; i++ {
		price += 2 * step
		closes = append(closes, price)
	}
	for _, c := range closes {
		highs = append(highs, c+wick)
		lows = append(lows, c-wick)
	}
	return highs, lows, closes
}

func TestEveryEntryHasBothMACDLinesBelowZero(t *testing.T) {
	// Decline, rally (first, below-zero entry opportunity), pullback, second rally: the
	// second rally offers a FRESH bullish MACD cross with both lines already at/above zero.
	// Bar 75 sits squarely mid-session (16:15 Mon), so a Buy refusal there is attributable
	// to the MACD-zero gate, not to the session gate (the fixture this test replaces put
	// its only above-zero cross past 17:00, where the session gate alone explained every
	// rejection — see task-5 review finding 2).
	highs, lows, closes := declineRallyPullbackRally(60, 8, 6, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	var buys, aboveZeroCrosses int
	for i := 40; i < len(closes); i++ {
		macd, signal := indicators.MACD(closes[:i+1], 3, 6, 9)
		if macd[i-1] <= signal[i-1] && macd[i] > signal[i] && (macd[i] >= 0 || signal[i] >= 0) {
			aboveZeroCrosses++
		}
		sig := s.Decide(mdPrefix(highs, lows, closes, times, i))
		if sig.Kind != model.SignalBuy {
			continue
		}
		buys++
		if macd[i] >= 0 || signal[i] >= 0 {
			t.Fatalf("Buy at bar %d with MACD above zero: macd=%v signal=%v", i, macd[i], signal[i])
		}
	}
	if buys == 0 {
		t.Fatalf("no Buy fired at all; the assertion above would be vacuous")
	}
	if aboveZeroCrosses == 0 {
		t.Fatalf("fixture never offers an above-zero bullish MACD cross; the gate is untested")
	}
}

// declineDoubleCrossBreach builds a synthetic 5m series where the RSI trigger (bar n-2) is
// one bar older than the MACD-confirmed entry bar (n-1): a first rally crosses RSI and MACD
// together, a sharp pullback drags MACD back under its signal line while RSI stays above
// the oversold zone (so the pullback bar's RSI dip-and-recover never registers a fresher
// trigger), and a further push re-crosses MACD bullish one bar later. That 1-bar gap is
// exactly where a planted low-side wick on the entry bar — breaching the trigger bar's low,
// i.e. the stop — exercises triggerAlive's wiring into enter() without touching any close
// (RSI/MACD only look at closes, so the breach is invisible to every other gate).
func declineDoubleCrossBreach() (highs, lows, closes []float64) {
	price := 100.0
	const step = 0.5
	for i := 0; i < 60; i++ {
		if i%7 == 6 {
			price += step * 0.25
		} else {
			price -= step
		}
		closes = append(closes, price)
	}
	for _, d := range []float64{1, 0.3, -2.5, 0.5, 0.5} {
		price += d
		closes = append(closes, price)
	}
	const wick = 0.2
	highs = make([]float64, len(closes))
	lows = make([]float64, len(closes))
	for i, c := range closes {
		highs[i] = c + wick
		lows[i] = c - wick
	}
	return highs, lows, closes
}

func TestTriggerAliveWiringBlocksEntry(t *testing.T) {
	highs, lows, closes := declineDoubleCrossBreach()
	n := len(closes)
	times := sessionTimes(n)
	s := NewWithParams("TEST", testParams())

	// Sanity: the RSI trigger sits exactly one bar before the MACD-confirmed entry bar —
	// otherwise the breach below wouldn't isolate triggerAlive from the other gates.
	rsi := indicators.RSISeries(closes, s.p.RSIPeriod)
	trig, found := s.lastRSITrigger(rsi, n)
	if !found || trig != n-2 {
		t.Fatalf("test setup broken: trig=%d found=%v want %d,true", trig, found, n-2)
	}

	// Baseline: with the stop untouched, this is a legitimate Buy.
	baseline := s.Decide(mdPrefix(highs, lows, closes, times, n-1))
	if baseline.Kind != model.SignalBuy {
		t.Fatalf("baseline entry missing; the breach assertion below would be vacuous (sig=%+v)", baseline)
	}

	// Plant a deep wick on the entry bar's own low, breaching the trigger bar's low (the
	// stop). No close changes, so RSI/MACD/session are unaffected — only triggerAlive can
	// catch this.
	breached := append([]float64(nil), lows...)
	breached[n-1] = lows[trig] - 5
	if sig := s.Decide(mdPrefix(highs, breached, closes, times, n-1)); sig.Kind == model.SignalBuy {
		t.Fatalf("Buy fired despite a stop breach between the RSI trigger and the entry bar")
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

func TestExplainReportsEveryGate(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	out := s.Explain(mdPrefix(highs, lows, closes, times, 63))
	for _, want := range []string{"сессия", "MACD", "RSI", "ATR", "стоп", "риск"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain output missing %q:\n%s", want, out)
		}
	}
}

func TestExplainHandlesShortHistory(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{
		Price:  10,
		Highs:  []float64{10},
		Lows:   []float64{9},
		Closes: []float64{10},
		Times:  []time.Time{mskAt(2026, 7, 20, 12, 0)},
	}
	if out := s.Explain(md); out == "" {
		t.Fatalf("Explain must always return a diagnosis, even on short history")
	}
}

// baselineEntryBar returns the series, times and the bar index on which the permissive
// testParams() strategy takes its first entry. Gate tests build on it so that a "no
// entry" assertion can never be vacuous.
func baselineEntryBar(t *testing.T) (highs, lows, closes []float64, times []time.Time, bar int) {
	t.Helper()
	highs, lows, closes = declineThenRally(60, 8, 100, 0.5, 0.2)
	times = sessionTimes(len(closes))
	_, bar, ok := firstBuy(NewWithParams("TEST", testParams()), highs, lows, closes, times)
	if !ok {
		t.Fatalf("baseline entry missing; gate assertions would be vacuous")
	}
	return highs, lows, closes, times, bar
}

func TestDefaultParamsEnableRSIEntryMin(t *testing.T) {
	if got := DefaultParams().RSIEntryMin; got != 50 {
		t.Fatalf("DefaultParams().RSIEntryMin = %v want 50", got)
	}
}

func TestRSIEntryMinBlocksWeakCross(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	rsi := indicators.RSISeries(closes[:bar+1], testParams().RSIPeriod)
	at := rsi[bar]

	p := testParams()
	p.RSIEntryMin = at + 1 // порог выше фактического RSI на баре кросса
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind == model.SignalBuy {
		t.Fatalf("entry fired with RSI %.2f below the %.2f threshold", at, p.RSIEntryMin)
	}
}

func TestRSIEntryMinAllowsStrongCross(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	rsi := indicators.RSISeries(closes[:bar+1], testParams().RSIPeriod)
	at := rsi[bar]

	p := testParams()
	p.RSIEntryMin = at - 1 // порог ниже фактического RSI
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("entry blocked though RSI %.2f clears the %.2f threshold", at, p.RSIEntryMin)
	}
	if !strings.Contains(sig.EntryReason, "RSI на кроссе") {
		t.Fatalf("EntryReason must mention the RSI threshold gate, got %q", sig.EntryReason)
	}
}

func TestRSIEntryMinZeroDisablesGate(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.RSIEntryMin = 0
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("RSIEntryMin=0 must not block anything")
	}
}

func TestExplainReportsRSIEntryMin(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.RSIEntryMin = 50
	out := NewWithParams("TEST", p).Explain(mdPrefix(highs, lows, closes, times, bar))
	if !strings.Contains(out, "RSI на кроссе") {
		t.Fatalf("Explain must report the RSI threshold gate, got:\n%s", out)
	}
}

// htfCloses returns n synthetic hourly closes moving by step per bar from start
// (negative step = downtrend), so the last close sits above (up) or below (down) its EMA.
func htfCloses(n int, start, step float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = start + float64(i)*step
	}
	return out
}

// TestDefaultParamsEnableHTFGate pins that the gate ships ENABLED — it started at 0 (off) and
// was turned on on 2026-07-23. The EMA period itself is a tuning knob (calibration may sweep
// it, including back to 0), so the exact value is deliberately not asserted.
func TestDefaultParamsEnableHTFGate(t *testing.T) {
	if got := DefaultParams().HTFTrendEMA; got <= 0 {
		t.Fatalf("DefaultParams().HTFTrendEMA = %d, the H1 trend gate must ship enabled", got)
	}
}

func TestHTFTrendGateAllowsUptrend(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = htfCloses(60, 100, 0.5) // растущий H1: цена выше EMA
	sig := NewWithParams("TEST", p).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("entry blocked though H1 close sits above its EMA")
	}
	if !strings.Contains(sig.EntryReason, "H1") {
		t.Fatalf("EntryReason must mention the H1 trend gate, got %q", sig.EntryReason)
	}
}

func TestHTFTrendGateBlocksDowntrend(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = htfCloses(60, 130, -0.5) // падающий H1: цена ниже EMA
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatalf("entry fired against a falling H1 trend")
	}
}

func TestHTFTrendGateFailsClosedWithoutData(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	cases := map[string][]float64{
		"nil":   nil,
		"short": htfCloses(10, 100, 0.5), // короче периода EMA
	}
	for name, series := range cases {
		t.Run(name, func(t *testing.T) {
			md := mdPrefix(highs, lows, closes, times, bar)
			md.HTFCloses = series
			if sig := NewWithParams("TEST", p).Decide(md); sig.Kind == model.SignalBuy {
				t.Fatalf("gate must fail closed when H1 history is missing")
			}
		})
	}
}

func TestHTFTrendGateOffIgnoresMissingData(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 0

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = nil
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("HTFTrendEMA=0 must not require H1 data")
	}
}

func TestExplainReportsHTFGate(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)
	p := testParams()
	p.HTFTrendEMA = 20

	md := mdPrefix(highs, lows, closes, times, bar)
	md.HTFCloses = nil
	out := NewWithParams("TEST", p).Explain(md)
	if !strings.Contains(out, "нет данных H1") {
		t.Fatalf("Explain must report missing H1 data, got:\n%s", out)
	}

	md.HTFCloses = htfCloses(60, 100, 0.5)
	out = NewWithParams("TEST", p).Explain(md)
	if !strings.Contains(out, "тренд H1") {
		t.Fatalf("Explain must report the H1 trend verdict, got:\n%s", out)
	}
}

func TestBarSpanMinutesInfersTimeframe(t *testing.T) {
	base := mskAt(2026, 7, 20, 10, 0)
	series := func(n, stepMin int) []time.Time {
		out := make([]time.Time, n)
		for i := range out {
			out[i] = base.Add(time.Duration(i*stepMin) * time.Minute)
		}
		return out
	}
	cases := []struct {
		name  string
		times []time.Time
		want  int
	}{
		{"5m", series(20, 5), 5},
		{"15m", series(20, 15), 15},
		{"30m", series(20, 30), 30},
		{"no times falls back", nil, defaultBarSpanMin},
		{"single bar falls back", series(1, 15), defaultBarSpanMin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := barSpanMinutes(tc.times); got != tc.want {
				t.Fatalf("barSpanMinutes = %d want %d", got, tc.want)
			}
		})
	}
}

// TestBarSpanMinutesIgnoresOvernightGaps pins the median: an intraday series carries a
// multi-hour jump at every session boundary, which must not be read as the bar span.
func TestBarSpanMinutesIgnoresOvernightGaps(t *testing.T) {
	var times []time.Time
	for d := 0; d < 3; d++ {
		day := mskAt(2026, 7, 20+d, 10, 0)
		for i := 0; i < 10; i++ {
			times = append(times, day.Add(time.Duration(i*15)*time.Minute))
		}
	}
	if got := barSpanMinutes(times); got != 15 {
		t.Fatalf("barSpanMinutes across day boundaries = %d want 15", got)
	}
}

func TestIsDayEndScalesWithBarSpan(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name    string
		at      time.Time
		spanMin int
		want    bool
	}{
		// Mon-Thu close 17:00. A 15m bar opening at 16:50 ends at 17:05 — it is the last
		// one, while a 5m bar opening at 16:50 ends exactly at 17:00 and is also last.
		{"15m 16:45 is last bar", mskAt(2026, 7, 20, 16, 45), 15, true},
		{"15m 16:30 is not", mskAt(2026, 7, 20, 16, 30), 15, false},
		{"30m 16:30 is last bar", mskAt(2026, 7, 20, 16, 30), 30, true},
		{"30m 16:00 is not", mskAt(2026, 7, 20, 16, 0), 30, false},
		{"5m 16:30 is not", mskAt(2026, 7, 20, 16, 30), 5, false},
		{"fri 15m 13:45 is last bar", mskAt(2026, 7, 24, 13, 45), 15, true},
		{"fri 15m 13:30 is not", mskAt(2026, 7, 24, 13, 30), 15, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isDayEnd(tc.at, tc.spanMin); got != tc.want {
				t.Fatalf("isDayEnd(%v, %d) = %v want %v", tc.at, tc.spanMin, got, tc.want)
			}
		})
	}
}

// TestEODClosesOnLastBarOfA15mSeries is the end-to-end guard: with 15-minute candles the
// forced close must fire on the 16:45 bar. Pinned against the pre-fix behavior, where the
// hard-coded 5-minute span let the position ride past the session close.
func TestEODClosesOnA15mSeries(t *testing.T) {
	s := NewWithParams("TEST", testParams())
	pos := &strategy.Position{PurchasePrice: 100, Quantity: 1}

	md := strategy.MarketData{
		Price:    100,
		Highs:    []float64{100, 100},
		Lows:     []float64{100, 100},
		Closes:   []float64{100, 100},
		Times:    []time.Time{mskAt(2026, 7, 20, 16, 30), mskAt(2026, 7, 20, 16, 45)},
		Position: pos,
	}
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("15m bar 16:45 must force the EOD close, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	md.Times = []time.Time{mskAt(2026, 7, 20, 16, 0), mskAt(2026, 7, 20, 16, 15)}
	if sig := s.Decide(md); sig.Kind == model.SignalSell && sig.Reason == "EOD" {
		t.Fatalf("15m bar 16:15 is not the last bar of the session")
	}
}

// atrAt returns the ATR the strategy sees on the entry bar of the baseline series.
func atrAt(highs, lows, closes []float64, bar, period int) float64 {
	return indicators.ATR(highs[:bar+1], lows[:bar+1], closes[:bar+1], period)
}

// The shipped defaults must use the ATR geometry for both the stop and the target. The
// multipliers themselves are tuning knobs the grid sweeps, so only "enabled" is pinned.
func TestDefaultParamsUseATRStopAndTarget(t *testing.T) {
	p := DefaultParams()
	if p.StopATR <= 0 {
		t.Fatalf("DefaultParams().StopATR = %v, the stop must be ATR-sized by default", p.StopATR)
	}
	if p.TPATR <= 0 {
		t.Fatalf("DefaultParams().TPATR = %v, the target must be ATR-sized by default", p.TPATR)
	}
}

func TestATRStopAndTargetAreMeasuredFromEntry(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)

	p := testParams()
	p.StopATR = 1.5
	p.TPATR = 2.5
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("baseline entry must survive the ATR geometry, got kind=%v", sig.Kind)
	}

	entry := closes[bar]
	atr := atrAt(highs, lows, closes, bar, p.ATRPeriod)
	wantStop := entry - 1.5*atr
	wantTP := entry + 2.5*atr
	if math.Abs(sig.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %.6f want %.6f (вход %.4f - 1.5×ATR %.4f)", sig.StopLoss, wantStop, entry, atr)
	}
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("TakeProfit = %.6f want %.6f (вход %.4f + 2.5×ATR %.4f)", sig.TakeProfit, wantTP, entry, atr)
	}
}

func TestStopATRZeroKeepsStructuralStop(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)

	p := testParams() // StopATR = 0
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("baseline entry expected, got kind=%v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-sig.Level) > 1e-9 {
		t.Fatalf("StopATR=0 must keep the swing-low stop: StopLoss %.6f vs level %.6f", sig.StopLoss, sig.Level)
	}
}

func TestTPATRZeroKeepsRRTarget(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)

	p := testParams()
	p.StopATR = 1.0
	p.TPATR = 0 // тейк снова считается как RR × риск
	p.RR = 3.0
	sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("baseline entry expected, got kind=%v", sig.Kind)
	}
	entry := closes[bar]
	wantTP := entry + 3.0*(entry-sig.StopLoss)
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("TakeProfit = %.6f want %.6f (RR-режим)", sig.TakeProfit, wantTP)
	}
}

// The [MinRiskATR..MaxRiskATR] bounds test entry-to-swing-low distance, which is a market
// measurement. In ATR mode the risk is StopATR×ATR by construction, so the same bounds would
// degenerate into a constant verdict on Params alone — the gate is skipped instead.
func TestATRStopSkipsVacuousRiskBounds(t *testing.T) {
	highs, lows, closes, times, bar := baselineEntryBar(t)

	p := testParams()
	p.StopATR = 1.0
	p.MinRiskATR = 2.0 // отвергло бы вход, будь граница применена к ATR-риску
	p.MaxRiskATR = 3.0
	if sig := NewWithParams("TEST", p).Decide(mdPrefix(highs, lows, closes, times, bar)); sig.Kind != model.SignalBuy {
		t.Fatalf("ATR-режим не должен проверять вырожденные границы риска, got kind=%v", sig.Kind)
	}
}

// volCollapse builds a flat-priced series whose bar range (and therefore ATR) is `wide` for
// the first `calm` bars and `narrow` afterwards. Closes never move, so the SL/TP/stochastic
// exits stay silent and only the ATR exit can fire.
func volCollapse(calm, quiet int, wide, narrow float64) (highs, lows, closes []float64) {
	for i := 0; i < calm+quiet; i++ {
		r := wide
		if i >= calm {
			r = narrow
		}
		highs = append(highs, 100+r)
		lows = append(lows, 100-r)
		closes = append(closes, 100)
	}
	return highs, lows, closes
}

// atrExitParams isolates the ATR exit: short ATR/averaging windows so a synthetic series can
// reach them, every other exit disarmed.
func atrExitParams() Params {
	p := DefaultParams()
	p.ATRPeriod = 3
	p.ATRAvgPeriod = 5
	p.ATRExitRatio = 0.8
	p.EnableStochExit = 0
	return p
}

func atrExitMD(highs, lows, closes []float64, stop, tp float64) strategy.MarketData {
	n := len(closes)
	return strategy.MarketData{
		Price:  closes[n-1],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
		Times:  sessionTimes(n),
		Position: &strategy.Position{
			PurchasePrice: 100, Quantity: 1, StopLoss: stop, TakeProfit: tp,
		},
	}
}

func TestDefaultParamsEnableATRExit(t *testing.T) {
	p := DefaultParams()
	if p.ATRExitRatio != 0.8 {
		t.Fatalf("DefaultParams().ATRExitRatio = %v want 0.8", p.ATRExitRatio)
	}
	if p.ATRAvgPeriod != 50 {
		t.Fatalf("DefaultParams().ATRAvgPeriod = %d want 50", p.ATRAvgPeriod)
	}
}

func TestATRExitFiresWhenVolatilityDropsIntoTheZone(t *testing.T) {
	highs, lows, closes := volCollapse(20, 1, 1.0, 0.05)
	sig := NewWithParams("TEST", atrExitParams()).Decide(atrExitMD(highs, lows, closes, 90, 200))
	if sig.Kind != model.SignalSell || sig.Reason != "ATR" {
		t.Fatalf("kind=%v reason=%q want Sell/ATR", sig.Kind, sig.Reason)
	}
	if model.IsStopReason(sig.Reason) {
		t.Fatalf("ATR exit fills at the bar close and must not be a stop-style reason")
	}
}

func TestATRExitSilentWhileVolatilityHolds(t *testing.T) {
	highs, lows, closes := volCollapse(21, 0, 1.0, 1.0) // ровная волатильность, отношение = 1
	if sig := NewWithParams("TEST", atrExitParams()).Decide(atrExitMD(highs, lows, closes, 90, 200)); sig.Kind == model.SignalSell {
		t.Fatalf("ATR exit fired on flat volatility (reason=%q)", sig.Reason)
	}
}

// The zone must be entered from above on THIS bar: a series that already sits deep inside it
// gives no cross and must not exit, otherwise the rule degenerates into "low volatility = out".
func TestATRExitNeedsTheCrossNotJustALowReading(t *testing.T) {
	highs, lows, closes := volCollapse(20, 6, 1.0, 0.05)
	if sig := NewWithParams("TEST", atrExitParams()).Decide(atrExitMD(highs, lows, closes, 90, 200)); sig.Kind == model.SignalSell {
		t.Fatalf("ATR exit fired without a cross into the zone (reason=%q)", sig.Reason)
	}
}

func TestATRExitDisabledByZeroRatio(t *testing.T) {
	highs, lows, closes := volCollapse(20, 1, 1.0, 0.05)
	p := atrExitParams()
	p.ATRExitRatio = 0
	if sig := NewWithParams("TEST", p).Decide(atrExitMD(highs, lows, closes, 90, 200)); sig.Kind == model.SignalSell {
		t.Fatalf("ATR exit fired while ATRExitRatio=0 (reason=%q)", sig.Reason)
	}
}

func TestATRExitYieldsToStopLoss(t *testing.T) {
	highs, lows, closes := volCollapse(20, 1, 1.0, 0.05)
	n := len(closes)
	lows[n-1] = 95 // бар задевает стоп на том же баре, где ATR входит в зону
	sig := NewWithParams("TEST", atrExitParams()).Decide(atrExitMD(highs, lows, closes, 96, 200))
	if sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL to outrank the ATR exit", sig.Reason)
	}
}

func TestATRExitSilentOnShortHistory(t *testing.T) {
	highs, lows, closes := volCollapse(4, 1, 1.0, 0.05) // короче ATRPeriod+ATRAvgPeriod+1
	if sig := NewWithParams("TEST", atrExitParams()).Decide(atrExitMD(highs, lows, closes, 90, 200)); sig.Kind == model.SignalSell {
		t.Fatalf("ATR exit fired without enough history (reason=%q)", sig.Reason)
	}
}
