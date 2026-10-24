package core

import (
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// manage exits the open position, in order on the current hour:
//   - the stop by the hour's low (SL); the engine fills it at min(stop, open);
//   - the target by the hour's high (TP); the engine fills it at max(target, open);
//   - the stochastic %K crossing %D down above StochExitAbove on this hour (STOCH), at its close;
//   - the hourly RSI crossing RSIExit up on this hour (RSI), at its close.
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
	reason, why, ok := s.softExit(md, s.exitIndicators(md.Closes, barsOf(md)), k)
	if !ok {
		return sig
	}
	sig.Kind, sig.Reason = model.SignalSell, reason
	sig.ExitReason = why
	return sig
}

// exitSeries holds the indicators of the close-based exits; a nil slice — that exit is off.
type exitSeries struct {
	rsi        []float64
	sk, sd     []float64
	stochFirst int
}

// exitIndicators computes the RSI of the RSI exit and the stochastic of the STOCH exit.
func (s *Strategy) exitIndicators(closes []float64, ser series) exitSeries {
	var ind exitSeries
	if s.p.HourRSIPeriod > 0 && s.p.RSIExit > 0 {
		ind.rsi = indicators.RSISeries(closes, s.p.HourRSIPeriod)
	}
	if s.p.StochExitAbove > 0 {
		ind.sk, ind.sd, ind.stochFirst = s.stochKD(ser)
	}
	return ind
}

// softExit reports whether hour m fires a close-based exit: the stochastic %K crossing %D down above
// StochExitAbove (STOCH) first, then the hourly RSI crossing RSIExit up (RSI). It returns the reason
// and the exit text.
func (s *Strategy) softExit(md strategy.MarketData, ind exitSeries, m int) (reason, why string, ok bool) {
	hm := md.Times[m].In(mskLoc).Format("15:04")
	if ind.sk != nil && m-1 >= ind.stochFirst && crossUnder(ind.sk, ind.sd, m) && ind.sk[m] > s.p.StochExitAbove {
		return "STOCH", fmt.Sprintf("STOCH: стохастик %%K %.2f → %.2f пересёк %%D %.2f → %.2f вниз выше %.0f на часе %s, выход по close %.4f (вход %.4f)",
			ind.sk[m-1], ind.sk[m], ind.sd[m-1], ind.sd[m], s.p.StochExitAbove, hm, md.Closes[m], md.Position.PurchasePrice), true
	}
	if ind.rsi != nil && m > s.p.HourRSIPeriod && crossUp(ind.rsi[m-1], ind.rsi[m], s.p.RSIExit) {
		return "RSI", fmt.Sprintf("RSI: часовой RSI(%d) %.2f → %.2f через %.0f на часе %s, выход по close %.4f (вход %.4f)",
			s.p.HourRSIPeriod, ind.rsi[m-1], ind.rsi[m], s.p.RSIExit, hm, md.Closes[m], md.Position.PurchasePrice), true
	}
	return "", "", false
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
