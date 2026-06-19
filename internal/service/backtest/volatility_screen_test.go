package backtest

import (
	"strings"
	"testing"

	domainbt "tinvest/internal/domain/backtest"
)

func TestVolMetrics_BasicShape(t *testing.T) {
	// 6 bars, period 3 → series valid from index 3; ATR% positive.
	candles := []domainbt.Candle{
		{High: 10, Low: 9, Close: 9.5, Volume: 100},
		{High: 11, Low: 9.5, Close: 10.5, Volume: 100},
		{High: 12, Low: 10, Close: 11.5, Volume: 100},
		{High: 13, Low: 11, Close: 12.5, Volume: 100},
		{High: 14, Low: 12, Close: 13.5, Volume: 100},
		{High: 15, Low: 13, Close: 14.5, Volume: 100},
	}
	mean, last, turn, _, _, bars := VolMetrics(candles, 1, 3)

	if bars != 6 {
		t.Fatalf("bars = %d, want 6", bars)
	}
	if mean <= 0 || last <= 0 {
		t.Errorf("mean=%v last=%v, want both > 0", mean, last)
	}
	// turnover = mean(volume*lot*close)/1e6 ; closes avg ~12 → ~100*12/1e6
	if turn <= 0 {
		t.Errorf("turnover = %v, want > 0", turn)
	}
}

func TestVolMetrics_InsufficientHistory(t *testing.T) {
	candles := []domainbt.Candle{
		{High: 10, Low: 9, Close: 9.5, Volume: 100},
		{High: 11, Low: 9.5, Close: 10.5, Volume: 100},
	}
	mean, last, _, _, _, bars := VolMetrics(candles, 1, 14)
	if bars != 2 {
		t.Fatalf("bars = %d, want 2", bars)
	}
	if mean != 0 || last != 0 {
		t.Errorf("mean=%v last=%v, want 0 (no valid ATR)", mean, last)
	}
}

func TestVolMetrics_VarianceRatio(t *testing.T) {
	// Oscillating closes with VARYING-magnitude moves so 2-bar return sums are
	// not near-constant (avoids the zero-variance degenerate case).
	// Repeating delta cycle: [+8,-5,+6,-7,+9,-6,+5,-8] applied over 40 bars.
	deltas := []float64{8, -5, 6, -7, 9, -6, 5, -8}
	osc := make([]domainbt.Candle, 41)
	osc[0] = domainbt.Candle{High: 101, Low: 99, Close: 100, Volume: 100}
	for i := 1; i <= 40; i++ {
		c := osc[i-1].Close + deltas[(i-1)%len(deltas)]
		osc[i] = domainbt.Candle{High: c + 1, Low: c - 1, Close: c, Volume: 100}
	}
	_, _, _, vrOsc, acOsc, _ := VolMetrics(osc, 1, 14)
	if acOsc >= 0 {
		t.Errorf("oscillating autocorr = %v, want negative (mean-reverting)", acOsc)
	}
	if vrOsc <= 0 || vrOsc >= 1 {
		t.Errorf("oscillating VR(2) = %v, want in (0,1)", vrOsc)
	}

	// Steadily trending closes → VR(2) > 1.
	trend := make([]domainbt.Candle, 40)
	for i := 0; i < 40; i++ {
		c := 100.0 + float64(i)
		trend[i] = domainbt.Candle{High: c + 1, Low: c - 1, Close: c, Volume: 100}
	}
	_, _, _, vrTrend, _, _ := VolMetrics(trend, 1, 14)
	if vrTrend <= 1 {
		t.Errorf("trending VR(2) = %v, want > 1", vrTrend)
	}

	// Mean-reverting VR must be below trending VR.
	if vrOsc >= vrTrend {
		t.Errorf("oscillating VR(2) = %v must be < trending VR(2) = %v", vrOsc, vrTrend)
	}
}

