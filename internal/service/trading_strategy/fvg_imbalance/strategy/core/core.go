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
	ATRPeriod                 int
	MinGapATR                 float64
	MaxWaitBars               int
	StochPeriod               int
	UseStochExit              int
	StopATR                   float64
	TakeProfitATR             float64
	HTFTrendEMA               int
	RequireEntryAboveFVGLower int // 0 disables; 1 requires signal close and fill >= FVG lower bound.
}

func DefaultParams() Params {
	return Params{ATRPeriod: 14, MinGapATR: .25, MaxWaitBars: 30, StochPeriod: 14, UseStochExit: 1, StopATR: 1, TakeProfitATR: 2}
}

func ValidateParams(p Params) error {
	switch {
	case p.RequireEntryAboveFVGLower != 0 && p.RequireEntryAboveFVGLower != 1:
		return fmt.Errorf("invalid RequireEntryAboveFVGLower: expected 0 or 1")
	case p.ATRPeriod <= 0 || p.MaxWaitBars <= 0 || p.StochPeriod <= 0 || p.HTFTrendEMA < 0 || (p.UseStochExit != 0 && p.UseStochExit != 1):
		return fmt.Errorf("invalid period or waiting window")
	case p.MinGapATR < 0 || math.IsNaN(p.MinGapATR) || math.IsInf(p.MinGapATR, 0):
		return fmt.Errorf("invalid MinGapATR")
	case p.StopATR <= 0 || math.IsNaN(p.StopATR) || math.IsInf(p.StopATR, 0):
		return fmt.Errorf("invalid StopATR")
	case p.TakeProfitATR < 0 || math.IsNaN(p.TakeProfitATR) || math.IsInf(p.TakeProfitATR, 0):
		return fmt.Errorf("invalid TakeProfitATR")
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

// Lookback retains the formation, waiting window, and stochastic history.
// ATR history is supplied separately as daily candles.
func (s *Strategy) Lookback() int {
	return max(3+s.p.MaxWaitBars, s.p.StochPeriod+1)
}

// atrBars supplies the Wilder seed and period-1 smoothing steps for a daily ATR.
func (s *Strategy) atrBars() int { return 2 * s.p.ATRPeriod }

var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// dailyATRAt returns the ATR of weekday daily candles completed before the M30
// formation candle. DailyTimes is required to exclude candles completed later.
func (s *Strategy) dailyATRAt(md strategy.MarketData, formed time.Time) float64 {
	n := len(md.DailyCloses)
	if n == 0 || len(md.DailyHighs) != n || len(md.DailyLows) != n || len(md.DailyTimes) != n {
		return 0
	}
	local := formed.In(mskLoc)
	bound := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, mskLoc)
	highs, lows, closes := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	for i, day := range md.DailyTimes {
		if !day.Before(bound) {
			break
		}
		weekday := day.In(mskLoc).Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			continue
		}
		highs = append(highs, md.DailyHighs[i])
		lows = append(lows, md.DailyLows[i])
		closes = append(closes, md.DailyCloses[i])
	}
	if len(closes) < s.atrBars() {
		return 0
	}
	return indicators.ATR(highs, lows, closes, s.p.ATRPeriod)
}

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
	for c := 2; c < n; c++ {
		a, b := c-2, c-1
		if !md.Times[b].Equal(md.Times[a].Add(s.step)) || !md.Times[c].Equal(md.Times[b].Add(s.step)) {
			continue
		}
		lower, upper := md.Highs[a], md.Lows[c]
		if md.Closes[b] <= md.Opens[b] || lower >= upper || md.Opens[b] > lower || md.Closes[b] < upper {
			continue
		}
		atr := s.dailyATRAt(md, md.Times[c])
		if atr <= 0 || upper-lower < s.p.MinGapATR*atr {
			continue
		}
		zones = append(zones, zone{lower: lower, upper: upper, atr: atr, formed: c})
	}
	return zones
}

