package model

// ShareResult holds all per-share data needed to render one line in the
// notification message. It replaces the previous pattern of parallel maps
// (thresholds, sellThresholds, trends, divergences, volumesConfirmed).
type ShareResult struct {
	InstrumentName   string
	RSI              float64
	BuyTier          AlertTier
	SellTier         AlertTier
	Thresholds       Thresholds
	SellThresholds   SellThresholds
	TrendStatus      TrendStatus
	DivergenceOK     bool
	VolumeOK         bool
	Score            int
	Sector           string
	FundamentalBonus int
}

// TradeResult aggregates the buy/sell signals for a single Trade tick. It is
// the sole argument to notification.Trade, replacing the previous 8-parameter
// call.
type TradeResult struct {
	BuyShares       map[string]ShareResult
	SellShares      map[string]ShareResult
	CappedBuyShares map[string]ShareResult
	Kind            StrategyKind
}
