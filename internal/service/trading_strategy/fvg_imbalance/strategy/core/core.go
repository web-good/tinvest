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
	ATRPeriod             int
	MinGapATR             float64
	MaxRetestBars         int
	RequireCloseAboveZone int
	TakeProfitR           float64
	RequireFirstRetest    int
	RequireBullishBOS     int
	RequireLiquiditySweep int
	SwingK                int
	StructureLookbackBars int
	MaxSweepToFVGBars     int
	HTFTrendEMA           int
	MaxEntryDistanceATR   float64
	MaxHoldBars           int
	StopBufferATR         float64
}

// DefaultParams returns the baseline M30 FVG settings.
func DefaultParams() Params {
	return Params{ATRPeriod: 14, MinGapATR: .25, MaxRetestBars: 8, RequireCloseAboveZone: 1, TakeProfitR: 1.5, SwingK: 2, StructureLookbackBars: 48, MaxSweepToFVGBars: 8}
}

// ValidateParams rejects invalid domains, including non-finite grid values.
func ValidateParams(p Params) error {
	switch {
	case p.RequireFirstRetest != 0 && p.RequireFirstRetest != 1:
		return fmt.Errorf("RequireFirstRetest must be 0 or 1")
	case p.RequireBullishBOS != 0 && p.RequireBullishBOS != 1:
		return fmt.Errorf("RequireBullishBOS must be 0 or 1")
	case p.RequireLiquiditySweep != 0 && p.RequireLiquiditySweep != 1:
		return fmt.Errorf("RequireLiquiditySweep must be 0 or 1")
	case p.SwingK <= 0:
		return fmt.Errorf("SwingK must be positive")
	case p.StructureLookbackBars <= 0:
		return fmt.Errorf("StructureLookbackBars must be positive")
	case p.MaxSweepToFVGBars <= 0:
		return fmt.Errorf("MaxSweepToFVGBars must be positive")
	case p.HTFTrendEMA < 0:
		return fmt.Errorf("HTFTrendEMA must be non-negative")
	case p.MaxHoldBars < 0:
		return fmt.Errorf("MaxHoldBars must be non-negative")
	case p.MaxEntryDistanceATR < 0 || math.IsNaN(p.MaxEntryDistanceATR) || math.IsInf(p.MaxEntryDistanceATR, 0):
		return fmt.Errorf("MaxEntryDistanceATR must be finite and non-negative")
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
func (s *Strategy) Lookback() int {
	formation := max(3, s.atrBars())
	if s.p.RequireBullishBOS == 1 || s.p.RequireLiquiditySweep == 1 {
		formation = max(formation, s.p.StructureLookbackBars+2*s.p.SwingK+3)
	}
	return formation + s.p.MaxRetestBars
}

func swingHighAt(highs []float64, i, k int) bool {
	if k <= 0 || i < k || i >= len(highs) || k >= len(highs)-i {
		return false
	}
	for j := i - k; j <= i+k; j++ {
		if j != i && !(highs[i] > highs[j]) {
			return false
		}
	}
	return true
}

func swingLowAt(lows []float64, i, k int) bool {
	if k <= 0 || i < k || i >= len(lows) || k >= len(lows)-i {
		return false
	}
	for j := i - k; j <= i+k; j++ {
		if j != i && !(lows[i] < lows[j]) {
			return false
		}
	}
	return true
}

// atrBars fixes the Wilder seed relative to the formation, never to the moving
// beginning of MarketData. Two periods of candles provide the seed and period-1
// smoothing steps. Every reconstruction of a zone therefore uses identical bars.
func (s *Strategy) atrBars() int { return 2 * s.p.ATRPeriod }

type zone struct {
	lower, upper, atr float64
	formed            int
}

// structurePasses reconstructs event-time structure using only completed bars.
func (s *Strategy) structurePasses(md strategy.MarketData, z zone) bool {
	if s.p.RequireLiquiditySweep == 0 && s.p.RequireBullishBOS == 0 {
		return true
	}
	c, k := z.formed, s.p.SwingK
	if k <= 0 || c < 2 || c >= len(md.Closes) || c >= len(md.Highs) || c >= len(md.Lows) {
		return false
	}
	start := max(0, c-s.p.StructureLookbackBars)
	latest := func(values []float64, event int, high bool) int {
		for i := event - k; i >= start; i-- {
			if high && swingHighAt(values, i, k) || !high && swingLowAt(values, i, k) {
				return i
			}
		}
		return -1
	}
	sweep := -1
	if s.p.RequireLiquiditySweep == 1 {
		for event := c - 1; event >= start && c-event <= s.p.MaxSweepToFVGBars; event-- {
			pivot := latest(md.Lows, event-1, false)
			if pivot >= 0 && md.Lows[event] < md.Lows[pivot] && md.Closes[event] > md.Lows[pivot] {
				sweep = event
				break
			}
		}
		if sweep < 0 {
			return false
		}
	}
	if s.p.RequireBullishBOS == 0 {
		return true
	}
	referenceAt, breakFrom := c-2, c-1
	if sweep >= 0 {
		referenceAt, breakFrom = sweep, sweep
	}
	pivot := latest(md.Highs, referenceAt, true)
	if pivot < 0 {
		return false
	}
	for i := breakFrom; i <= c; i++ {
		if md.Closes[i] > md.Highs[pivot] {
			return true
		}
	}
	return false
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
			if md.Lows[j] <= z.lower || (s.p.RequireFirstRetest == 1 && j < k && md.Lows[j] <= z.upper) {
				invalid = true
				break
			}
		}
		if invalid || !s.structurePasses(md, z) || md.Lows[k] > z.upper || (s.p.RequireCloseAboveZone == 1 && md.Closes[k] <= z.upper) {
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
		if s.p.MaxEntryDistanceATR > 0 {
			sig.MaxEntryPrice = z.upper + s.p.MaxEntryDistanceATR*z.atr
		}
		if s.p.TakeProfitR > 0 {
			sig.TakeProfit = md.Closes[k] + s.p.TakeProfitR*(md.Closes[k]-sig.StopLoss)
		}
		sig.FillBy = fillBy
		sig.EntryReason = fmt.Sprintf("M30 bullish FVG [%.4f, %.4f] protected retest, ATR %.4f", z.lower, z.upper, z.atr)
		if s.p.RequireFirstRetest == 1 {
			sig.EntryReason += "; first retest"
		}
		if s.p.RequireLiquiditySweep == 1 {
			sig.EntryReason += "; liquidity sweep passed"
		}
		if s.p.RequireBullishBOS == 1 {
			sig.EntryReason += "; bullish BOS passed"
		}
		if s.p.RequireFirstRetest == 1 || s.p.RequireLiquiditySweep == 1 || s.p.RequireBullishBOS == 1 || s.p.HTFTrendEMA > 0 || s.p.MaxEntryDistanceATR > 0 || s.p.StopBufferATR > 0 {
			sig.EntryReason += fmt.Sprintf("; stop %.4f", sig.StopLoss)
		}
		if s.p.MaxEntryDistanceATR > 0 {
			sig.EntryReason += fmt.Sprintf("; max entry %.4f", sig.MaxEntryPrice)
		}
		if s.p.HTFTrendEMA > 0 {
			sig.EntryReason += fmt.Sprintf("; H4 close %.4f > EMA(%d) %.4f", htfClose, s.p.HTFTrendEMA, htfEMA)
		}
		return sig
	}
	return sig
}
