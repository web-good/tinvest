package backtest

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// syntheticSeries builds n 5m bars and their aligned 1h HTF series, oldest-first,
// covering the same order of magnitude as a multi-month SBER backtest (6 months of
// 5m bars on an ~8h trading day is roughly 17k bars / 1.4k hourly bars).
func syntheticSeries(bars int) (m5 []Candle, h1 []Candle) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	m5 = make([]Candle, bars)
	for i := 0; i < bars; i++ {
		px := 100 + float64(i%50)
		m5[i] = Candle{Time: base.Add(time.Duration(i) * 5 * time.Minute), Open: px, High: px + 1, Low: px - 1, Close: px, Volume: 100}
	}
	hourly := bars / 12
	h1 = make([]Candle, hourly)
	for i := 0; i < hourly; i++ {
		px := 100 + float64(i%50)
		h1[i] = Candle{Time: base.Add(time.Duration(i) * time.Hour), Open: px, High: px + 1, Low: px - 1, Close: px, Volume: 1000}
	}
	return m5, h1
}

// noopStrategy never trades; it isolates Run's per-bar assembly cost from portfolio
// bookkeeping.
type noopStrategy struct{}

func (noopStrategy) Ticker() string { return "BENCH" }
func (noopStrategy) Lookback() int  { return 50 }
func (noopStrategy) Decide(md strategy.MarketData) model.Signal {
	return model.Signal{Kind: model.SignalNone}
}

// BenchmarkRunWithHTF measures Run's per-bar cost with an attached H1 series (the
// scalping_rsimacd HTFTrendEMA gate path) — this is what visibleCompletedHTF's
// full-rescan-per-bar made ~12x slower than the no-HTF path before the htfCursor fix.
func BenchmarkRunWithHTF(b *testing.B) {
	m5, h1 := syntheticSeries(17280) // ~6 months of 5m bars on an 8h trading day
	cfg := Config{InitialCash: 100000, Fraction: 1.0, Lot: 1, HTFInterval: time.Hour}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Run(noopStrategy{}, m5, nil, h1, cfg)
	}
}

// BenchmarkRunWithoutHTF is the same run with htf=nil, i.e. a strategy that never
// enables the HTF gate — must cost the same regardless of the htfCursor fix.
func BenchmarkRunWithoutHTF(b *testing.B) {
	m5, _ := syntheticSeries(17280)
	cfg := Config{InitialCash: 100000, Fraction: 1.0, Lot: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Run(noopStrategy{}, m5, nil, nil, cfg)
	}
}

// syntheticDailies builds `days` daily bars aligned with syntheticSeries' start — the
// series rsi_pullback attaches to every run (два года дневок ради сходимости сида Уайлдера).
func syntheticDailies(days int) []Candle {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]Candle, days)
	for i := 0; i < days; i++ {
		px := 100 + float64(i%50)
		out[i] = Candle{Time: base.AddDate(0, 0, i-days+1), Open: px, High: px + 5, Low: px - 5, Close: px, Volume: 10000}
	}
	return out
}

// BenchmarkRunWithDailies измеряет путь дневной серии — тот, которым ходит rsi_pullback на
// каждом баре. До dailyCursor visibleDaily пересканировала все 730 дневок на каждом из
// 17280 баров и аллоцировала четыре среза на бар: 1596 мс против 279 мс с курсором на этой
// машине (5.7x), при 273 мс у пути вообще без дневных.
func BenchmarkRunWithDailies(b *testing.B) {
	m5, _ := syntheticSeries(17280)
	daily := syntheticDailies(730)
	cfg := Config{InitialCash: 100000, Fraction: 1.0, Lot: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Run(noopStrategy{}, m5, daily, nil, cfg)
	}
}
