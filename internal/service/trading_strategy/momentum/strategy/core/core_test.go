package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func defaultParams() Params {
	return Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.0, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
	}
}

// buildEntryMD constructs a snapshot engineered to pass all entry gates:
//   - 260 rising-then-dipping closes so close>EMA200 and a fresh bullish MACD cross,
//   - last bar volume well above the VolLookback average,
//   - daily series giving a positive ATR with plenty of remaining room.
func buildEntryMD() strategy.MarketData {
	n := 260
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		base := 100.0 + float64(i)*0.5 // strong uptrend keeps close>EMA200
		closes[i] = base
		highs[i] = base + 0.3
		lows[i] = base - 0.3
		vols[i] = 1000
	}
	// Engineer a fresh MACD bullish cross at the last bar: dip for a few bars then pop.
	closes[n-4], closes[n-3], closes[n-2] = closes[n-5]-1, closes[n-5]-2, closes[n-5]-2.5
	highs[n-4], highs[n-3], highs[n-2] = closes[n-4]+0.3, closes[n-3]+0.3, closes[n-2]+0.3
	lows[n-4], lows[n-3], lows[n-2] = closes[n-4]-0.3, closes[n-3]-0.3, closes[n-2]-0.3
	closes[n-1] = closes[n-5] + 8 // strong pop -> MACD crosses up
	highs[n-1] = closes[n-1] + 0.5
	lows[n-1] = closes[n-1] - 0.5
	vols[n-1] = 5000 // above 1.2x average

	dailyH := []float64{105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120}
	dailyL := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115}
	dailyC := []float64{104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119}

	return strategy.MarketData{
		Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols,
		DailyHighs: dailyH, DailyLows: dailyL, DailyCloses: dailyC,
		TodayHigh: closes[n-1] + 0.5, TodayLow: closes[n-1] - 0.5, // tiny consumed range -> room OK
	}
}

func TestEntryFiresWhenAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(buildEntryMD())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind=%v want Buy", sig.Kind)
	}
	if sig.StopLoss <= 0 || sig.TakeProfit <= sig.Price {
		t.Fatalf("SL=%f TP=%f price=%f want SL>0 and TP>price", sig.StopLoss, sig.TakeProfit, sig.Price)
	}
	if sig.ATR <= 0 {
		t.Fatalf("ATR=%f want >0", sig.ATR)
	}
	if sig.Ticker != "TEST" {
		t.Fatalf("ticker=%q want TEST", sig.Ticker)
	}
	if !strings.Contains(sig.EntryReason, "MACD") || !strings.Contains(sig.EntryReason, "ATR") {
		t.Fatalf("EntryReason missing detail: %q", sig.EntryReason)
	}
}

func TestEntryBlockedByTrendFilter(t *testing.T) {
	md := buildEntryMD()
	for i := range md.Closes { // flat-low series -> price below EMA200
		md.Closes[i] = 50
		md.Highs[i] = 50.3
		md.Lows[i] = 49.7
	}
	md.Price = 50
	s := NewWithParams("TEST", defaultParams())
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when close < EMA200")
	}
}

func TestEntryBlockedByVolume(t *testing.T) {
	md := buildEntryMD()
	md.Volumes[len(md.Volumes)-1] = 100 // below average
	s := NewWithParams("TEST", defaultParams())
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked on weak volume")
	}
}

func TestEntryBlockedByDailyATRRoom(t *testing.T) {
	md := buildEntryMD()
	// Make today's consumed range exceed MaxDailyATRUsed*dailyATR.
	md.TodayHigh = md.Price + 50
	md.TodayLow = md.Price - 50
	s := NewWithParams("TEST", defaultParams())
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when daily ATR room is used up")
	}
}

func TestEntryBlockedByMACDBelowZeroFlag(t *testing.T) {
	md := buildEntryMD()
	p := defaultParams()
	p.MACDBelowZeroOnly = 1 // a strong uptrend has MACD>0, so the cross is above zero -> blocked
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when MACDBelowZeroOnly=1 and macd>0")
	}
}

