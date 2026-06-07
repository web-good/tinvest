// Package core implements a long-only "levels" trading strategy. It builds a
// volume profile over an intraday window, extracts high-volume-node price levels,
// and trades bounces off support / retests of broken resistance with a chandelier
// trail, an ATR hard stop, a higher-timeframe downtrend filter, and trade-quality
// gates. The decision logic is pure and ticker-agnostic; per-share packages supply
// the ticker and calibrated Params.
package core

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/levels"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable for the levels strategy. All fields are int or
// float64 so they can be swept by reflection in grid calibration.
type Params struct {
	ProfileWindow     int     // intraday bars used to build the volume profile
	ProfileBins       int     // number of price bins in the profile
	HVNFactor         float64 // bin is a high-volume node when vol >= HVNFactor*mean
	ATRPeriod         int     // ATR period for stops/room/anti-churn
	LevelTolATR       float64 // proximity to a level counts as a touch within LevelTolATR*ATR
	RoomATR           float64 // require resistance above to be >= RoomATR*ATR away (runway)
	MaxExtensionATR   float64 // reject entry if price is > MaxExtensionATR*ATR above the support
	SLMult            float64 // initial stop = support - SLMult*ATR
	SwingLowWindow    int     // bars scanned for the structural low anchoring the hard stop
	TrailMult         float64 // chandelier = recentHigh(ChandelierWindow) - TrailMult*ATR
	ChandelierWindow  int     // window for the chandelier high
	TrailArmATR       float64 // trail arms only after price >= entry + TrailArmATR*ATR; <=0 arms immediately
	BreakoutLookback  int     // bars to look back to classify a breakout-retest vs a bounce
	TrendFilterPeriod int     // daily EMA period for the HTF downtrend filter; 0 disables
	TrendSlopeTol     float64 // downtrend = daily EMA falling faster than TrendSlopeTol (relative) AND price below it
	ConfirmCloseFrac  float64 // bullish confirmation: (close-low)/(high-low) >= ConfirmCloseFrac
	MinATRFrac        float64 // reject entry if ATR < MinATRFrac*price (anti-churn); <=0 disables
	MinRR             float64 // reject entry if (target-price) < MinRR*(price-stop); <=0 disables
}

