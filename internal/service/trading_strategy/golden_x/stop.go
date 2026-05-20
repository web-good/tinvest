package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Dividend holds longer and needs wider stops; Growth exits sooner and uses
// tighter stops. Unknown kinds fall back to Dividend — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind, settings dto.Settings) float64 {
	if kind == dto.StrategyKindGrowth {
		return settings.ATRMultiplierGrowth
	}
	return settings.ATRMultiplierDividend
}

// stopFromATR composes a dto.Stop from the last close, ATR value, and
// multiplier. Returns the zero Stop{} when atr or lastClose are non-positive,
// or when the computed stop level would be <= 0 (degenerate).
func stopFromATR(lastClose, atr, k float64) dto.Stop {
	if atr <= 0 || lastClose <= 0 {
		return dto.Stop{}
	}
	price := lastClose - k*atr
	if price <= 0 {
		return dto.Stop{}
	}
	return dto.Stop{
		Price:       price,
		DistancePct: (lastClose - price) / lastClose * 100,
	}
}
