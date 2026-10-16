package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// barsPerHour is the number of 5-minute bars in an hour.
const barsPerHour = 12

// hourRSIOn reports whether half-hour signals use the hourly RSI gate instead of RSIEntryMax.
func (s *Strategy) hourRSIOn() bool { return s.p.HourRSIPeriod > 0 && s.p.HourRSIMax > 0 }

// hourCloses is the close of every MSK hour (:00 boundaries; MSK is UTC+3 in whole hours) built
// from md's 5m bars that started before end. The last hour holds only those bars, so it may be
// still forming.
func hourCloses(md strategy.MarketData, end time.Time) []float64 {
	var closes []float64
	var last time.Time
	for j, t := range md.Times {
		if !t.Before(end) {
			break
		}
		h := t.Truncate(time.Hour)
		if len(closes) == 0 || !h.Equal(last) {
			closes = append(closes, md.Closes[j])
			last = h
			continue
		}
		closes[len(closes)-1] = md.Closes[j]
	}
	return closes
}

// signalRSI is the RSI the reversal gate reads on bar r of ser (the bar before the reversal), its
// label and whether the gate passes. A half-hour signal with the hourly gate on reads
// RSI(HourRSIPeriod) of the hour up to half-hour r's close and needs it strictly below HourRSIMax;
// too few hours fail. Any other signal reads entryRSI on its own timeframe.
func (s *Strategy) signalRSI(md strategy.MarketData, ser series, r int, tf string) (float64, string, bool) {
	if tf != "30m" || !s.hourRSIOn() {
		rsi, ok := s.entryRSI(ser.c[:r+1])
		return rsi, fmt.Sprintf("RSI(%d) %s", s.p.RSIPeriod, tf), ok
	}
	label := fmt.Sprintf("RSI(%d) 1h", s.p.HourRSIPeriod)
	closes := hourCloses(md, ser.t[r].Add(halfHour))
	n := len(closes)
	if n <= s.p.HourRSIPeriod {
		return 0, label, false
	}
	rsi := indicators.RSISeries(closes, s.p.HourRSIPeriod)[n-1]
	return rsi, label, rsi < s.p.HourRSIMax
}
