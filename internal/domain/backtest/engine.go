package backtest

import (
	"fmt"
	"strings"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// mskLoc anchors the trading-day boundary used to decide which daily candles are
// already closed at a given intraday bar. Fallback to UTC if the tz DB is absent.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// startOfDay returns midnight of t's calendar day in loc.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	tl := t.In(loc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, loc)
}

// visibleDailyCloses returns closes of daily candles whose calendar day (in loc) is
// strictly before t's calendar day — i.e. days that have fully closed by t. This is
// the no-lookahead rule: the current, still-forming day is never visible.
func visibleDailyCloses(daily []Candle, t time.Time, loc *time.Location) []float64 {
	bound := startOfDay(t, loc)
	out := make([]float64, 0, len(daily))
	for _, c := range daily {
		if c.Time.Before(bound) {
			out = append(out, c.Close)
		}
	}
	return out
}

// visibleDailyHighsLows returns highs and lows of daily candles whose calendar day
// (in loc) is strictly before t's calendar day — the same completed days as
// visibleDailyCloses, so the three series are index-aligned.
func visibleDailyHighsLows(daily []Candle, t time.Time, loc *time.Location) (highs, lows []float64) {
	bound := startOfDay(t, loc)
	for _, c := range daily {
		if c.Time.Before(bound) {
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
		}
	}
	return highs, lows
}

// todayExtent returns the high and low across all bars sharing candles[i]'s MSK
// calendar day, scanning back from i only (no lookahead). Returns (0,0) when i is
// out of range.
func todayExtent(candles []Candle, i int, loc *time.Location) (high, low float64) {
	if i < 0 || i >= len(candles) {
		return 0, 0
	}
	day := startOfDay(candles[i].Time, loc)
	high, low = candles[i].High, candles[i].Low
	for j := i - 1; j >= 0; j-- {
		if startOfDay(candles[j].Time, loc).Before(day) {
			break
		}
		if candles[j].High > high {
			high = candles[j].High
		}
		if candles[j].Low < low {
			low = candles[j].Low
		}
	}
	return high, low
}

// Run replays a strategy over oldest-first candles on a mock portfolio and
// returns the raw result. It mirrors the live runner: per bar it builds a
// lookback-sized MarketData, calls Decide, and acts only on Buy (when flat) or
// Sell (when in position), filling at the bar's close. An open position at the
// end is marked-to-market, never force-closed.
func Run(s strategy.Strategy, candles []Candle, dailyCandles []Candle, cfg Config) Result {
	res := Result{InitialCash: cfg.InitialCash, FinalEquity: cfg.InitialCash}
	l := s.Lookback()
	if l <= 0 || len(candles) < l {
		return res
	}
	p := newPortfolio(cfg)
	lastClose := candles[len(candles)-1].Close
	for i := l - 1; i < len(candles); i++ {
		p.bar = i
		md := buildMarketData(candles[i-l+1 : i+1])
		md.DailyCloses = visibleDailyCloses(dailyCandles, candles[i].Time, mskLoc)
		md.DailyHighs, md.DailyLows = visibleDailyHighsLows(dailyCandles, candles[i].Time, mskLoc)
		md.TodayHigh, md.TodayLow = todayExtent(candles, i, mskLoc)
		if p.qty != 0 {
			p.mark(candles[i].Close)
		}
		md.Position = p.strategyPosition()

		c := candles[i]
		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss, sig.EntryReason)
			}
		case model.SignalSell:
			if p.qty != 0 {
				exitPrice := c.Close
				// Stop exits fill at the stop level, adjusted for a gap-down open:
				// min(level, open) lands inside the bar and charges real gap risk.
				// TP exits fill at the target, adjusted for a gap-up open: max(target,
				// open) is the limit fill and rewards a gap through the target.
				switch sig.Reason {
				case "SL", "TRAIL":
					exitPrice = min(sig.StopLoss, c.Open)
				case "TP":
					if sig.TakeProfit > 0 {
						exitPrice = max(sig.TakeProfit, c.Open)
					}
				}
				res.Trades = append(res.Trades, p.close(exitPrice, c.Time, sig.Reason))
			}
		}

		res.Equity = append(res.Equity, EquityPoint{Time: c.Time, Equity: p.equity(c.Close)})
		if p.qty != 0 {
			res.BarsInMarket++
		}
		lastClose = c.Close
	}
	res.FinalEquity = p.equity(lastClose)
	return res
}