// Strategy trades a single instrument off volume-profile levels. It is
// ticker-agnostic; per-share packages supply the ticker and calibrated Params.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the levels strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer: the profile
// window, the chandelier high, the ATR (period+1 for the seed) or the breakout
// lookback, plus a small margin.
func (s *Strategy) Lookback() int {
	m := s.p.ProfileWindow
	if s.p.ChandelierWindow > m {
		m = s.p.ChandelierWindow
	}
	if s.p.ATRPeriod+1 > m {
		m = s.p.ATRPeriod + 1
	}
	if s.p.BreakoutLookback+1 > m {
		m = s.p.BreakoutLookback + 1
	}
	if s.p.SwingLowWindow > m {
		m = s.p.SwingLowWindow
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure decision core.
type decideInput struct {
	price         float64
	atr           float64
	levels        []levels.Level
	barHigh       float64 // most-recent bar's high
	barLow        float64 // most-recent bar's low
	barClose      float64 // most-recent bar's close
	recentHigh    float64 // chandelier high over ChandelierWindow
	recentLow     float64 // structural low over SwingLowWindow
	downtrend     bool    // HTF soft downtrend flag (blocks entries only)
	recentlyBelow bool    // some close within BreakoutLookback was below the support level
	pos           *strategy.Position
}

// Decide computes every indicator from md, packs them, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)

	// Build the volume profile over the last ProfileWindow bars (all if fewer).
	hi := tailF(md.Highs, s.p.ProfileWindow)
	lo := tailF(md.Lows, s.p.ProfileWindow)
	vol := tailI(md.Volumes, s.p.ProfileWindow)
	vp := indicators.ComputeVolumeProfile(hi, lo, vol, s.p.ProfileBins)
	lvls := levels.ExtractLevels(vp, s.p.HVNFactor)

	var barHigh, barLow, barClose float64
	if n := len(md.Closes); n > 0 {
		barClose = md.Closes[n-1]
	}
	if n := len(md.Highs); n > 0 {
		barHigh = md.Highs[n-1]
	}
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	// HTF soft downtrend flag: only blocks. Uptrend and sideways both pass.
	downtrend := false
	if s.p.TrendFilterPeriod > 0 && len(md.DailyCloses) >= s.p.TrendFilterPeriod+1 {
		emaD := ema.Compute(md.DailyCloses, s.p.TrendFilterPeriod)
		last, prev := emaD[len(emaD)-1], emaD[len(emaD)-2]
		if prev > 0 {
			slope := (last - prev) / prev
			lastClose := md.DailyCloses[len(md.DailyCloses)-1]
			downtrend = slope < -s.p.TrendSlopeTol && lastClose < last
		}
	}

	// Classify a setup as a breakout-retest when price was recently below the
	// candidate support (a former resistance now broken). Resolve the support
	// here so recentlyBelow refers to the same level the core will act on.
	recentlyBelow := false
	if support, ok := levels.NearestSupportBelow(lvls, md.Price); ok {
		recentlyBelow = recentlyBelowLevel(md.Closes, support.Price, s.p.BreakoutLookback)
	}

	in := decideInput{
		price:         md.Price,
		atr:           atr,
		levels:        lvls,
		barHigh:       barHigh,
		barLow:        barLow,
		barClose:      barClose,
		recentHigh:    recentHigh(md.Highs, s.p.ChandelierWindow),
		recentLow:     recentLow(md.Lows, s.p.SwingLowWindow),
		downtrend:     downtrend,
		recentlyBelow: recentlyBelow,
		pos:           md.Position,
	}

	sig := s.decide(in)
	sig.Ticker = s.ticker
	return sig
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price}

	// Manage an open long position: hard stop then armed chandelier trail.
	if in.pos != nil {
		entry := in.pos.PurchasePrice
		hardSL := entry - s.p.SLMult*in.atr
		chandelier := in.recentHigh - s.p.TrailMult*in.atr
		// The trail only arms once the trade is in profit by TrailArmATR*ATR, so a
		// fresh entry near a recent high is not stopped out on the first down-tick.
		// TrailArmATR<=0 arms immediately.
		armed := s.p.TrailArmATR <= 0 || in.price >= entry+s.p.TrailArmATR*in.atr
		sig.StopLoss = hardSL
		switch {
		case in.barLow <= hardSL:
			sig.Kind, sig.Reason = model.SignalSell, "SL"
		case armed && in.barLow <= chandelier:
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
		return sig
	}

	// Flat -> long-only entry off a support level.
	if in.downtrend {
		return sig
	}
	// Anti-churn: skip when the instrument is barely moving.
	if s.p.MinATRFrac > 0 && in.atr < s.p.MinATRFrac*in.price {
		return sig
	}

	support, okS := levels.NearestSupportBelow(in.levels, in.price)
	resist, okR := levels.NearestResistanceAbove(in.levels, in.price)

	// Room gate: need a resistance above with enough runway to the target.
	if !okR || (resist.Price-in.price) < s.p.RoomATR*in.atr {
		return sig
	}
	if !okS {
		return sig
	}

	dist := in.price - support.Price
	notExtended := dist <= s.p.MaxExtensionATR*in.atr
	// Touch: the latest bar reached down to the level (within tolerance). Price may
	// sit a bit above the level after bouncing, so proximity is judged by the touch
	// and the not-extended gate, not by requiring price itself near the level.
	touched := in.barLow <= support.Price+s.p.LevelTolATR*in.atr
	// Bullish confirmation: closed in the upper part of the bar AND back above the level.
	bullish := bullishClose(in.barHigh, in.barLow, in.barClose, s.p.ConfirmCloseFrac) &&
		in.barClose > support.Price

	if touched && notExtended && bullish {
		stop := support.Price - s.p.SLMult*in.atr
		target := resist.Price
		if s.entryQualifies(in.price, stop, target, in.atr) {
			sig.Kind, sig.StopLoss, sig.TakeProfit = model.SignalBuy, stop, target
			sig.Level, sig.ATR = support.Price, in.atr
			if in.recentlyBelow {
				sig.Reason = "RETEST"
			} else {
				sig.Reason = "BOUNCE"
			}
		}
	}
	return sig
}

// entryQualifies applies the trade-quality gates: a minimum ATR (anti-churn) and a
// minimum reward:risk ratio. Either check is skipped when its param is <= 0.
func (s *Strategy) entryQualifies(price, stop, target, atr float64) bool {
	if s.p.MinATRFrac > 0 && atr < s.p.MinATRFrac*price {
		return false
	}
	if s.p.MinRR > 0 {
		risk := price - stop
		// A non-positive risk (stop >= price) is rejected by the risk<=0 guard.
		// A negative reward (target <= price) is implicitly rejected by the same
		// (target-price) < MinRR*risk comparison.
		if risk <= 0 || (target-price) < s.p.MinRR*risk {
			return false
		}
	}
	return true
}

// bullishClose reports whether the bar closed in its upper part. The bar's body is
// measured by (close-low)/(high-low); a zero-range bar is treated as bullish.
func bullishClose(high, low, close, frac float64) bool {
	rng := high - low
	if rng <= 0 {
		return true
	}
	return (close-low)/rng >= frac
}

// recentlyBelowLevel reports whether any of the last `lookback` completed closes
// (excluding the current bar) was strictly below level — i.e. price has just
// reclaimed a former resistance and is now retesting it from above.
func recentlyBelowLevel(closes []float64, level float64, lookback int) bool {
	n := len(closes)
	if n < 2 || lookback <= 0 {
		return false
	}
	end := n - 1 // exclude the current bar
	start := end - lookback
	if start < 0 {
		start = 0
	}
	for i := start; i < end; i++ {
		if closes[i] < level {
			return true
		}
	}
	return false
}

// recentHigh returns the highest high over the last `window` bars (all bars if fewer).
// A non-positive window is clamped to the last bar so it can never index out of range.
func recentHigh(highs []float64, window int) float64 {
	n := len(highs)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	h := highs[start]
	for i := start + 1; i < n; i++ {
		if highs[i] > h {
			h = highs[i]
		}
	}
	return h
}

// recentLow returns the lowest low over the last `window` bars (all bars if fewer).
// A non-positive window is clamped to the last bar so it can never index out of range.
func recentLow(lows []float64, window int) float64 {
	n := len(lows)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	l := lows[start]
	for i := start + 1; i < n; i++ {
		if lows[i] < l {
			l = lows[i]
		}
	}
	return l
}

// tailF returns the last `window` elements of xs (all if fewer or window<=0).
func tailF(xs []float64, window int) []float64 {
	if window <= 0 || len(xs) <= window {
		return xs
	}
	return xs[len(xs)-window:]
}

// tailI returns the last `window` elements of xs (all if fewer or window<=0).
func tailI(xs []int64, window int) []int64 {
	if window <= 0 || len(xs) <= window {
		return xs
	}
	return xs[len(xs)-window:]
}
