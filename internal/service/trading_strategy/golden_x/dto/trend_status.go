package dto

// TrendStatus marks a share's position relative to the higher-trend filter
// (EMA200 on the same weekly timeframe as the strategy).
type TrendStatus int

const (
	TrendUnknown TrendStatus = iota
	TrendWith                // close > EMA200_W
	TrendAgainst             // close <= EMA200_W
)

// Mark returns the Telegram emoji to render next to the share name.
// Empty string means "no marker" — used either when the filter was not applied
// for this strategy instance (Gold) or before any value has been set.
func (t TrendStatus) Mark() string {
	switch t {
	case TrendWith:
		return "✅"
	case TrendAgainst:
		return "🚫"
	default:
		return ""
	}
}