// htfTrendOK evaluates the completed daily closes supplied by the engine.
func (s *Strategy) htfTrendOK(md strategy.MarketData) (ok bool, close, value float64) {
	if s.p.HTFTrendEMA == 0 {
		return true, 0, 0
	}
	if len(md.DailyCloses) < s.p.HTFTrendEMA {
		return false, 0, 0
	}
	for _, v := range md.DailyCloses {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false, 0, 0
		}
	}
	close = md.DailyCloses[len(md.DailyCloses)-1]
	values := ema.Compute(md.DailyCloses, s.p.HTFTrendEMA)
	value = values[len(values)-1]
	return !math.IsNaN(value) && !math.IsInf(value, 0) && close > value, close, value
}

// crossedBelowSince reports whether a zone has already had an entry trigger.
// An FVG is a one-shot setup: later stochastic recrosses must not reuse it.
func crossedBelowSince(ks []float64, period, formed, current int) bool {
	first := max(formed+1, period)
	for bar := first; bar < current; bar++ {
		cur := bar - period + 1
		prev := cur - 1
		if prev >= 0 && ks[prev] >= 20 && ks[cur] < 20 {
			return true
		}
	}
	return false
}

// Decide evaluates completed candles without retaining mutable state.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if s.step != 30*time.Minute || ValidateParams(s.p) != nil || n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	k := n - 1
	ks, _ := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochPeriod, 1)
	if pos := md.Position; pos != nil {
		if pos.StopLoss > 0 && md.Lows[k] <= pos.StopLoss {
			sig.Kind = model.SignalSell
			sig.Reason = "SL"
			sig.StopLoss = pos.StopLoss
			sig.ExitReason = fmt.Sprintf("FVG stop %.4f", pos.StopLoss)
			return sig
		}
		if pos.TakeProfit > 0 && md.Highs[k] >= pos.TakeProfit {
			sig.Kind = model.SignalSell
			sig.Reason = "TP"
			sig.TakeProfit = pos.TakeProfit
			sig.ExitReason = fmt.Sprintf("FVG target %.4f", pos.TakeProfit)
			return sig
		}
		if s.p.UseStochExit == 1 && len(ks) >= 2 && ks[len(ks)-2] <= 80 && ks[len(ks)-1] > 80 {
			sig.Kind = model.SignalSell
			sig.Reason = "STOCH_EXIT"
			sig.ExitReason = fmt.Sprintf("FVG stochastic %%K(%d) crossed above 80", s.p.StochPeriod)
			return sig
		}
		return sig
	}
	if len(ks) < 2 || ks[len(ks)-2] < 20 || ks[len(ks)-1] >= 20 {
		return sig
	}
	zones := s.discover(md)
	for i := len(zones) - 1; i >= 0; i-- {
		z := zones[i]
		age := k - z.formed
		if age < 1 || age > s.p.MaxWaitBars {
			continue
		}
		if s.p.RequireEntryAboveFVGLower == 1 && md.Closes[k] < z.lower {
			continue
		}
		if crossedBelowSince(ks, s.p.StochPeriod, z.formed, k) {
			continue
		}
		if ok, _, _ := s.htfTrendOK(md); !ok {
			continue
		}
		fillBy := md.Times[k].Add(s.step)
		if day := fillBy.Weekday(); day == time.Saturday || day == time.Sunday {
			return sig
		}
		atr := s.dailyATRAt(md, md.Times[k])
		if atr <= 0 {
			continue
		}
		if s.p.RequireEntryAboveFVGLower == 1 {
			sig.MinEntryPrice = z.lower
		}
		sig.Kind = model.SignalBuy
		sig.ATR = atr
		sig.Level = z.upper
		sig.StopLoss = md.Closes[k] - s.p.StopATR*atr
		if sig.StopLoss <= 0 {
			return model.Signal{Ticker: s.ticker, Price: md.Price}
		}
		if s.p.TakeProfitATR > 0 {
			sig.TakeProfit = md.Closes[k] + s.p.TakeProfitATR*atr
		}
		sig.StopDistance = s.p.StopATR * atr
		sig.TargetDistance = s.p.TakeProfitATR * atr
		sig.FillBy = fillBy
		sig.EntryReason = fmt.Sprintf("M30 bullish FVG [%.4f, %.4f], stochastic crossed below 20", z.lower, z.upper)
		return sig
	}
	return sig
}
