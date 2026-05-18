package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// atrMultiplierDividend is the ATR stop multiplier for the Dividend
// (long-hold) strategy: wider stops survive deeper weekly noise.
const atrMultiplierDividend = 2.0

// atrMultiplierGrowth is the stop multiplier for Growth — tighter, since the
// strategy exits sooner on RSI overheats.
const atrMultiplierGrowth = 1.5

// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Dividend holds longer and needs wider stops; Growth exits sooner and uses
// tighter stops. Unknown kinds fall back to Dividend — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind) float64 {
	if kind == dto.StrategyKindGrowth {
		return atrMultiplierGrowth
	}
	return atrMultiplierDividend
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
