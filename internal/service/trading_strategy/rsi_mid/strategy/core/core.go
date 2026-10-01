// Package core implements rsi_mid, a long-only multi-day strategy that buys a resumption of
// momentum: a short RSI crossing UP through its midline 50 on the current bar while the close
// sits above a trend EMA. It sells when the protective stop — sized in daily ATR at entry and
// frozen on the position — is touched, when RSI crosses UP through its upper band, or when RSI
// has read below 50 on BelowMidBars consecutive bars after the entry bar. There is no target,
// no trail and no end-of-day close: the position is held across nights and weekends until an
// exit fires. The decision logic is pure, stateless between bars and ticker-agnostic. Every
// period is counted in bars, so a literal is tied to the timeframe it was calibrated on: ticker
// packages hold one literal per interval and the backtest picks it by -interval. Intended as a
// companion to rsi_pullback on a shared account: pullback buys deep dips, rsi_mid the shallow
// continuation pullback never reaches. Run with `-strategy rsi_mid -interval <Interval>`.
package core

import (
	"fmt"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback floors the candle window at 120 bars so the RSI always sees enough history even
// with a short trend EMA; with the usual EMA lengths 2·EMAPeriod+20 dominates.
const minLookback = 120

// midLine is the RSI midline whose upward cross opens a trade and whose underside counts toward
// the MID exit. Fixed by design, not a tunable.
const midLine = 50.0

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod      int     // RSI length (grid: entry; default 14)
	RSIUpper       float64 // upper band; an UPWARD cross of it is the RSI exit (grid: exit; default 70)
	BelowMidBars   int     // exit once RSI read below 50 on this many consecutive bars after the entry bar; 0 disables it (grid: exit; default 2)
	EMAPeriod      int     // trend EMA period; the entry needs close > EMA; <= 0 refuses every entry (grid: trend; default 100)
	DailyATRPeriod int     // daily ATR length, over WEEKDAY completed dailies (fixed; default 14)
	StopDailyATR   float64 // stop = entry - StopDailyATR*dailyATR; 0 disables it (grid: risk; never 0 in the grid)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      14,
		RSIUpper:       70,
		BelowMidBars:   2,
		EMAPeriod:      100,
		DailyATRPeriod: 14,
		StopDailyATR:   0.5,
	}
}

// Strategy trades a single instrument with the rsi_mid rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. ema.Compute seeds on
// an SMA over the first `period` closes, so a window shorter than the period yields an all-zero
// series that silently fails the trend gate. Doubling the largest period leaves as many
// recursion steps as the seed span; the +20 covers the two-bar cross lookups.
func (s *Strategy) Lookback() int {
	return max(minLookback, 2*max(s.p.EMAPeriod, s.p.RSIPeriod)+20)
}

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter emits a long when RSI crosses UP through 50 on the current bar while the close sits
// above a warmed trend EMA on a weekday and the daily ATR is known. Everything is recomputed
// from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n || s.p.RSIPeriod <= 0 || s.p.EMAPeriod <= 0 {
		return sig
	}
	// 1. weekday: any time of a trading day will do, weekends will not.
	if !tradingDay(barTime(md)) {
		return sig
	}
	i := n - 1
	// 2. the trigger: RSI crosses up through the midline on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n || !crossedUp(rsi, i, s.p.RSIPeriod, midLine) {
		return sig
	}
	// 3. trend: close above a warmed EMA.
	trend := ema.Compute(md.Closes, s.p.EMAPeriod)
	if len(trend) != n || !trendUp(md.Closes[i], trend[i]) {
		return sig
	}
	// 4. the daily ATR is the unit of the stop: no ATR, no trade.
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	// 5. an armed stop must land above zero, otherwise the long would be unprotected.
	entry := md.Closes[i]
	stop := StopLevel(s.p, entry, atr)
	if s.p.StopDailyATR > 0 && stop <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i-1], rsi[i], trend[i], entry, stop, atr)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiPrev, rsiNow, emaNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен"
	if stop > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (−%.2f ATR)", stop, s.p.StopDailyATR)
	}
	return fmt.Sprintf(
		"RSI(%d) пересёк 50 снизу вверх (%.1f → %.1f), close %.4f > EMA(%d) %.4f, дневной ATR %.4f; вход %.4f, %s",
		s.p.RSIPeriod, rsiPrev, rsiNow, entry, s.p.EMAPeriod, emaNow, atr, entry, stopHow,
	)
}
