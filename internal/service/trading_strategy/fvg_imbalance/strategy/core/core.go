// Package core implements the pure, long-only M30 FVG imbalance strategy.
// MarketData must contain completed candles only, ordered oldest first.
package core

import (
	"fmt"
	"math"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds numeric fields suitable for reflection-based parameter grids.
type Params struct {
	ATRPeriod             int
	MinGapATR             float64
	MaxRetestBars         int
	RequireCloseAboveZone int
	TakeProfitR           float64
}

// DefaultParams returns the baseline M30 FVG settings.
func DefaultParams() Params {
	return Params{ATRPeriod: 14, MinGapATR: .25, MaxRetestBars: 8, RequireCloseAboveZone: 1, TakeProfitR: 1.5}
}

// ValidateParams rejects invalid domains, including non-finite grid values.
func ValidateParams(p Params) error {
	switch {
	case p.ATRPeriod <= 0:
		return fmt.Errorf("ATRPeriod must be positive")
	case p.MaxRetestBars <= 0:
		return fmt.Errorf("MaxRetestBars must be positive")
	case p.MinGapATR < 0 || math.IsNaN(p.MinGapATR) || math.IsInf(p.MinGapATR, 0):
		return fmt.Errorf("MinGapATR must be finite and non-negative")
	case p.TakeProfitR < 0 || math.IsNaN(p.TakeProfitR) || math.IsInf(p.TakeProfitR, 0):
		return fmt.Errorf("TakeProfitR must be finite and non-negative")
	case p.RequireCloseAboveZone != 0 && p.RequireCloseAboveZone != 1:
		return fmt.Errorf("RequireCloseAboveZone must be 0 or 1")
	}
	return nil
}

// Strategy is stateless between decisions. Unsupported steps and invalid parameters fail closed.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams constructs a ticker-specific core; only a 30-minute step can trade.
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}
func (s *Strategy) Ticker() string { return s.ticker }

// Lookback retains the complete formation ATR history through the last eligible
// retest, together with all three formation candles even when ATRPeriod is 1.
func (s *Strategy) Lookback() int { return max(3, s.atrBars()) + s.p.MaxRetestBars }

// atrBars fixes the Wilder seed relative to the formation, never to the moving
// beginning of MarketData. Two periods of candles provide the seed and period-1
// smoothing steps. Every reconstruction of a zone therefore uses identical bars.
func (s *Strategy) atrBars() int { return 2 * s.p.ATRPeriod }

type zone struct {
	lower, upper, atr float64
	formed            int
}

func (s *Strategy) discover(md strategy.MarketData) []zone {
	n := len(md.Closes)
	if s.step != 30*time.Minute || ValidateParams(s.p) != nil || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n || len(md.Times) != n {
		return nil
	}
	var zones []zone
	for c := max(2, s.atrBars()-1); c < n; c++ {
		a, b := c-2, c-1
		if !md.Times[b].Equal(md.Times[a].Add(s.step)) || !md.Times[c].Equal(md.Times[b].Add(s.step)) {
			continue
		}
		lower, upper := md.Highs[a], md.Lows[c]
		if md.Closes[b] <= md.Opens[b] || lower >= upper || md.Opens[b] > lower || md.Closes[b] < upper {
			continue
		}
		start := c + 1 - s.atrBars()
		atr := indicators.ATR(md.Highs[start:c+1], md.Lows[start:c+1], md.Closes[start:c+1], s.p.ATRPeriod)
		if atr <= 0 || upper-lower < s.p.MinGapATR*atr {
			continue
		}
		zones = append(zones, zone{lower: lower, upper: upper, atr: atr, formed: c})
	}
	return zones
}

// Decide reconstructs zones from completed bars and never mutates the snapshot.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if s.step != 30*time.Minute || ValidateParams(s.p) != nil || n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	k := n - 1
	if pos := md.Position; pos != nil {
		if pos.StopLoss <= 0 {
			return sig
		}
		if pos.PurchasePrice <= pos.StopLoss || md.Lows[k] <= pos.StopLoss {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", pos.StopLoss
			sig.ExitReason = fmt.Sprintf("FVG stop %.4f", pos.StopLoss)
			return sig
		}
		target := pos.PurchasePrice + s.p.TakeProfitR*(pos.PurchasePrice-pos.StopLoss)
		if s.p.TakeProfitR > 0 && md.Highs[k] >= target {
			sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", target
			sig.ExitReason = fmt.Sprintf("FVG target %.4f (%.2fR from fill)", target, s.p.TakeProfitR)
		}
		return sig
	}
	zones := s.discover(md)
	for i := len(zones) - 1; i >= 0; i-- {
		z := zones[i]
		age := k - z.formed
		if age < 1 || age > s.p.MaxRetestBars {
			continue
		}
		invalid := false
		for j := z.formed + 1; j <= k; j++ {
			if md.Lows[j] <= z.lower {
				invalid = true
				break
			}
		}
		if invalid || md.Lows[k] > z.upper || (s.p.RequireCloseAboveZone == 1 && md.Closes[k] <= z.upper) {
			continue
		}
		fillBy := md.Times[k].Add(s.step)
		if day := fillBy.Weekday(); day == time.Saturday || day == time.Sunday {
			return sig
		}
		sig.Kind = model.SignalBuy
		sig.StopLoss, sig.ATR, sig.Level = z.lower, z.atr, z.upper
		if s.p.TakeProfitR > 0 {
			sig.TakeProfit = md.Closes[k] + s.p.TakeProfitR*(md.Closes[k]-z.lower)
		}
		sig.FillBy = fillBy
		sig.EntryReason = fmt.Sprintf("M30 bullish FVG [%.4f, %.4f] protected retest, ATR %.4f", z.lower, z.upper, z.atr)
		return sig
	}
	return sig
}
