package backtest

import (
	"fmt"
	"sort"
	"strings"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/domain/ema"
	"tinvest/pkg/indicators"
)

// VolRow is one ticker's Hour1 reversion-fitness result.
type VolRow struct {
	Ticker       string
	Name         string  // company / instrument name
	RecoveryRate float64 // share of pullback-in-trend dips that recover within RecoverBars
	EventFreq    float64 // pullback events per 1000 bars
	Events       int     // completed pullback events
	MeanATRpct   float64 // mean Hour1 ATR% (reference column, not scored)
	LastATRpct   float64 // latest Hour1 ATR% (reference)
	TurnoverM    float64 // mean daily turnover in millions of RUB
	VR2          float64 // Lo-MacKinlay variance ratio at q=2 (reference column, not scored)
	Autocorr1    float64 // lag-1 autocorrelation (reference)
	Score        float64 // composite reversion-fitness score (filled by ScoreVolRows)
	Bars         int
}

// VolParams holds the pullback-metric configuration for one screening run.
type VolParams struct {
	ATRPeriod   int     // ATR period for the reference ATR% column
	EMAPeriod   int     // trend EMA period (uptrend context, e.g. 200)
	RSIPeriod   int     // RSI period for the dip trigger
	RSIOversold float64 // oversold threshold RSI crosses down through
	RecoverRSI  float64 // RSI level that marks a recovered dip
	RecoverBars int     // bars allowed for a dip to recover
}

// VolStats are the per-ticker metrics computed by VolMetrics.
type VolStats struct {
	MeanATRpct   float64
	LastATRpct   float64
	TurnoverM    float64
	VR2          float64
	Autocorr1    float64
	RecoveryRate float64
	EventFreq    float64
	Events       int
	Bars         int
}

// VolMeta carries the run parameters shown in the report header.
type VolMeta struct {
	Months      int
	ATRPeriod   int
	MinTurnover float64
	RSIPeriod   int
	RSIOversold float64
	RecoverBars int
	RecoverRSI  float64
	WRecov      float64 // composite weights
	WFreq       float64
	WLiq        float64
	Scanned     int // universe size after the currency/trading filter
	Passed      int // rows that cleared liquidity/history filters (scored)
}

// VolMetrics computes the Hour1 reversion-fitness metrics for one ticker. The
// headline metrics are RecoveryRate/EventFreq (pullback-in-trend opportunity);
// ATR%, VR2 and Autocorr1 are reference-only. TurnoverM is the mean daily
// turnover (Volume*lot*Close summed per calendar day, averaged), in millions.
func VolMetrics(candles []backtest.Candle, lot int32, p VolParams) VolStats {
	out := VolStats{Bars: len(candles)}
	if out.Bars == 0 {
		return out
	}

	highs := make([]float64, out.Bars)
	lows := make([]float64, out.Bars)
	closes := make([]float64, out.Bars)
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}
	out.TurnoverM = backtest.MeanDailyTurnoverM(candles, lot)

	returns := backtest.SimpleReturns(closes)
	out.VR2 = backtest.VarianceRatio(returns, 2)
	out.Autocorr1 = backtest.Autocorr1(returns)

	atrSeries := indicators.ATRSeries(highs, lows, closes, p.ATRPeriod)
	pctSum := 0.0
	count := 0
	for i, atr := range atrSeries {
		if atr > 0 && closes[i] > 0 {
			pct := atr / closes[i] * 100
			pctSum += pct
			count++
			out.LastATRpct = pct
		}
	}
	if count > 0 {
		out.MeanATRpct = pctSum / float64(count)
	}

	trendEMA := ema.Compute(closes, p.EMAPeriod)
	rsi := indicators.RSISeries(closes, p.RSIPeriod)
	out.RecoveryRate, out.EventFreq, out.Events = backtest.PullbackStats(
		closes, trendEMA, rsi, p.RSIOversold, p.RecoverRSI, p.RecoverBars,
	)
	return out
}

// percentileRanks maps each value to its fractional rank in [0,1]: the smallest
// value gets 0, the largest 1, ties share a rank. A single value scores 1 (it is
// the sole — hence top — candidate).
func percentileRanks(vals []float64) []float64 {
	n := len(vals)
	out := make([]float64, n)
	if n == 1 {
		out[0] = 1
		return out
	}
	for i, v := range vals {
		less := 0
		for _, w := range vals {
			if w < v {
				less++
			}
		}
		out[i] = float64(less) / float64(n-1)
	}
	return out
}

// ScoreVolRows fills each row's Score with a weighted blend of percentile ranks
// across recovery rate, event frequency and liquidity (turnover). ATR% and VR2
// are reference-only and do NOT contribute. Percentiles are relative to the rows
// passed in.
func ScoreVolRows(rows []VolRow, wRecov, wFreq, wLiq float64) {
	n := len(rows)
	if n == 0 {
		return
	}
	recov := make([]float64, n)
	freq := make([]float64, n)
	liq := make([]float64, n)
	for i, r := range rows {
		recov[i] = r.RecoveryRate
		freq[i] = r.EventFreq
		liq[i] = r.TurnoverM
	}
	pRecov, pFreq, pLiq := percentileRanks(recov), percentileRanks(freq), percentileRanks(liq)
	for i := range rows {
		rows[i].Score = wRecov*pRecov[i] + wFreq*pFreq[i] + wLiq*pLiq[i]
	}
}

// RenderVolatilityMarkdown renders the reversion-fitness screen as a Markdown
// table ranked by Score descending. When topN > 0 the table is truncated to the
// top N rows.
func RenderVolatilityMarkdown(rows []VolRow, meta VolMeta, topN int) string {
	sorted := make([]VolRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})
	if topN > 0 && len(sorted) > topN {
		sorted = sorted[:topN]
	}

	var b strings.Builder
	b.WriteString("# Скринер пригодности под reversion (Hour1, pullback-in-trend)\n\n")
	fmt.Fprintf(&b, "Окно: %d мес (Hour1); EMA-тренд + RSI(%d), отскок: вход при пересечении вниз %.0f, успех если RSI > %.0f за %d баров; порог ликвидности: %.0f млн ₽/день.\n",
		meta.Months, meta.RSIPeriod, meta.RSIOversold, meta.RecoverRSI, meta.RecoverBars, meta.MinTurnover)
	fmt.Fprintf(&b, "Просканировано %d тикеров (RUB, торгуемые); прошло фильтр: %d.\n\n", meta.Scanned, meta.Passed)
	fmt.Fprintf(&b, "Score = %.2g·перцентиль(восстановление) + %.2g·перцентиль(частота) + %.2g·перцентиль(оборот). ATR%%/VR(2) — справочные, в Score не входят. Ранжир по Score (убыв.).\n\n",
		meta.WRecov, meta.WFreq, meta.WLiq)
	b.WriteString("| # | Тикер | Название | Score | Восстановл., % | События/1000 | Кол-во соб. | Оборот, млн ₽/день | ATR% (H1) | VR(2) | Autocorr | Баров |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range sorted {
		fmt.Fprintf(&b, "| %d | %s | %s | %.3f | %.1f | %.2f | %d | %.1f | %.2f | %.3f | %+.3f | %d |\n",
			i+1, r.Ticker, r.Name, r.Score, r.RecoveryRate*100, r.EventFreq, r.Events,
			r.TurnoverM, r.MeanATRpct, r.VR2, r.Autocorr1, r.Bars)
	}
	return b.String()
}