func TestRenderVolatilityMarkdown_SortsByScore(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Name: "Alpha Co", MeanATRpct: 1.0, LastATRpct: 1.5, TurnoverM: 200, VR2: 0.9, Autocorr1: -0.1, Score: 0.20, Bars: 120},
		{Ticker: "BBB", Name: "Beta Co", MeanATRpct: 3.0, LastATRpct: 2.0, TurnoverM: 50, VR2: 0.7, Autocorr1: -0.3, Score: 0.80, Bars: 120},
	}
	meta := VolMeta{Months: 6, ATRPeriod: 14, MinTurnover: 50, MaxVR: 1.05, WVol: 0.4, WRev: 0.4, WLiq: 0.2, Scanned: 100, Passed: 2}

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
	for _, want := range []string{"Alpha Co", "Beta Co", "Score", "VR(2)", "Autocorr", "Вердикт"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output; got: %q", want, out)
		}
	}
}

func TestRenderVolatilityMarkdown_TopN(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Score: 0.1, Bars: 120},
		{Ticker: "BBB", Score: 0.9, Bars: 120},
		{Ticker: "CCC", Score: 0.5, Bars: 120},
	}
	out := RenderVolatilityMarkdown(rows, VolMeta{}, 2)
	if strings.Contains(out, "AAA") {
		t.Errorf("topN=2 must drop AAA (lowest score); out=%q", out)
	}
	if !strings.Contains(out, "BBB") || !strings.Contains(out, "CCC") {
		t.Errorf("topN=2 must keep BBB and CCC")
	}
}

func TestScoreVolRows_RewardsAllThree(t *testing.T) {
	rows := []VolRow{
		// strong: high ATR%, low VR2 (mean-reverting), high turnover
		{Ticker: "GOOD", MeanATRpct: 4.0, VR2: 0.7, TurnoverM: 500},
		// weak: low ATR%, high-ish VR2, low turnover
		{Ticker: "WEAK", MeanATRpct: 1.0, VR2: 1.0, TurnoverM: 50},
		// middle
		{Ticker: "MID", MeanATRpct: 2.5, VR2: 0.9, TurnoverM: 200},
	}
	ScoreVolRows(rows, 0.4, 0.4, 0.2)

	byTicker := map[string]float64{}
	for _, r := range rows {
		byTicker[r.Ticker] = r.Score
	}
	if !(byTicker["GOOD"] > byTicker["MID"] && byTicker["MID"] > byTicker["WEAK"]) {
		t.Errorf("score order wrong: GOOD=%.3f MID=%.3f WEAK=%.3f",
			byTicker["GOOD"], byTicker["MID"], byTicker["WEAK"])
	}
	// best on all dims ⇒ max blended score 1.0
	if byTicker["GOOD"] < 0.999 {
		t.Errorf("GOOD score = %.3f, want ~1.0 (top on every dimension)", byTicker["GOOD"])
	}
}

func TestScoreVolRows_WeightShiftsOrder(t *testing.T) {
	// A is more volatile; B mean-reverts harder. Weighting reversion flips the winner.
	mk := func() []VolRow {
		return []VolRow{
			{Ticker: "A", MeanATRpct: 5.0, VR2: 0.95, TurnoverM: 100},
			{Ticker: "B", MeanATRpct: 2.0, VR2: 0.60, TurnoverM: 100},
		}
	}
	volHeavy := mk()
	ScoreVolRows(volHeavy, 0.9, 0.1, 0.0)
	if volHeavy[0].Score <= volHeavy[1].Score {
		t.Errorf("vol-heavy weights: A must outscore B")
	}
	revHeavy := mk()
	ScoreVolRows(revHeavy, 0.1, 0.9, 0.0)
	if revHeavy[1].Score <= revHeavy[0].Score {
		t.Errorf("rev-heavy weights: B must outscore A")
	}
}

func TestScoreVolRows_SingleRow(t *testing.T) {
	rows := []VolRow{{Ticker: "ONLY", MeanATRpct: 2.0, VR2: 0.8, TurnoverM: 100}}
	ScoreVolRows(rows, 0.4, 0.4, 0.2) // must not panic; single candidate scores top
	if rows[0].Score == 0 {
		t.Errorf("single row score = 0, want > 0")
	}
}
