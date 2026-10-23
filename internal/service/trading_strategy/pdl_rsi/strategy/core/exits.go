package core

import (
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// manage exits the open position, in order on the current bar:
//   - the stop by the bar's low: the stop frozen at entry (SL), lifted to the fill once a completed
//     hour after the fill crossed RSIBreakeven up before this bar (TRAIL); the engine fills both at
//     min(StopLoss, open);
//   - the hourly RSI crossing RSIExit up on the hour this bar completes (RSI), at the bar's close.
//
// No target and no end-of-day exit: the position carries overnight and over weekends.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	pos := md.Position
	if n == 0 || len(md.Times) != n || len(md.Lows) != n {
		return sig
	}
	var hs []hourBar
	var rh []float64
	if s.p.RSIPeriod > 0 {
		hs = s.hourBars(md)
		rh = indicators.RSISeries(hourCloses(hs), s.p.RSIPeriod)
	}
	low := md.Lows[n-1]
	stop, code := pos.StopLoss, "SL"
	if s.breakevenHit(md, hs, rh) {
		stop, code = max(stop, pos.PurchasePrice), "TRAIL"
	}
	if stop > 0 && low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, code, stop
		if code == "TRAIL" {
			sig.ExitReason = fmt.Sprintf("TRAIL: low %.4f ≤ стоп %.4f (безубыток после креста часового RSI(%d) через %.0f; вход %.4f, исходный стоп %.4f)",
				low, stop, s.p.RSIPeriod, s.p.RSIBreakeven, pos.PurchasePrice, pos.StopLoss)
		} else {
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		}
		return sig
	}
	if s.p.RSIExit <= 0 {
		return sig
	}
	// Hours are ordered by done; more than one completes on this bar only around a missing bar.
	for h := len(hs) - 1; h > s.p.RSIPeriod && hs[h].done == n-1; h-- {
		if md.Times[hs[h].last].Before(pos.EntryTime) || !crossUp(rh[h-1], rh[h], s.p.RSIExit) {
			continue
		}
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.ExitReason = fmt.Sprintf("RSI: часовой RSI(%d) %.2f → %.2f через %.0f на часе %s (закрыт на свече %s), выход по close %.4f (вход %.4f)",
			s.p.RSIPeriod, rh[h-1], rh[h], s.p.RSIExit, hs[h].start.In(mskLoc).Format("15:04"),
			md.Times[n-1].In(mskLoc).Format("15:04"), md.Closes[n-1], pos.PurchasePrice)
		return sig
	}
	return sig
}

// breakevenHit reports whether the stop has moved to the fill: RSIBreakeven > 0 and some hour
// completed before the current bar, ending at or after the fill bar, crossed RSIBreakeven up. An
// EntryTime older than the window lets every hour qualify. Hours start counting at RSIPeriod + 1,
// so both RSIs of the cross are real values: the step from the warm-up zero at RSIPeriod - 1 to
// the first value at RSIPeriod is no cross.
func (s *Strategy) breakevenHit(md strategy.MarketData, hs []hourBar, rh []float64) bool {
	if s.p.RSIBreakeven <= 0 || md.Position == nil {
		return false
	}
	n := len(md.Closes)
	entry := md.Position.EntryTime
	for h := len(hs) - 1; h > s.p.RSIPeriod; h-- {
		if md.Times[hs[h].last].Before(entry) {
			return false // older hours end even earlier
		}
		if hs[h].done < n-1 && crossUp(rh[h-1], rh[h], s.p.RSIBreakeven) {
			return true
		}
	}
	return false
}
