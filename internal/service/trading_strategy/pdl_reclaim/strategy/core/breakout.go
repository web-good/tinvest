package core

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// isC: bar i is green and closed above PDH, the bar before it, on the same date, closed at or
// below PDH — the first close above yesterday's high.
func isC(ser series, i int, pdh float64) bool {
	if i < 1 || dayKey(ser.t[i]) != dayKey(ser.t[i-1]) {
		return false
	}
	return ser.c[i] > ser.o[i] && ser.c[i] > pdh && ser.c[i-1] <= pdh
}

// strength reports the body of bar i of ser, how far it closed past level, and whether both are
// strictly above BodyATR daily ATRs; with level 0 (a reversal away from the levels) only the body
// counts. The filter is off (always passes) when BodyATR is 0.
func (s *Strategy) strength(ser series, i int, level, atr float64) (body, past float64, ok bool) {
	body = ser.c[i] - ser.o[i]
	if level > 0 {
		past = ser.c[i] - level
	}
	if s.p.BodyATR <= 0 {
		return body, past, true
	}
	limit := s.p.BodyATR * atr
	return body, past, atr > 0 && body > limit && (level <= 0 || past > limit)
}

// breakoutHeld reports whether the held position is a PDH breakout (entry C): with EntryC on, it
// was bought above today's PDH. An A/B position cannot fill above PDH and survive the target, so
// the price tells them apart while the target is on.
func (s *Strategy) breakoutHeld(md strategy.MarketData, pos *strategy.Position) bool {
	if s.p.EntryC <= 0 {
		return false
	}
	_, pdh, ok := s.levels(md)
	return ok && pos.PurchasePrice > pdh
}