func inPositionMD(barLow, barHigh, recentHigh float64, pos *strategy.Position) strategy.MarketData {
	// 30 tight flat bars around the recentHigh level so ATR is small and
	// well-defined (~1); override the last bar's high/low. Price is unused by the
	// manage branch (it keys off Position + bar high/low), so it stays 100.
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		closes[i], highs[i], lows[i], vols[i] = recentHigh, recentHigh, recentHigh-1, 1000
	}
	highs[n-1], lows[n-1] = barHigh, barLow
	return strategy.MarketData{
		Price: 100, Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Position: pos,
	}
}

func TestExitStopLoss(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 101, 100, pos)) // barLow 94 <= SL 95
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind=%v reason=%q want Sell/SL", sig.Kind, sig.Reason)
	}
}

func TestExitTakeProfit(t *testing.T) {
	// TP = entry + RR*(entry-stop) = 100 + 2*5 = 110. barHigh 111 >= 110.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 111, 100, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 110 {
		t.Fatalf("TP=%f want 110", sig.TakeProfit)
	}
}

func TestExitStopLossWinsOverTP(t *testing.T) {
	// Both SL (95) and TP (110) touched in the same bar: SL has priority.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 111, 100, pos))
	if sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL (priority over TP)", sig.Reason)
	}
}

func TestExitHoldsWhenNeitherHit(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 109, 100, pos)) // below TP 110, above SL 95
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind=%v want None (hold)", sig.Kind)
	}
}

func TestExitTrailWhenEnabled(t *testing.T) {
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0 // arm immediately
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 120}
	s := NewWithParams("TEST", p)
	// recentHigh 120, ATR≈1, TrailMult 2.5 -> chandelier≈117.5; barLow 117 <= chandelier.
	sig := s.Decide(inPositionMD(117, 121, 120, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("kind=%v reason=%q want Sell/TRAIL", sig.Kind, sig.Reason)
	}
}

func TestEntryFiresWithRisingDailyTrend(t *testing.T) {
	p := defaultParams()
	p.DailyTrendPeriod = 5 // дневные closes в buildEntryMD растут -> наклон вверх
	s := NewWithParams("TEST", p)
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when daily EMA slope is up")
	}
}

func TestEntryBlockedByDailyTrendFilter(t *testing.T) {
	md := buildEntryMD()
	for i := range md.DailyCloses { // плоские дневные closes -> EMA не растёт
		md.DailyCloses[i] = 110
	}
	p := defaultParams()
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when daily EMA is not rising")
	}
}

func TestDailyTrendFilterDisabledIgnoresSlope(t *testing.T) {
	md := buildEntryMD()
	for i := range md.DailyCloses { // падающие дневные closes
		md.DailyCloses[i] = 200 - float64(i)
	}
	p := defaultParams() // DailyTrendPeriod = 0 -> фильтр выключен
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when daily filter is disabled, regardless of slope")
	}
}

func TestDailyTrendFilterPassesWithInsufficientHistory(t *testing.T) {
	md := buildEntryMD()
	md.DailyCloses = md.DailyCloses[:5] // меньше, чем period+slopeBars (5+3=8)
	md.DailyHighs = md.DailyHighs[:5]
	md.DailyLows = md.DailyLows[:5]
	p := defaultParams()
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire (filter passes) when daily history is insufficient")
	}
}

func TestExplainReportsDailyTrendBlock(t *testing.T) {
	md := buildEntryMD()
	for i := range md.DailyCloses {
		md.DailyCloses[i] = 110 // плоско -> фильтр блокирует
	}
	p := defaultParams()
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	out := s.Explain(md)
	if !strings.Contains(out, "Дневной тренд") {
		t.Fatalf("Explain should mention daily trend gate, got: %q", out)
	}
}

func TestCooldownBlocksReentryAfterExit(t *testing.T) {
	p := defaultParams()
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)

	// Бар 1: открываем позицию (md.Position == nil).
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("expected entry on bar 1")
	}
	// Бар 2: в позиции, ловим SL -> выход.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	if sig := s.Decide(inPositionMD(94, 101, 100, pos)); sig.Reason != "SL" {
		t.Fatalf("expected SL exit, got %q", sig.Reason)
	}
	// Бары 3-5: flat, кулдаун активен (barsSinceExit 0,1,2 < 3) -> вход блокируется.
	for i := 0; i < 3; i++ {
		if sig := s.Decide(buildEntryMD()); sig.Kind == model.SignalBuy {
			t.Fatalf("entry should be blocked during cooldown, offset %d", i)
		}
	}
	// Бар 6: кулдаун истёк (barsSinceExit 3 >= 3) -> вход снова разрешён.
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire after cooldown elapses")
	}
}

