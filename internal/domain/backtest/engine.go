package backtest

import (
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
		md.Position = p.strategyPosition()

		c := candles[i]
		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR)
			}
		case model.SignalSell:
			if p.qty != 0 {
				exitPrice := c.Close
				// Stop exits fill at the stop level, adjusted for a gap-down open:
				// min(level, open) lands inside the bar (always >= c.Low) and charges
				// real gap risk when the bar opened below the stop.
				if sig.Reason == "SL" || sig.Reason == "TRAIL" {
					exitPrice = min(sig.StopLoss, c.Open)
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
