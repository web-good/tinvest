package backtest

import (
	"strings"
	"testing"
	"time"

	domainbt "tinvest/internal/domain/backtest"
)

func screenParams() VolParams {
	return VolParams{ATRPeriod: 14, EMAPeriod: 5, RSIPeriod: 3, RSIOversold: 30, RecoverRSI: 50, RecoverBars: 5}
}

func TestVolMetrics_BasicShape(t *testing.T) {
	// 40 bars, short periods so EMA/RSI/ATR series are valid; turnover positive.
	candles := make([]domainbt.Candle, 40)
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := range candles {
		c := 100.0 + float64(i%5) // oscillate a little so RSI moves
		candles[i] = domainbt.Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			High: c + 1, Low: c - 1, Close: c, Volume: 100,
		}
	}
	got := VolMetrics(candles, 1, screenParams())
	if got.Bars != 40 {
		t.Fatalf("bars = %d, want 40", got.Bars)
	}
	if got.MeanATRpct <= 0 || got.LastATRpct <= 0 {
		t.Errorf("ATR%% mean=%v last=%v, want both > 0", got.MeanATRpct, got.LastATRpct)
	}
	if got.TurnoverM <= 0 {
		t.Errorf("turnover = %v, want > 0", got.TurnoverM)
	}
}

func TestVolMetrics_InsufficientHistory(t *testing.T) {
	candles := []domainbt.Candle{
		{Time: time.Now(), High: 10, Low: 9, Close: 9.5, Volume: 100},
		{Time: time.Now(), High: 11, Low: 9.5, Close: 10.5, Volume: 100},
	}
	got := VolMetrics(candles, 1, VolParams{ATRPeriod: 14, EMAPeriod: 200, RSIPeriod: 14, RSIOversold: 30, RecoverRSI: 50, RecoverBars: 24})
	if got.Bars != 2 {
		t.Fatalf("bars = %d, want 2", got.Bars)
	}
	if got.MeanATRpct != 0 || got.LastATRpct != 0 {
		t.Errorf("ATR%% mean=%v last=%v, want 0 (no valid ATR)", got.MeanATRpct, got.LastATRpct)
	}
	if got.Events != 0 {
		t.Errorf("events = %d, want 0 (no history)", got.Events)
	}
}

func TestVolMetrics_DetectsPullbackRecovery(t *testing.T) {
	// Build a long steady uptrend that lifts price well above a slow EMA(50),
	// then inject a short sharp dip (5 down bars) that drives RSI(6) below the
	// oversold threshold WHILE price stays above the EMA, then resume the
	// uptrend so RSI recovers above 50 within RecoverBars. With a fast EMA the
	// dip would push price under the EMA and the trend filter would (correctly)
	// reject the event — hence the deliberately slow EMA(50) and big cushion.
	n := 100
	candles := make([]domainbt.Candle, n)
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	price := 100.0
	for i := 0; i < n; i++ {
		switch {
		case i >= 70 && i < 75:
			price -= 4 // sharp dip → RSI falls into oversold
		default:
			price += 3 // steady uptrend (builds cushion over the slow EMA)
		}
		candles[i] = domainbt.Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			High: price + 0.5, Low: price - 0.5, Close: price, Volume: 100,
		}
	}
	got := VolMetrics(candles, 1, VolParams{ATRPeriod: 14, EMAPeriod: 50, RSIPeriod: 6, RSIOversold: 35, RecoverRSI: 50, RecoverBars: 10})
	if got.Events < 1 {
		t.Fatalf("events = %d, want >= 1 (a recoverable dip in uptrend)", got.Events)
	}
	if got.EventFreq <= 0 {
		t.Errorf("eventFreq = %v, want > 0", got.EventFreq)
	}
}

