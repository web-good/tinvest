package core

import (
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// manage exits the open position, in order on the current M30 bar:
//   - the stop by the bar's low (SL); the engine fills it at min(stop, open);
//   - the target by the bar's high (TP); the engine fills it at max(target, open);
//   - the M30 RSI crossing RSIExit up on this bar (RSI), at its close: the RSI entered the upper
//     zone.
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
	if s.p.RSIPeriod <= 0 || k <= s.p.RSIPeriod || s.p.RSIExit <= 0 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if crossUp(rsi[k-1], rsi[k], s.p.RSIExit) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.2f → %.2f через %.0f вверх на M30 %s, выход по close %.4f (вход %.4f)",
			s.p.RSIPeriod, rsi[k-1], rsi[k], s.p.RSIExit, md.Times[k].In(mskLoc).Format("15:04"), md.Closes[k], pos.PurchasePrice)
		return sig
	}
	return sig
}

// exitLevels rebuilds the stop and the target off the fill: the stop StopATR entry daily ATRs (the
// ATR the signal carried) under the fill, at most MaxStopPct percent of the fill, the target TargetR
// stop distances above it (0 — none). Without an entry ATR the stop is the one the signal carried
// (off the trigger close).
func (s *Strategy) exitLevels(pos *strategy.Position) (stop, target float64, ok bool) {
	f := pos.PurchasePrice
	stop = pos.StopLoss
	if pos.EntryATR > 0 && s.p.StopATR > 0 {
		stop = StopLevel(s.p, f, pos.EntryATR)
	}
	if f <= 0 || stop <= 0 || stop >= f {
		return 0, 0, false
	}
	if s.p.TargetR > 0 {
		target = f + s.p.TargetR*(f-stop)
	}
	return stop, target, true
}