// explainer is the optional gate-level diagnostic a strategy may implement.
type explainer interface {
	Explain(md strategy.MarketData) string
}

// Trace replays the strategy exactly like Run but, at the bar whose timestamp
// equals target, captures the live position state and (if the strategy
// implements explainer) the gate-by-gate verdict. It returns a human-readable
// diagnostic. Used to answer "why did/didn't it act at this bar?". Replaying
// from the start means the reported position state is the real one.
func Trace(s strategy.Strategy, candles []Candle, dailyCandles []Candle, cfg Config, target time.Time) string {
	l := s.Lookback()
	if l <= 0 || len(candles) < l {
		return "недостаточно свечей для lookback"
	}
	p := newPortfolio(cfg)
	for i := l - 1; i < len(candles); i++ {
		p.bar = i
		md := buildMarketData(candles[i-l+1 : i+1])
		md.DailyCloses = visibleDailyCloses(dailyCandles, candles[i].Time, mskLoc)
		md.DailyHighs, md.DailyLows = visibleDailyHighsLows(dailyCandles, candles[i].Time, mskLoc)
		md.TodayHigh, md.TodayLow = todayExtent(candles, i, mskLoc)
		if p.qty != 0 {
			p.mark(candles[i].Close)
		}
		md.Position = p.strategyPosition()

		c := candles[i]
		if c.Time.Equal(target) {
			var sb strings.Builder
			fmt.Fprintf(&sb, "Бар %s (MSK %s)\n", c.Time.Format(time.RFC3339), c.Time.In(mskLoc).Format("2006-01-02 15:04"))
			fmt.Fprintf(&sb, "OHLC: O=%.4f H=%.4f L=%.4f C=%.4f V=%d\n", c.Open, c.High, c.Low, c.Close, c.Volume)
			if p.qty != 0 {
				fmt.Fprintf(&sb, "Состояние: В ПОЗИЦИИ (qty=%d, вход %.4f)\n", p.qty, p.entryPrice)
			} else {
				sb.WriteString("Состояние: вне позиции (flat)\n")
			}
			sb.WriteString("--- фильтры входа ---\n")
			// Advance per-bar strategy state (e.g. the momentum cooldown counter) for
			// the target bar exactly as Run does, so Explain reads the same state the
			// real Decide path sees on this bar. The returned signal is discarded; it
			// is a no-op for stateless strategies.
			s.Decide(md)
			if ex, ok := s.(explainer); ok {
				sb.WriteString(ex.Explain(md))
			} else {
				sb.WriteString("стратегия не поддерживает Explain")
			}
			return sb.String()
		}

		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss, sig.EntryReason)
			}
		case model.SignalSell:
			if p.qty != 0 {
				exitPrice := c.Close
				switch sig.Reason {
				case "SL", "TRAIL":
					exitPrice = min(sig.StopLoss, c.Open)
				case "TP":
					if sig.TakeProfit > 0 {
						exitPrice = max(sig.TakeProfit, c.Open)
					}
				}
				p.close(exitPrice, c.Time, sig.Reason)
			}
		}
	}
	return fmt.Sprintf("бар с временем %s не найден в свечах", target.Format(time.RFC3339))
}

// buildMarketData converts an oldest-first window into a strategy snapshot,
// mirroring scalping/trade.go's buildMarketData.
func buildMarketData(window []Candle) strategy.MarketData {
	md := strategy.MarketData{
		Highs:   make([]float64, len(window)),
		Lows:    make([]float64, len(window)),
		Closes:  make([]float64, len(window)),
		Volumes: make([]int64, len(window)),
	}
	for i, c := range window {
		md.Highs[i] = c.High
		md.Lows[i] = c.Low
		md.Closes[i] = c.Close
		md.Volumes[i] = c.Volume
	}
	if n := len(window); n > 0 {
		md.Price = window[n-1].Close
	}
	return md
}
