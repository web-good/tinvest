package backtest

import (
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// Run replays a strategy over oldest-first candles on a mock portfolio and
// returns the raw result. It mirrors the live runner: per bar it builds a
// lookback-sized MarketData, calls Decide, and acts only on Buy (when flat) or
// Sell (when in position), filling at the bar's close. An open position at the
// end is marked-to-market, never force-closed.
func Run(s strategy.Strategy, candles []Candle, cfg Config) Result {
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
		md.Position = p.strategyPosition()

		c := candles[i]
		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time)
			}
		case model.SignalSell:
			if p.qty != 0 {
				res.Trades = append(res.Trades, p.close(c.Close, c.Time, sig.Reason))
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
