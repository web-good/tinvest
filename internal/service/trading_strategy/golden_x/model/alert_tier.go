package model

// AlertTier classifies a share's RSI position relative to its own adaptive
// percentiles. Buy tiers use the lower tail; sell tiers use the upper tail.
type AlertTier int

const (
	TierNone       AlertTier = iota
	TierYellow               // buy: RSI < p15
	TierGreen                // buy: RSI < p5
	TierSellYellow           // sell (Gold only): RSI > p80
	TierSellOrange           // sell: RSI > p90
	TierSellRed              // sell (Gold only): RSI > p95
)
