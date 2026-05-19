package dto

// Signal is the pure output of golden_x.Detect for a single share against a
// closed-week candle history. Backtest replay derives entry/exit decisions
// from this directly; service.Trade composes it with dedup + Telegram.
type Signal struct {
	RSI            float64
	LastClose      float64
	Stop           Stop
	Thresholds     Thresholds
	SellThresholds SellThresholds
	TrendStatus    TrendStatus // TrendUnknown if useTrendFilter was false
	GreenBuy       bool        // RSI < Thresholds.P5
	YellowBuy      bool        // RSI < Thresholds.P15 && !GreenBuy
	DivergenceOK   bool        // bullish RSI divergence on the buy side
	VolumeOK       bool        // last-week volume > multiplier × SMA20
}
