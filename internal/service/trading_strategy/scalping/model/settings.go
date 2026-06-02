package model

import "tinvest/internal/enum"

// Settings holds the tunable algorithm knobs for the scalping strategy.
type Settings struct {
	EmaPeriod         int           // trend filter EMA period
	RsiPeriod         int32         // RSI period
	RsiReversalLevel  float64       // RSI level that must be crossed upward to enter
	AtrTakeProfitMult float64       // take-profit = entry + mult*ATR
	AtrStopLossMult   float64       // stop-loss   = entry - mult*ATR
	UniverseSize      int           // number of most-volatile shares to scan
	MaxOpenPositions  int           // cap on simultaneously open positions
	Interval          enum.Interval // timeframe
}

// DefaultSettings returns the conservative hourly defaults.
func DefaultSettings() Settings {
	return Settings{
		EmaPeriod:         200,
		RsiPeriod:         14,
		RsiReversalLevel:  35,
		AtrTakeProfitMult: 1.5,
		AtrStopLossMult:   1.0,
		UniverseSize:      10,
		MaxOpenPositions:  5,
		Interval:          enum.Hour1,
	}
}
