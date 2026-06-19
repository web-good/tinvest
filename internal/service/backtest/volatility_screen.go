package backtest

import (
	"fmt"
	"sort"
	"strings"

	"tinvest/internal/domain/backtest"
	"tinvest/pkg/indicators"
)

// VolRow is one ticker's daily-ATR volatility result.
type VolRow struct {
	Ticker     string
	Name       string  // company / instrument name
	MeanATRpct float64 // mean ATR% over the window (headline ranking metric)
	LastATRpct float64 // latest ATR% (regime: heating up vs cooling)
	TurnoverM  float64 // mean daily turnover in millions of RUB
	Bars       int
}

// VolMeta carries the run parameters shown in the report header.
type VolMeta struct {
	Months      int
	ATRPeriod   int
	MinTurnover float64
	Scanned     int // universe size after the currency/trading filter
	Passed      int // rows that cleared the liquidity/history filter
}

// VolMetrics computes the daily-ATR volatility metrics for one ticker from its
// daily candle slice. meanATRpct/lastATRpct are 0 when there is not enough
// history for a valid ATR series (len < atrPeriod+1). turnoverM is the mean of
// volume*lot*close across all candles, in millions of RUB. vr2 is the variance
// ratio at lag 2 (mean-reversion metric); autocorr1 is lag-1 autocorrelation of
// simple returns (negative for mean-reverting, positive for trending).
func VolMetrics(candles []backtest.Candle, lot int32, atrPeriod int) (meanATRpct, lastATRpct, turnoverM, vr2, autocorr1 float64, bars int) {
	bars = len(candles)
	if bars == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	highs := make([]float64, bars)
	lows := make([]float64, bars)
	closes := make([]float64, bars)
	turnoverSum := 0.0
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
		turnoverSum += float64(c.Volume) * float64(lot) * c.Close
	}
	turnoverM = turnoverSum / float64(bars) / 1e6

	returns := backtest.SimpleReturns(closes)
	vr2 = backtest.VarianceRatio(returns, 2)
	autocorr1 = backtest.Autocorr1(returns)

	series := indicators.ATRSeries(highs, lows, closes, atrPeriod)
	pctSum := 0.0
	count := 0
	for i, atr := range series {
		if atr > 0 && closes[i] > 0 {
			pct := atr / closes[i] * 100
			pctSum += pct
			count++
			lastATRpct = pct
		}
	}
	if count == 0 {
		return 0, 0, turnoverM, vr2, autocorr1, bars
	}
	meanATRpct = pctSum / float64(count)
	return meanATRpct, lastATRpct, turnoverM, vr2, autocorr1, bars
}

// RenderVolatilityMarkdown renders the volatility screen as a Markdown table
// ranked by MeanATRpct descending (most volatile first). When topN > 0 the
// table is truncated to the top N rows.
func RenderVolatilityMarkdown(rows []VolRow, meta VolMeta, topN int) string {
	sorted := make([]VolRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].MeanATRpct > sorted[j].MeanATRpct
	})
	if topN > 0 && len(sorted) > topN {
		sorted = sorted[:topN]
	}

	var b strings.Builder
	b.WriteString("# Волатильность акций по дневному ATR\n\n")
	fmt.Fprintf(&b, "Окно: %d мес; ATR(%d) на дневном ТФ; порог ликвидности: %.0f млн ₽/день.\n",
		meta.Months, meta.ATRPeriod, meta.MinTurnover)
	fmt.Fprintf(&b, "Просканировано %d тикеров (RUB, торгуемые); прошло фильтр: %d.\n\n",
		meta.Scanned, meta.Passed)
	b.WriteString("Метрика — ATR% = ATR / цена. Ранжир по средней ATR% за окно (убыв.).\n\n")
	b.WriteString("| # | Тикер | Название | Ср. ATR% | Тек. ATR% | Тренд | Ликвидность, млн ₽/день | Баров |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for i, r := range sorted {
		trend := "↓"
		if r.LastATRpct > r.MeanATRpct {
			trend = "↑"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.2f | %.2f | %s | %.1f | %d |\n",
			i+1, r.Ticker, r.Name, r.MeanATRpct, r.LastATRpct, trend, r.TurnoverM, r.Bars)
	}
	return b.String()
}
