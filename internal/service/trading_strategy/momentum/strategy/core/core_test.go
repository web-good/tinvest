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
