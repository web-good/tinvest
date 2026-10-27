package core

import "tinvest/internal/service/trading_strategy/scalping/strategy"

// prevDayLow returns the low of the last weekday daily before the date of bar h (yesterday's low,
// PDL); ok is false without such a daily.
func prevDayLow(md strategy.MarketData, h int) (pdl float64, ok bool) {
	if len(md.DailyLows) != len(md.DailyTimes) {
		return 0, false
	}
	day := dayKey(md.Times[h])
	for i := len(md.DailyTimes) - 1; i >= 0; i-- {
		t := md.DailyTimes[i]
		if dayKey(t) < day && !isWeekend(t.In(mskLoc)) {
			return md.DailyLows[i], true
		}
	}
	return 0, false
}

// atPDL reports whether the low of the hammer at bar h lies within PDLZoneATR daily ATRs of
// yesterday's low, either side. It returns PDL; PDLZoneATR 0 — off (always passes).
func (s *Strategy) atPDL(md strategy.MarketData, h int, atr float64) (pdl float64, ok bool) {
	if s.p.PDLZoneATR <= 0 {
		return 0, true
	}
	pdl, ok = prevDayLow(md, h)
	if !ok || atr <= 0 {
		return 0, false
	}
	d := s.p.PDLZoneATR * atr
	low := md.Lows[h]
	return pdl, low >= pdl-d && low <= pdl+d
}