func TestCooldownDisabledAllowsImmediateReentry(t *testing.T) {
	p := defaultParams() // CooldownBars = 0
	s := NewWithParams("TEST", p)
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("expected entry on bar 1")
	}
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s.Decide(inPositionMD(94, 101, 100, pos)) // выход по SL
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire immediately when cooldown disabled")
	}
}

func TestExplainReportsCooldownBlock(t *testing.T) {
	p := defaultParams()
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)
	// Открыть позицию, затем выйти по SL.
	s.Decide(buildEntryMD())
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s.Decide(inPositionMD(94, 101, 100, pos)) // выход; prevInPosition=true
	// Первый flat-бар после выхода обнуляет barsSinceExit (edge ловится в Decide).
	s.Decide(buildEntryMD()) // barsSinceExit -> 0, вход заблокирован кулдауном
	// Теперь Explain должен показать БЛОК кулдауна (0 из 3), а не pass.
	out := s.Explain(buildEntryMD())
	if !strings.Contains(out, "Кулдаун") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should report cooldown BLOCK, got: %q", out)
	}
}

func TestExplainReportsCooldownPass(t *testing.T) {
	p := defaultParams()
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)
	// Свежая стратегия: barsSinceExit насыщен (≥ CooldownBars) -> кулдаун пройден.
	out := s.Explain(buildEntryMD())
	if !strings.Contains(out, "Кулдаун") {
		t.Fatalf("Explain should mention cooldown gate, got: %q", out)
	}
	if strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("cooldown should PASS on a fresh strategy, got a block: %q", out)
	}
}

// deferredBars returns numBars consecutive hourly snapshots simulating the bars
// after a fresh MACD bullish cross. Snapshot 0 is the cross bar. Each later
// snapshot appends one "holding" bar that nudges price up slightly (no new cross,
// MACD stays bullish). Confirming volume (5000) is placed only on the snapshot at
// index volumeAtBar; every other last-bar volume is weak (100). Feed the snapshots
// to one Strategy in order to drive the arming state machine.
func deferredBars(numBars, volumeAtBar int) []strategy.MarketData {
	base := buildEntryMD()
	snaps := make([]strategy.MarketData, numBars)
	for k := 0; k < numBars; k++ {
		closes := append([]float64(nil), base.Closes...)
		highs := append([]float64(nil), base.Highs...)
		lows := append([]float64(nil), base.Lows...)
		vols := append([]int64(nil), base.Volumes...)
		last := closes[len(closes)-1]
		for j := 0; j < k; j++ { // append k holding bars
			last += 0.2
			closes = append(closes, last)
			highs = append(highs, last+0.5)
			lows = append(lows, last-0.2)
			vols = append(vols, 100)
		}
		if k == volumeAtBar {
			vols[len(vols)-1] = 5000
		} else {
			vols[len(vols)-1] = 100
		}
		md := base
		md.Closes, md.Highs, md.Lows, md.Volumes = closes, highs, lows, vols
		md.Price = closes[len(closes)-1]
		md.TodayHigh = md.Price + 0.5
		md.TodayLow = md.Price - 0.5
		snaps[k] = md
	}
	return snaps
}

func TestDeferredEntryFiresWhenVolumeArrivesWithinWindow(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5 // generous; drift is tested separately
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1) // cross bar weak volume; volume on bar 1
	if sig := s.Decide(bars[0]); sig.Kind == model.SignalBuy {
		t.Fatal("should NOT enter on the cross bar (volume weak)")
	}
	if sig := s.Decide(bars[1]); sig.Kind != model.SignalBuy {
		t.Fatal("should enter on the next bar when volume confirms within the window")
	}
}

