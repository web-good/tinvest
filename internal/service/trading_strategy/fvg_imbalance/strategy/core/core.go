// Package core implements the pure, long-only M30 FVG imbalance strategy.
// MarketData must contain completed candles only, ordered oldest first.
package core

import (
	"fmt"
	"math"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds numeric fields suitable for reflection-based parameter grids.
type Params struct {
	ATRPeriod                    int
	MinGapATR                    float64
	MaxRetestBars                int
	MinRetestDepthPct            float64
	TakeProfitR                  float64
	RequireThirdCandleHigherHigh int
	RetestDepthByBody            int
	UseStochFilter               int
	HTFTrendEMA                  int
	StopBufferATR                float64
}

// DefaultParams returns the baseline M30 FVG settings.
func DefaultParams() Params {
	return Params{ATRPeriod: 14, MinGapATR: .25, MaxRetestBars: 8, MinRetestDepthPct: 50, TakeProfitR: 1.5, RequireThirdCandleHigherHigh: 1, RetestDepthByBody: 1, UseStochFilter: 0}
}

// ValidateParams rejects invalid domains, including non-finite grid values.
func ValidateParams(p Params) error {
	switch {
	case p.RequireThirdCandleHigherHigh != 0 && p.RequireThirdCandleHigherHigh != 1:
		return fmt.Errorf("RequireThirdCandleHigherHigh must be 0 or 1")
	case p.RetestDepthByBody != 0 && p.RetestDepthByBody != 1:
		return fmt.Errorf("RetestDepthByBody must be 0 or 1")
	case p.UseStochFilter != 0 && p.UseStochFilter != 1:
		return fmt.Errorf("UseStochFilter must be 0 or 1")
	case p.HTFTrendEMA < 0:
		return fmt.Errorf("HTFTrendEMA must be non-negative")
	case p.StopBufferATR < 0 || math.IsNaN(p.StopBufferATR) || math.IsInf(p.StopBufferATR, 0):
		return fmt.Errorf("StopBufferATR must be finite and non-negative")
	case p.ATRPeriod <= 0:
		return fmt.Errorf("ATRPeriod must be positive")
	case p.MaxRetestBars <= 0:
		return fmt.Errorf("MaxRetestBars must be positive")
	case p.MinGapATR < 0 || math.IsNaN(p.MinGapATR) || math.IsInf(p.MinGapATR, 0):
		return fmt.Errorf("MinGapATR must be finite and non-negative")
	case p.TakeProfitR < 0 || math.IsNaN(p.TakeProfitR) || math.IsInf(p.TakeProfitR, 0):
		return fmt.Errorf("TakeProfitR must be finite and non-negative")
	case p.MinRetestDepthPct < 0 || p.MinRetestDepthPct > 100 || math.IsNaN(p.MinRetestDepthPct) || math.IsInf(p.MinRetestDepthPct, 0):
		return fmt.Errorf("MinRetestDepthPct must be finite and in 0..100")
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

// HTFDecisionOffset makes completed H4 data visible at the M30 decision close.
func (s *Strategy) HTFDecisionOffset() time.Duration { return s.step }

// Lookback retains the complete formation ATR history through the last eligible
// retest, together with all three formation candles even when ATRPeriod is 1.
func (s *Strategy) Lookback() int {
	formation := max(3, s.atrBars())
	if s.p.UseStochFilter == 1 {
		formation = max(formation, 14)
	}
	return formation + s.p.MaxRetestBars
}

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
		if md.Closes[b] <= md.Opens[b] || (s.p.RequireThirdCandleHigherHigh == 1 && md.Highs[c] <= md.Highs[b]) || lower >= upper || md.Opens[b] > lower || md.Closes[b] < upper {
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

// htfTrendOK evaluates the completed higher-timeframe closes supplied by the engine.
func (s *Strategy) htfTrendOK(md strategy.MarketData) (ok bool, close, value float64) {
	if s.p.HTFTrendEMA == 0 {
		return true, 0, 0
	}
	if len(md.HTFCloses) < s.p.HTFTrendEMA {
		return false, 0, 0
	}
	for _, v := range md.HTFCloses {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false, 0, 0
		}
	}
	close = md.HTFCloses[len(md.HTFCloses)-1]
	values := ema.Compute(md.HTFCloses, s.p.HTFTrendEMA)
	value = values[len(values)-1]
	return !math.IsNaN(value) && !math.IsInf(value, 0) && close > value, close, value
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
		emergency := pos.StopLoss - 2*s.p.StopBufferATR*pos.EntryATR
		if md.Lows[k] <= emergency {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL_EMERGENCY", emergency
			sig.ExitReason = fmt.Sprintf("FVG emergency stop %.4f", emergency)
			return sig
		}
		if md.Closes[k] <= pos.StopLoss {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL_CLOSE", pos.StopLoss
			sig.ExitReason = fmt.Sprintf("FVG close-confirmed stop %.4f", pos.StopLoss)
			return sig
		}
		target := pos.PurchasePrice + s.p.TakeProfitR*(pos.PurchasePrice-pos.StopLoss)
		if s.p.TakeProfitR > 0 && md.Highs[k] >= target {
			sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", target
			sig.ExitReason = fmt.Sprintf("FVG target %.4f (%.2fR from fill)", target, s.p.TakeProfitR)
			return sig
		}
		return sig
	}
	zones := s.discover(md)
	var ks []float64
	if s.p.UseStochFilter == 1 {
		ks, _ = indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, 14, 3)
	}
	for i := len(zones) - 1; i >= 0; i-- {
		z := zones[i]
		age := k - z.formed
		if age < 1 || age > s.p.MaxRetestBars {
			continue
		}
		invalid, depthReached, confirmedAt := false, false, -1
		for j := z.formed + 1; j <= k; j++ {
			if md.Lows[j] < z.lower {
				invalid = true
				break
			}
			if j != k && s.p.UseStochFilter == 0 {
				continue
			}
			if !depthReached {
				depthPrice := md.Lows[j]
				if s.p.RetestDepthByBody == 1 {
					depthPrice = min(md.Opens[j], md.Closes[j])
				}
				depthReached = md.Lows[j] <= z.upper && depthPrice <= z.upper-s.p.MinRetestDepthPct/100*(z.upper-z.lower)
			}
			// %K is right-aligned: ks[0] belongs to candle 13.
			if s.p.UseStochFilter == 1 && depthReached && confirmedAt < 0 && j >= 13 && ks[j-13] < 20 {
				confirmedAt = j
			}
		}
		if invalid || !depthReached {
			continue
		}
		if s.p.UseStochFilter == 1 && confirmedAt != k {
			// A prior confirmation had one exact next-open slot; never carry it forward.
			continue
		}
		htfOK, htfClose, htfEMA := s.htfTrendOK(md)
		if !htfOK {
			continue
		}
		fillBy := md.Times[k].Add(s.step)
		if day := fillBy.Weekday(); day == time.Saturday || day == time.Sunday {
			return sig
		}
		stop := z.lower - s.p.StopBufferATR*z.atr
		if stop <= 0 || math.IsNaN(stop) || math.IsInf(stop, 0) {
			continue
		}
		sig.Kind = model.SignalBuy
		sig.StopLoss, sig.ATR, sig.Level = stop, z.atr, z.upper
		if s.p.TakeProfitR > 0 {
			sig.TakeProfit = md.Closes[k] + s.p.TakeProfitR*(md.Closes[k]-sig.StopLoss)
		}
		sig.FillBy = fillBy
		sig.EntryReason = fmt.Sprintf("M30 bullish FVG [%.4f, %.4f] protected retest, ATR %.4f", z.lower, z.upper, z.atr)
		if s.p.HTFTrendEMA > 0 || s.p.StopBufferATR > 0 {
			sig.EntryReason += fmt.Sprintf("; stop %.4f", sig.StopLoss)
		}
		if s.p.HTFTrendEMA > 0 {
			sig.EntryReason += fmt.Sprintf("; H4 close %.4f > EMA(%d) %.4f", htfClose, s.p.HTFTrendEMA, htfEMA)
		}
		return sig
	}
	return sig
}
