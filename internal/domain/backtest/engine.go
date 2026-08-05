package backtest

import (
	"fmt"
	"strings"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultHTFInterval is the HTF bar-span assumed when Config.HTFInterval is unset: the
// Hour4 series reversion was built on. Run/Trace pass Config.htfSpan() to
// visibleCompletedHTF at each bar to decide which HTF bars have fully closed.
const defaultHTFInterval = 4 * time.Hour

// mskLoc anchors the trading-day boundary used to decide which daily candles are
// already closed at a given intraday bar. Fallback to UTC if the tz DB is absent.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// startOfDay returns midnight of t's calendar day in loc.
func startOfDay(t time.Time, loc *time.Location) time.Time {
	tl := t.In(loc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, loc)
}

// visibleDaily returns the closes, highs, lows and open-times of daily candles whose
// calendar day (in loc) is strictly before t's calendar day — i.e. days that have fully
// closed by t. This is the no-lookahead rule: the current, still-forming day is never
// visible. The four series are index-aligned, oldest-first.
func visibleDaily(daily []Candle, t time.Time, loc *time.Location) (closes, highs, lows []float64, times []time.Time) {
	bound := startOfDay(t, loc)
	for _, c := range daily {
		if c.Time.Before(bound) {
			closes = append(closes, c.Close)
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
			times = append(times, c.Time)
		}
	}
	return closes, highs, lows, times
}

// visibleCompletedHTF returns closes/highs/lows of higher-timeframe candles that have
// FULLY closed by cur. A bar opening at c.Time spanning `interval` is closed once
// c.Time.Add(interval) <= cur; the current, still-forming HTF bar is never visible
// (no-lookahead). The three series are index-aligned, oldest-first, so the last element
// is the most recent HTF bar closed at/before cur.
func visibleCompletedHTF(htf []Candle, cur time.Time, interval time.Duration) (closes, highs, lows []float64) {
	for _, c := range htf {
		if !c.Time.Add(interval).After(cur) { // c.Time+interval <= cur
			closes = append(closes, c.Close)
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
		}
	}
	return closes, highs, lows
}

// htfCursor produces the same series visibleCompletedHTF would return, but amortized
// O(1) per bar instead of O(len(htf)) per bar: since Run/Trace advance cur
// monotonically, the completed-prefix boundary only ever moves forward, so each call
// resumes scanning where the previous one left off instead of rescanning from the
// start. The closes/highs/lows backing arrays are precomputed once; each call returns
// a prefix slice of them (no per-bar allocation). A nil *htfCursor is a valid
// zero-cost no-op for the no-HTF case.
//
// Precondition: visible must be called with a non-decreasing cur across the lifetime
// of a cursor, and htf must be sorted oldest-first (both hold for Run's and Trace's
// forward bar loops, and the cache writer sorts/dedups before this cursor is built). A
// rewound cur would NOT re-shrink idx, so visible would keep returning the wider prefix
// already confirmed — i.e. lookahead — instead of the narrower one visibleCompletedHTF
// would compute for that earlier cur.
type htfCursor struct {
	htf                 []Candle
	closes, highs, lows []float64
	interval            time.Duration
	idx                 int // number of htf bars confirmed visible so far (monotonic non-decreasing)
}

// newHTFCursor builds a cursor over htf (oldest-first). Returns nil when htf is empty
// so callers pay nothing for the no-HTF case.
func newHTFCursor(htf []Candle, interval time.Duration) *htfCursor {
	if len(htf) == 0 {
		return nil
	}
	closes := make([]float64, len(htf))
	highs := make([]float64, len(htf))
	lows := make([]float64, len(htf))
	for i, c := range htf {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
	}
	return &htfCursor{htf: htf, closes: closes, highs: highs, lows: lows, interval: interval}
}

// visible returns the same (closes, highs, lows) visibleCompletedHTF(htf, cur, interval)
// would, provided cur is non-decreasing across calls (Run/Trace's bar loop). Returns
// nil, nil, nil for a nil cursor or an empty prefix, matching visibleCompletedHTF.
func (h *htfCursor) visible(cur time.Time) (closes, highs, lows []float64) {
	if h == nil {
		return nil, nil, nil
	}
	for h.idx < len(h.htf) && !h.htf[h.idx].Time.Add(h.interval).After(cur) {
		h.idx++
	}
	if h.idx == 0 {
		return nil, nil, nil
	}
	// Full slice expressions cap the returned prefix at its own length, so a caller
	// appending to it (e.g. md.HTFCloses = append(md.HTFCloses, x)) forces a new backing
	// array instead of silently overwriting the shared closes/highs/lows arrays that
	// later bars still read from.
	return h.closes[:h.idx:h.idx], h.highs[:h.idx:h.idx], h.lows[:h.idx:h.idx]
}

// TodayExtent returns the high and low of the MSK calendar day that bar i belongs to,
// scanning back through the (oldest-first) window. Exported so a live runner fills
// MarketData.TodayHigh/TodayLow with the ENGINE's own rule instead of a lookalike:
// AssembleMarketData deliberately leaves those two fields to the caller.
func TodayExtent(candles []Candle, i int) (high, low float64) {
	return todayExtentIn(candles, i, mskLoc)
}

// todayExtentIn returns the high and low across all bars sharing candles[i]'s MSK
// calendar day, scanning back from i only (no lookahead). Returns (0,0) when i is
// out of range.
func todayExtentIn(candles []Candle, i int, loc *time.Location) (high, low float64) {
	if i < 0 || i >= len(candles) {
		return 0, 0
	}
	day := startOfDay(candles[i].Time, loc)
	high, low = candles[i].High, candles[i].Low
	for j := i - 1; j >= 0; j-- {
		if startOfDay(candles[j].Time, loc).Before(day) {
			break
		}
		if candles[j].High > high {
			high = candles[j].High
		}
		if candles[j].Low < low {
			low = candles[j].Low
		}
	}
	return high, low
}

// Run replays a strategy over oldest-first candles on a mock portfolio and
// returns the raw result. It mirrors the live runner: per bar it builds a
// lookback-sized MarketData, calls Decide, and acts only on Buy (when flat) or
// Sell (when in position), filling at the bar's close. An open position at the
// end is marked-to-market, never force-closed.
func Run(s strategy.Strategy, candles []Candle, dailyCandles, htfCandles []Candle, cfg Config) Result {
	res := Result{InitialCash: cfg.InitialCash, FinalEquity: cfg.InitialCash}
	l := s.Lookback()
	if l <= 0 || len(candles) < l {
		return res
	}
	p := newPortfolio(cfg)
	lastClose := candles[len(candles)-1].Close
	htf := newHTFCursor(htfCandles, cfg.htfSpan())
	for i := l - 1; i < len(candles); i++ {
		p.bar = i
		md := buildMarketData(candles[i-l+1 : i+1])
		md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes = visibleDaily(dailyCandles, candles[i].Time, mskLoc)
		md.HTFCloses, md.HTFHighs, md.HTFLows = htf.visible(candles[i].Time)
		md.TodayHigh, md.TodayLow = todayExtentIn(candles, i, mskLoc)
		if p.qty != 0 {
			p.mark(candles[i].Close)
		}
		md.Position = p.strategyPosition()

		c := candles[i]
		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss, sig.EntryReason)
			}
		case model.SignalSell:
			if p.qty != 0 {
				exitPrice := c.Close
				// Stop exits fill at the stop level, adjusted for a gap-down open:
				// min(level, open) lands inside the bar and charges real gap risk.
				// TP exits fill at the target, adjusted for a gap-up open: max(target,
				// open) is the limit fill and rewards a gap through the target.
				// The set of stop-style reasons is centralized in model.IsStopReason.
				switch {
				case model.IsStopReason(sig.Reason):
					if sig.StopLoss > 0 {
						exitPrice = min(sig.StopLoss, c.Open)
					}
				case sig.Reason == "TP":
					if sig.TakeProfit > 0 {
						exitPrice = max(sig.TakeProfit, c.Open)
					}
				}
				res.Trades = append(res.Trades, p.close(exitPrice, c.Time, sig.Reason, sig.ExitReason))
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

// explainer is the optional gate-level diagnostic a strategy may implement.
type explainer interface {
	Explain(md strategy.MarketData) string
}

// Trace replays the strategy exactly like Run but, at the bar whose timestamp
// equals target, captures the live position state and (if the strategy
// implements explainer) the gate-by-gate verdict. It returns a human-readable
// diagnostic. Used to answer "why did/didn't it act at this bar?". Replaying
// from the start means the reported position state is the real one.
func Trace(s strategy.Strategy, candles []Candle, dailyCandles, htfCandles []Candle, cfg Config, target time.Time) string {
	l := s.Lookback()
	if l <= 0 || len(candles) < l {
		return "недостаточно свечей для lookback"
	}
	p := newPortfolio(cfg)
	htf := newHTFCursor(htfCandles, cfg.htfSpan())
	for i := l - 1; i < len(candles); i++ {
		p.bar = i
		md := buildMarketData(candles[i-l+1 : i+1])
		md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes = visibleDaily(dailyCandles, candles[i].Time, mskLoc)
		md.HTFCloses, md.HTFHighs, md.HTFLows = htf.visible(candles[i].Time)
		md.TodayHigh, md.TodayLow = todayExtentIn(candles, i, mskLoc)
		if p.qty != 0 {
			p.mark(candles[i].Close)
		}
		md.Position = p.strategyPosition()

		c := candles[i]
		if c.Time.Equal(target) {
			var sb strings.Builder
			fmt.Fprintf(&sb, "Бар %s (MSK %s)\n", c.Time.Format(time.RFC3339), c.Time.In(mskLoc).Format("2006-01-02 15:04"))
			fmt.Fprintf(&sb, "OHLC: O=%.4f H=%.4f L=%.4f C=%.4f V=%d\n", c.Open, c.High, c.Low, c.Close, c.Volume)
			if p.qty != 0 {
				fmt.Fprintf(&sb, "Состояние: В ПОЗИЦИИ (qty=%d, вход %.4f)\n", p.qty, p.entryPrice)
			} else {
				sb.WriteString("Состояние: вне позиции (flat)\n")
			}
			sb.WriteString("--- фильтры входа ---\n")
			// Advance per-bar strategy state (e.g. the momentum cooldown counter) for
			// the target bar exactly as Run does, so Explain reads the same state the
			// real Decide path sees on this bar. The returned signal is discarded; it
			// is a no-op for stateless strategies.
			s.Decide(md)
			if ex, ok := s.(explainer); ok {
				sb.WriteString(ex.Explain(md))
			} else {
				sb.WriteString("стратегия не поддерживает Explain")
			}
			return sb.String()
		}

		sig := s.Decide(md)
		switch sig.Kind {
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss, sig.EntryReason)
			}
		case model.SignalSell:
			if p.qty != 0 {
				exitPrice := c.Close
				switch {
				case model.IsStopReason(sig.Reason):
					if sig.StopLoss > 0 {
						exitPrice = min(sig.StopLoss, c.Open)
					}
				case sig.Reason == "TP":
					if sig.TakeProfit > 0 {
						exitPrice = max(sig.TakeProfit, c.Open)
					}
				}
				p.close(exitPrice, c.Time, sig.Reason, sig.ExitReason)
			}
		}
	}
	return fmt.Sprintf("бар с временем %s не найден в свечах", target.Format(time.RFC3339))
}