func TestDeferredDisabledIgnoresLateVolume(t *testing.T) {
	p := defaultParams() // SignalValidBars == 0 -> deferral off
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	if sig := s.Decide(bars[0]); sig.Kind == model.SignalBuy {
		t.Fatal("no entry on cross bar (weak volume)")
	}
	if sig := s.Decide(bars[1]); sig.Kind == model.SignalBuy {
		t.Fatal("with deferral off, late volume must NOT produce an entry")
	}
}

func TestDeferredEntryExpiresAfterWindow(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 1
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(3, 2) // cross(0) weak, hold(1) weak, volume(2) too late
	s.Decide(bars[0])          // arm
	s.Decide(bars[1])          // age to 1 (still within window=1), no volume
	if sig := s.Decide(bars[2]); sig.Kind == model.SignalBuy {
		t.Fatal("signal must expire after SignalValidBars and block the late entry")
	}
}

func TestDeferredEntryBlockedByPriceDrift(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 0.01 // cap = 0.01×ATR; the +0.2 drift exceeds it so entry is blocked
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1) // holding bar nudges price +0.2 from the cross
	s.Decide(bars[0])          // arm at the cross price
	if sig := s.Decide(bars[1]); sig.Kind == model.SignalBuy {
		t.Fatal("entry must be blocked when price drifted beyond MaxDriftATR*ATR")
	}
}

// TestDeferredEntryAllowedWithinDrift is the control case for the drift cap: same
// fixture as the blocked case, but a generous MaxDriftATR proves the +0.2 drift is
// what the cap (not volume or the window) gates on.
func TestDeferredEntryAllowedWithinDrift(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5 // generous tolerance
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	s.Decide(bars[0])
	if sig := s.Decide(bars[1]); sig.Kind != model.SignalBuy {
		t.Fatal("entry should be allowed when price stayed within the drift cap")
	}
}

// armInput is a minimal decideInput that satisfies qualifiesAsTrigger:
// fresh cross, clear uptrend, MACD above signal, daily filter not engaged.
func armInput() decideInput {
	return decideInput{price: 100, emaTrend: 90, atr: 1, crossUp: true, macdAboveSignal: true}
}

func TestAdvanceArmArmsOnQualifyingCross(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 3
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput())
	if s.armedCrossPrice != 100 || s.barsSinceArm != 0 {
		t.Fatalf("want armed at 100 age 0, got price=%f age=%d", s.armedCrossPrice, s.barsSinceArm)
	}
}

func TestAdvanceArmExpiresAfterWindow(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 1
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput()) // age 0
	hold := decideInput{price: 100.1, emaTrend: 90, atr: 1, macdAboveSignal: true}
	s.advanceArm(hold) // age 1, still within window
	if s.armedCrossPrice == 0 {
		t.Fatal("should still be armed at age 1 with window 1")
	}
	s.advanceArm(hold) // age 2 > window 1 -> cancel
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel once age exceeds SignalValidBars")
	}
}

func TestAdvanceArmCancelsOnMomentumDeath(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput())
	dead := decideInput{price: 100.1, emaTrend: 90, atr: 1, macdAboveSignal: false}
	s.advanceArm(dead)
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel when MACD line falls back below the signal line")
	}
}

func TestAdvanceArmCancelsOnTrendBreak(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput())
	below := decideInput{price: 80, emaTrend: 90, atr: 1, macdAboveSignal: true} // price < EMA
	s.advanceArm(below)
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel when price falls back below the trend EMA")
	}
}

func TestAdvanceArmCancelsOnDailyTrendStall(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	armed := armInput()
	armed.dailyTrendKnown = true
	armed.dailyEMANow, armed.dailyEMAPast = 110, 100 // rising at arm time
	s.advanceArm(armed)
	stall := decideInput{price: 100.1, emaTrend: 90, atr: 1, macdAboveSignal: true,
		dailyTrendKnown: true, dailyEMANow: 100, dailyEMAPast: 110} // now falling
	s.advanceArm(stall)
	if s.armedCrossPrice != 0 {
		t.Fatal("should cancel when the daily trend stops rising")
	}
}

func TestAdvanceArmDoesNotArmDuringCooldown(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)
	in := armInput()
	in.barsSinceExit = 1 // inside cooldown (1 < 3)
	s.advanceArm(in)
	if s.armedCrossPrice != 0 {
		t.Fatal("should not arm a signal while cooling down")
	}
}

