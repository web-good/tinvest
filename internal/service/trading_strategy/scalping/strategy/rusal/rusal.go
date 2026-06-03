package rusal

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const ticker = "RUAL"

// Strategy trades RUSAL with an EMA trend filter, an upward RSI reversal entry,
// and ATR-based take-profit / stop-loss.
type Strategy struct {
	emaPeriod        int
	rsiPeriod        int
	atrPeriod        int
	rsiReversalLevel float64
	tpMult           float64
	slMult           float64
}

// New returns the RUSAL strategy with its default knobs.
func New() *Strategy {
	return &Strategy{
		emaPeriod:        200,
		rsiPeriod:        14,
		atrPeriod:        14,
		rsiReversalLevel: 35,
		tpMult:           1.5,
		slMult:           1.0,
	}
}

func (s *Strategy) Ticker() string { return ticker }

// Lookback is the candle count needed to seed the EMA plus headroom.
func (s *Strategy) Lookback() int { return s.emaPeriod + 50 }

// Decide computes the indicators from md and applies the RUSAL rule.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	closes := md.Closes
	n := len(closes)

	emaSeries := ema.Compute(closes, s.emaPeriod)
	rsiSeries := indicators.RSISeries(closes, s.rsiPeriod)
	atr := indicators.ATR(md.Highs, md.Lows, closes, s.atrPeriod)

	aboveEMA := n > 0 && emaSeries[n-1] > 0 && md.Price > emaSeries[n-1]

	var rsiPrev, rsiNow float64
	if n >= 2 {
		rsiNow = rsiSeries[n-1]
		rsiPrev = rsiSeries[n-2]
	}

	sig := s.decide(md.Price, atr, aboveEMA, rsiPrev, rsiNow, md.Position)
	sig.Ticker = ticker
	return sig
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(price, atr float64, aboveEMA bool, rsiPrev, rsiNow float64, pos *strategy.Position) model.Signal {
	sig := model.Signal{Price: price, RSI: rsiNow}

	if pos != nil {
		tp := pos.PurchasePrice + s.tpMult*atr
		sl := pos.PurchasePrice - s.slMult*atr
		sig.TakeProfit = tp
		sig.StopLoss = sl
		switch {
		case price >= tp:
			sig.Kind = model.SignalSell
			sig.Reason = "TP"
		case price <= sl:
			sig.Kind = model.SignalSell
			sig.Reason = "SL"
		}
		return sig
	}

	if aboveEMA && rsiPrev < s.rsiReversalLevel && rsiNow >= s.rsiReversalLevel {
		sig.Kind = model.SignalBuy
		sig.TakeProfit = price + s.tpMult*atr
		sig.StopLoss = price - s.slMult*atr
	}
	return sig
}
