package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// manage exits the open position, in order on the current hour:
//   - the stop by the hour's low (SL); the engine fills it at min(stop, open);
//   - the target by the hour's high (TP); the engine fills it at max(target, open);
//   - the hourly RSI crossing RSIExit down on this hour (RSI), at its close: the RSI left the upper
//     zone;
//   - the RSI ladder (RSITR), at its close: see trailLevel.
//
// Both levels come from exitLevels. No end-of-day exit: the position carries overnight and over
// weekends.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	pos := md.Position
	if n == 0 || len(md.Times) != n || len(md.Lows) != n || len(md.Highs) != n {
		return sig
	}
	if stop, target, ok := s.exitLevels(pos); ok {
		if low := md.Lows[n-1]; low <= stop {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
			return sig
		}
		if high := md.Highs[n-1]; target > 0 && high >= target {
			sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", target
			sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ цели %.4f = вход %.4f + %.2f R", high, target, pos.PurchasePrice, s.p.TargetR)
			return sig
		}
	}
	k := n - 1
	if md.Times[k].Before(pos.EntryTime) {
		return sig
	}
	if s.p.HourRSIPeriod <= 0 || k <= s.p.HourRSIPeriod || (s.p.RSIExit <= 0 && s.p.RSITrailFrom <= 0) {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, s.p.HourRSIPeriod)
	if s.p.RSIExit > 0 && crossDown(rsi[k-1], rsi[k], s.p.RSIExit) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: часовой RSI(%d) %.2f → %.2f через %.0f вниз на часе %s, выход по close %.4f (вход %.4f)",
			s.p.HourRSIPeriod, rsi[k-1], rsi[k], s.p.RSIExit, md.Times[k].In(mskLoc).Format("15:04"), md.Closes[k], pos.PurchasePrice)
		return sig
	}
	if rung, level, ok := s.trailLevel(md.Times, rsi, k, pos); ok && crossDown(rsi[k-1], rsi[k], level) {
		sig.Kind, sig.Reason = model.SignalSell, "RSITR"
		sig.ExitReason = fmt.Sprintf("RSITR: часовой RSI(%d) дошёл до %.0f, %.2f → %.2f через %.0f вниз на часе %s, выход по close %.4f (вход %.4f)",
			s.p.HourRSIPeriod, rung, rsi[k-1], rsi[k], level, md.Times[k].In(mskLoc).Format("15:04"), md.Closes[k], pos.PurchasePrice)
	}
	return sig
}

// trailLevel is the RSI ladder: the rungs run from RSITrailFrom to RSITrailTo every RSITrailStep;
// once a closed hour of the position (from the fill hour, before hour k) has an RSI at or above a
// rung, the exit level is that rung minus RSITrailStep, and the highest rung reached wins (with
// 50/70/10: reached 50 — exit on a cross of 40 down, 60 — of 50, 70 — of 60). The level only
// ratchets up. ok is false while no rung is reached or the ladder is off (RSITrailFrom 0).
func (s *Strategy) trailLevel(times []time.Time, rsi []float64, k int, pos *strategy.Position) (rung, level float64, ok bool) {
	if s.p.RSITrailFrom <= 0 || s.p.RSITrailStep <= 0 {
		return 0, 0, false
	}
	top := max(s.p.RSITrailTo, s.p.RSITrailFrom)
	peak := -1.0
	for j := k - 1; j >= 0 && !times[j].Before(pos.EntryTime); j-- {
		peak = max(peak, rsi[j])
	}
	if peak < s.p.RSITrailFrom {
		return 0, 0, false
	}
	rung = s.p.RSITrailFrom
	for rung+s.p.RSITrailStep <= top && peak >= rung+s.p.RSITrailStep {
		rung += s.p.RSITrailStep
	}
	return rung, rung - s.p.RSITrailStep, true
}

// exitLevels rebuilds the stop and the target off the fill: the stop StopATR entry daily ATRs (the
// ATR the signal carried) under the fill, at most MaxStopPct percent of the fill, the target TargetR
// stop distances above it (0 — none). Without an entry ATR the stop is the one the signal carried
// (off the trigger close).
func (s *Strategy) exitLevels(pos *strategy.Position) (stop, target float64, ok bool) {
	f := pos.PurchasePrice
	stop = pos.StopLoss
	if pos.EntryATR > 0 && s.p.StopATR > 0 {
		stop = f - s.stopDistance(f, pos.EntryATR)
	}
	if f <= 0 || stop <= 0 || stop >= f {
		return 0, 0, false
	}
	if s.p.TargetR > 0 {
		target = f + s.p.TargetR*(f-stop)
	}
	return stop, target, true
}