func TestAdvanceArmClearsWhenInPosition(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 5
	s := NewWithParams("TEST", p)
	s.advanceArm(armInput()) // arm
	in := decideInput{price: 100, emaTrend: 90, atr: 1, pos: &strategy.Position{PurchasePrice: 100}}
	s.advanceArm(in)
	if s.armedCrossPrice != 0 {
		t.Fatal("should clear the armed signal while a position is open")
	}
}

func TestEntryReasonNotesDeferral(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	s.Decide(bars[0]) // arm
	sig := s.Decide(bars[1])
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected deferred entry, got kind=%v", sig.Kind)
	}
	if !strings.Contains(sig.EntryReason, "отложенный вход") {
		t.Fatalf("EntryReason should note the deferral, got: %q", sig.EntryReason)
	}
}

func TestImmediateEntryReasonHasNoDeferralNote(t *testing.T) {
	p := defaultParams() // SignalValidBars == 0 -> immediate entry only
	s := NewWithParams("TEST", p)
	sig := s.Decide(buildEntryMD())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected immediate entry, got kind=%v", sig.Kind)
	}
	if strings.Contains(sig.EntryReason, "отложенный вход") {
		t.Fatalf("immediate entry must not carry a deferral note, got: %q", sig.EntryReason)
	}
}

func TestExplainReportsArmedWaitingSignal(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(3, 2)
	s.Decide(bars[0]) // arm; bar[1] has no fresh cross and weak volume
	out := s.Explain(bars[1])
	if !strings.Contains(out, "взвед") {
		t.Fatalf("Explain should report the armed waiting signal, got: %q", out)
	}
}

func TestExplainReportsDeferredEntryBar(t *testing.T) {
	p := defaultParams()
	p.SignalValidBars = 2
	p.MaxDriftATR = 5
	s := NewWithParams("TEST", p)
	bars := deferredBars(2, 1)
	s.Decide(bars[0])        // arm, no entry (weak volume)
	sig := s.Decide(bars[1]) // deferred entry fires here
	if sig.Kind != model.SignalBuy {
		t.Fatalf("expected a deferred entry on bar 1, got kind=%v", sig.Kind)
	}
	// Engine calls Explain AFTER Decide on the same bar; the arm was just consumed.
	out := s.Explain(bars[1])
	if strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain must not report a block on a bar where a deferred entry fired: %q", out)
	}
	if !strings.Contains(out, "взвед") {
		t.Fatalf("Explain should show the armed signal that produced the entry: %q", out)
	}
}

func TestDailyATRRoomDisabledAllowsEntry(t *testing.T) {
	// Same setup as TestEntryBlockedByDailyATRRoom: consumed range far exceeds any
	// ATR threshold. With MaxDailyATRUsed=0 the gate must be skipped and the entry
	// must fire; with a positive value it must block (control case).
	md := buildEntryMD()
	md.TodayHigh = md.Price + 50
	md.TodayLow = md.Price - 50

	// Gate disabled: entry fires despite exhausted daily range.
	p := defaultParams()
	p.MaxDailyATRUsed = 0
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when MaxDailyATRUsed=0 disables the daily-ATR gate")
	}

	// Control: gate enabled (positive value) -> same range blocks.
	p2 := defaultParams() // MaxDailyATRUsed=0.6
	s2 := NewWithParams("TEST", p2)
	if sig := s2.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when MaxDailyATRUsed>0 and daily range is exhausted")
	}
}

func TestExplainDailyATRRoomDisabledReportsFilterOff(t *testing.T) {
	// With MaxDailyATRUsed=0, Explain must report the "фильтр выключен" pass message
	// even when today's range would normally block.
	md := buildEntryMD()
	md.TodayHigh = md.Price + 50
	md.TodayLow = md.Price - 50

	p := defaultParams()
	p.MaxDailyATRUsed = 0
	s := NewWithParams("TEST", p)
	out := s.Explain(md)
	if !strings.Contains(out, "фильтр выключен") {
		t.Fatalf("Explain should report 'фильтр выключен' when MaxDailyATRUsed<=0, got: %q", out)
	}
	if strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain must not block when daily-ATR gate is disabled, got: %q", out)
	}
}