func TestScoreVolRows_RewardsRecoveryFreqLiquidity(t *testing.T) {
	rows := []VolRow{
		{Ticker: "GOOD", RecoveryRate: 0.9, EventFreq: 10, TurnoverM: 500},
		{Ticker: "WEAK", RecoveryRate: 0.2, EventFreq: 1, TurnoverM: 50},
		{Ticker: "MID", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 200},
	}
	ScoreVolRows(rows, 0.5, 0.3, 0.2)
	byTicker := map[string]float64{}
	for _, r := range rows {
		byTicker[r.Ticker] = r.Score
	}
	if !(byTicker["GOOD"] > byTicker["MID"] && byTicker["MID"] > byTicker["WEAK"]) {
		t.Errorf("score order wrong: GOOD=%.3f MID=%.3f WEAK=%.3f", byTicker["GOOD"], byTicker["MID"], byTicker["WEAK"])
	}
	if byTicker["GOOD"] < 0.999 {
		t.Errorf("GOOD = %.3f, want ~1.0 (top on every dimension)", byTicker["GOOD"])
	}
}

func TestScoreVolRows_IgnoresATRAndVR(t *testing.T) {
	// ATR% and VR2 differ wildly but recovery/freq/liq are identical → equal score.
	rows := []VolRow{
		{Ticker: "A", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 100, MeanATRpct: 9.0, VR2: 0.3},
		{Ticker: "B", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 100, MeanATRpct: 0.5, VR2: 1.4},
	}
	ScoreVolRows(rows, 0.5, 0.3, 0.2)
	if rows[0].Score != rows[1].Score {
		t.Errorf("ATR%%/VR must not affect score: A=%.4f B=%.4f", rows[0].Score, rows[1].Score)
	}
}

func TestScoreVolRows_SingleRow(t *testing.T) {
	rows := []VolRow{{Ticker: "ONLY", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 100}}
	ScoreVolRows(rows, 0.5, 0.3, 0.2)
	if rows[0].Score == 0 {
		t.Errorf("single row score = 0, want > 0")
	}
}

func TestRenderVolatilityMarkdown_SortsByScore(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Name: "Alpha Co", RecoveryRate: 0.3, EventFreq: 2, TurnoverM: 200, MeanATRpct: 1.0, VR2: 0.9, Autocorr1: -0.1, Score: 0.20, Bars: 300},
		{Ticker: "BBB", Name: "Beta Co", RecoveryRate: 0.8, EventFreq: 6, TurnoverM: 50, MeanATRpct: 3.0, VR2: 0.7, Autocorr1: -0.3, Score: 0.80, Bars: 300},
	}
	meta := VolMeta{Months: 12, ATRPeriod: 14, MinTurnover: 50, WRecov: 0.5, WFreq: 0.3, WLiq: 0.2, RSIPeriod: 14, RSIOversold: 30, RecoverBars: 24, RecoverRSI: 50, Scanned: 100, Passed: 2}
	out := RenderVolatilityMarkdown(rows, meta, 0)
	bbb, aaa := strings.Index(out, "BBB"), strings.Index(out, "AAA")
	if bbb == -1 || aaa == -1 {
		t.Fatalf("both tickers must appear; out=%q", out)
	}
	if bbb > aaa {
		t.Errorf("BBB (score 0.80) must rank before AAA (score 0.20)")
	}
	if strings.Contains(out, "%%") {
		t.Errorf("rendered output must not contain literal '%%%%'; got: %q", out)
	}
	for _, want := range []string{"Alpha Co", "Beta Co", "Score", "Восстановл", "VR(2)", "Autocorr"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output; got: %q", want, out)
		}
	}
}

func TestRenderVolatilityMarkdown_TopN(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Score: 0.1, Bars: 300},
		{Ticker: "BBB", Score: 0.9, Bars: 300},
		{Ticker: "CCC", Score: 0.5, Bars: 300},
	}
	out := RenderVolatilityMarkdown(rows, VolMeta{}, 2)
	if strings.Contains(out, "AAA") {
		t.Errorf("topN=2 must drop AAA (lowest score); out=%q", out)
	}
	if !strings.Contains(out, "BBB") || !strings.Contains(out, "CCC") {
		t.Errorf("topN=2 must keep BBB and CCC")
	}
}
