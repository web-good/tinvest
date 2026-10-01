package core

import (
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// manage handles an open long. It exits on one of three signals, evaluated in precedence order
// SL → RSI → MID. The stop triggers INTRABAR (the bar's low touching the level), because a real
// stop order fills as soon as price trades through it; the engine prices that fill via
// model.IsStopReason. It wins a same-bar tie: the intrabar order is unknowable from OHLC, and
// assuming the worse outcome is the honest choice. The stop level is rebuilt from the entry
// price and the daily ATR frozen at entry, never from the current ATR. RSI and MID fill at the
// bar close and cannot coincide (RSI above RSIUpper versus RSI below 50 on the same bar —
// RSIUpper below 50 is not a meaningful setting). There is no calendar gate on exits.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low, closeP := md.Lows[i], md.Closes[i]

	// 1. protective stop, frozen at entry.
	if level := StopLevel(s.p, pos.PurchasePrice, pos.EntryATR); level > 0 && low <= level {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = level
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, level, pos.PurchasePrice)
		return sig
	}
	if s.p.RSIPeriod <= 0 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		return sig
	}
	// 2. RSI crosses UP through the upper band — the move reached the upper zone.
	if crossedUp(rsi, i, s.p.RSIPeriod, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
		return sig
	}
	// 3. RSI has read below the midline on BelowMidBars consecutive bars after the entry bar.
	if s.p.BelowMidBars > 0 {
		if run := s.belowMidRun(md, rsi); run >= s.p.BelowMidBars {
			sig.Kind, sig.Reason = model.SignalSell, "MID"
			sig.RSI = rsi[i]
			sig.ExitReason = fmt.Sprintf("MID: RSI(%d) %d бар(ов) подряд ниже 50 (%.1f), выход по %.4f (вход %.4f)",
				s.p.RSIPeriod, run, rsi[i], closeP, pos.PurchasePrice)
		}
	}
	return sig
}

// belowMidRun counts the trailing bars, ending at the current one, on which RSI read strictly
// below the midline. The run stops at the first bar at or above 50, at an unwarmed RSI index, or
// at the entry bar — the last bar that opened at or before Position.EntryTime, which never
// counts itself. When every bar of the window opened after EntryTime, the entry bar has left the
// window and the whole window counts: on short timeframes the default window is shorter than a
// multi-day hold, and going silent there would drop the exit for exactly those trades. Without
// EntryTime or aligned Times there is no anchor and the result is 0.
func (s *Strategy) belowMidRun(md strategy.MarketData, rsi []float64) int {
	pos, n := md.Position, len(md.Closes)
	if pos == nil || pos.EntryTime.IsZero() || len(md.Times) != n || len(rsi) != n {
		return 0
	}
	first := 0 // first bar strictly after the entry bar
	for b := n - 1; b >= 0; b-- {
		if !md.Times[b].After(pos.EntryTime) {
			first = b + 1
			break
		}
	}
	run := 0
	for b := n - 1; b >= first; b-- {
		if b < s.p.RSIPeriod || rsi[b] >= midLine {
			break
		}
		run++
	}
	return run
}