// buildMarketData converts an oldest-first window into a strategy snapshot,
// mirroring scalping/trade.go's buildMarketData.
func buildMarketData(window []Candle) strategy.MarketData {
	md := strategy.MarketData{
		Highs:   make([]float64, len(window)),
		Lows:    make([]float64, len(window)),
		Closes:  make([]float64, len(window)),
		Volumes: make([]int64, len(window)),
		Times:   make([]time.Time, len(window)),
	}
	for i, c := range window {
		md.Highs[i] = c.High
		md.Lows[i] = c.Low
		md.Closes[i] = c.Close
		md.Volumes[i] = c.Volume
		md.Times[i] = c.Time
	}
	if n := len(window); n > 0 {
		md.Price = window[n-1].Close
	}
	return md
}

// AssembleMarketData builds the per-bar snapshot with the default 4H higher-timeframe
// span. Kept for callers built on the Hour4 series (reversion, including its live
// market-data assembler).
func AssembleMarketData(window, daily, htf []Candle, cur time.Time) strategy.MarketData {
	return AssembleMarketDataWithHTFInterval(window, daily, htf, cur, defaultHTFInterval)
}

// AssembleMarketDataWithHTFInterval builds the per-bar snapshot from an oldest-first
// window plus completed-daily and higher-timeframe series, identically to Run's per-bar
// assembly — minus TodayHigh/TodayLow, which the caller sets separately. cur is the
// open-time of the current (latest) bar; it anchors the no-lookahead completeness test
// for the daily and HTF series. htfSpan is the bar-span of the htf series (e.g. 1h for
// scalping_rsimacd, 4h for reversion).
func AssembleMarketDataWithHTFInterval(window, daily, htf []Candle, cur time.Time, htfSpan time.Duration) strategy.MarketData {
	md := buildMarketData(window)
	md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes = visibleDaily(daily, cur, mskLoc)
	md.HTFCloses, md.HTFHighs, md.HTFLows = visibleCompletedHTF(htf, cur, htfSpan)
	return md
}
