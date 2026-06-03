package rusal

import (
	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const ticker = "RUAL"

// Params holds every tunable for the RUSAL adaptive strategy. They are exposed so
// they can be calibrated on real history without touching the decision logic.
type Params struct {
	EMAPeriod        int     // fast EMA for the trend pullback
	ADXPeriod        int     // ADX/DMI period
	ADXTrendLevel    float64 // ADX >= -> trend regime
	ADXRangeLevel    float64 // ADX <= -> range regime (between the two = dead zone)
	RSIPeriod        int     // RSI period
	RSITrendLevel    float64 // RSI reversal threshold in trend (shallow pullbacks)
	RSIRangeLevel    float64 // RSI reversal threshold in range (oversold)
	PullbackWindow   int     // bars back over which an EMA "touch" still counts
	DonchianPeriod   int     // channel period: lower for entry, mid for range exit
	ATRPeriod        int     // ATR period for stops/trailing
	SLMult           float64 // initial stop = entry - SLMult*ATR
	TrailMult        float64 // chandelier = max(High over window) - TrailMult*ATR
	ChandelierWindow int     // window for the chandelier high
	EMATouchTol      float64 // EMA touch tolerance (fraction, e.g. 0.002 = 0.2%)
	BandTol          float64 // lower-band proximity tolerance (fraction)
}

// DefaultParams returns standard, NOT-yet-calibrated starting values.
func DefaultParams() Params {
	return Params{
		EMAPeriod:        21,
		ADXPeriod:        14,
		ADXTrendLevel:    25,
		ADXRangeLevel:    20,
		RSIPeriod:        14,
		RSITrendLevel:    45,
		RSIRangeLevel:    35,
		PullbackWindow:   5,
		DonchianPeriod:   20,
		ATRPeriod:        14,
		SLMult:           1.0,
		TrailMult:        2.5,
		ChandelierWindow: 20,
		EMATouchTol:      0.002,
		BandTol:          0.003,
	}
}

// Strategy trades RUSAL adaptively: it picks a regime from ADX and applies
// mean-reversion in a range or momentum in a trend.
type Strategy struct{ p Params }

// New returns the RUSAL strategy with default (uncalibrated) params.
func New() *Strategy { return &Strategy{p: DefaultParams()} }

// NewWithParams returns the RUSAL strategy with explicit params (for calibration/tests).
func NewWithParams(p Params) *Strategy { return &Strategy{p: p} }

func (s *Strategy) Ticker() string { return ticker }

// Lookback sizes the candle window for ADX's double smoothing (the hungriest indicator).
func (s *Strategy) Lookback() int { return 6*s.p.ADXPeriod + s.p.DonchianPeriod + 50 }

// regime classifies the market from ADX.
type regime int

const (
	regimeDead regime = iota
	regimeTrend
	regimeRange
)

func (s *Strategy) regimeOf(adx float64) regime {
	switch {
	case adx <= 0: // ADX returns 0 on insufficient/invalid history — treat as no-signal
		return regimeDead
	case adx >= s.p.ADXTrendLevel:
		return regimeTrend
	case adx <= s.p.ADXRangeLevel:
		return regimeRange
	default:
		return regimeDead
	}
}

// decideInput carries already-computed indicator values into the pure decision core.
type decideInput struct {
	price          float64
	atr            float64
	emaNow         float64
	rsiPrev        float64
	rsiNow         float64
	adx            float64
	diPlus         float64
	diMinus        float64
	donUpper       float64
	donLower       float64
	emaTouched     bool
	chandelierHigh float64
	pos            *strategy.Position
}

// Decide computes every indicator from md, packs them, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	closes := md.Closes
	n := len(closes)

	emaSeries := ema.Compute(closes, s.p.EMAPeriod)
	rsiSeries := indicators.RSISeries(closes, s.p.RSIPeriod)
	atr := indicators.ATR(md.Highs, md.Lows, closes, s.p.ATRPeriod)
	adx, diPlus, diMinus := indicators.ADX(md.Highs, md.Lows, closes, s.p.ADXPeriod)
	donUpper, donLower := indicators.Donchian(md.Highs, md.Lows, s.p.DonchianPeriod)

	var emaNow, rsiPrev, rsiNow float64
	if n > 0 {
		emaNow = emaSeries[n-1]
	}
	if n >= 2 {
		rsiNow = rsiSeries[n-1]
		rsiPrev = rsiSeries[n-2]
	}

	in := decideInput{
		price:          md.Price,
		atr:            atr,
		emaNow:         emaNow,
		rsiPrev:        rsiPrev,
		rsiNow:         rsiNow,
		adx:            adx,
		diPlus:         diPlus,
		diMinus:        diMinus,
		donUpper:       donUpper,
		donLower:       donLower,
		emaTouched:     emaTouched(md.Lows, emaSeries, s.p.PullbackWindow, s.p.EMATouchTol),
		chandelierHigh: recentHigh(md.Highs, s.p.ChandelierWindow),
		pos:            md.Position,
	}

	sig := s.decide(in)
	sig.Ticker = ticker
	return sig
}

// decide is the pure decision core. STUB: filled in Task 4.
func (s *Strategy) decide(in decideInput) model.Signal {
	return model.Signal{Price: in.price, RSI: in.rsiNow}
}

// emaTouched reports whether a low dipped to the EMA (within tol) on any of the last
// `window` bars — the pullback condition for a trend entry.
func emaTouched(lows, ema []float64, window int, tol float64) bool {
	n := len(lows)
	if n == 0 || len(ema) != n {
		return false
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	for i := start; i < n; i++ {
		if ema[i] > 0 && lows[i] <= ema[i]*(1+tol) {
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
