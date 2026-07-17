// Package core implements the SMC liquidity-sweep long-only swing strategy:
// a fractal swing-low is confirmed SwingK bars after its extreme; a bar
// piercing that level with its low and a close back above it within
// ReclaimBars is a stop-hunt — the reclaim bar buys at its close with a hard
// stop under the sweep extreme; exits are the intrabar stop, an R-multiple
// take-profit and a trading-day time-stop. Optional OB/FVG/discount filters
// are int toggles (0/1) so the calibration grid can sweep them. See
// docs/superpowers/specs/2026-07-17-smc-liquidity-sweep-design.md.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Fixed knobs — deliberately NOT part of Params: sweeping window/warm-up
// mechanics is how past strategies overfit.
const (
	atrPeriod        = 14 // stop unit
	levelWindowDays  = 10 // sliding window of distinct MSK days where levels live
	sessionOpenHour  = 10 // MSK: bars opening before 10:00 never enter
	eveningStartHour = 19 // MSK: bars opening at/after 19:00 (evening session) never enter
	// barsPerDayMax is the worst-case Hour1 bar count of one MSK day
	// (morning + main + evening sessions). Lookback must cover the whole
	// level window plus indicator warm-up.
	barsPerDayMax = 17
)

// Params are the SMC tunables. UseOB/UseFVG/UseDiscount are int toggles
// (grid values are numeric): 0 = off, 1 = on.
type Params struct {
	SwingK      int     // fractal wing: a swing-low needs SwingK strictly higher lows on each side
	ReclaimBars int     // max bars from pierce to reclaim close (0 = same-bar sweep only)
	Buffer      float64 // hard stop = sweepLow - Buffer*ATR(atrPeriod) at entry
	TPR         float64 // take-profit = entry + TPR*(entry - stop); <=0 disables
	MaxHoldDays int     // time-stop after this many distinct trading days; <=0 disables
	UseOB       int     // 1 = reclaim close must sit inside an unmitigated bullish order block
	UseFVG      int     // 1 = a bullish FVG must form between pierce and reclaim
	UseDiscount int     // 1 = entry close must be below the level-window range midpoint
}

// Strategy is the SMC rule bound to one ticker.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams builds the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

// Ticker returns the bound instrument ticker.
func (s *Strategy) Ticker() string { return s.ticker }

// Lookback returns the candle window the strategy needs: the full level
// window plus ATR warm-up.
func (s *Strategy) Lookback() int { return levelWindowDays*barsPerDayMax + atrPeriod + 2 }

// mskLoc anchors all session logic to the Moscow trading calendar (UTC
// fallback if the tz DB is absent), mirroring the backtest engine.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether t falls on Saturday or Sunday in mskLoc.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// sameMSKDay reports whether a and b share an MSK calendar day.
func sameMSKDay(a, b time.Time) bool {
	al, bl := a.In(mskLoc), b.In(mskLoc)
	return al.Year() == bl.Year() && al.YearDay() == bl.YearDay()
}

// startOfMSKDay returns midnight of t's MSK calendar day.
func startOfMSKDay(t time.Time) time.Time {
	tl := t.In(mskLoc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, mskLoc)
}

// windowStart returns the index of the oldest bar belonging to the last
// levelWindowDays distinct MSK days of times. Levels whose swing bar is
// older are forgotten.
func windowStart(times []time.Time) int {
	days := 0
	for i := len(times) - 1; i >= 0; i-- {
		if i == len(times)-1 || !sameMSKDay(times[i], times[i+1]) {
			days++
			if days > levelWindowDays {
				return i + 1
			}
		}
	}
	return 0
}

// tradingDaysSince counts distinct MSK days among times strictly after
// entry's day. Only days visible in the window count — safe while
// MaxHoldDays*barsPerDayMax stays well inside Lookback.
func tradingDaysSince(times []time.Time, entry time.Time) int {
	entryDay := startOfMSKDay(entry)
	days := 0
	var lastDay time.Time
	for _, t := range times {
		d := startOfMSKDay(t)
		if !d.After(entryDay) || d.Equal(lastDay) {
			continue
		}
		days++
		lastDay = d
	}
	return days
}

// level is one confirmed swing-low and its sweep lifecycle inside the window.
type level struct {
	price      float64 // the swing-low value — the liquidity line
	barIdx     int     // bar of the swing-low extreme
	confirmIdx int     // barIdx + SwingK: first bar the level is visible on (anti-lookahead)
	pierceIdx  int     // first bar > confirmIdx with Low < price; -1 while untouched
	reclaimIdx int     // first bar >= pierceIdx with Close > price; -1 while under water
	sweepLow   float64 // lowest Low from pierce through reclaim (or through the window end)
}

// levelStates finds confirmed fractal swing-lows inside the level window and
// classifies each one's sweep lifecycle. Bars within SwingK of the extreme
// cannot pierce it by construction (their lows are strictly higher), so the
// pierce scan starts after confirmation. A level is consumed by its FIRST
// reclaim; later sweeps of the same line never signal again.
func levelStates(lows, closes []float64, times []time.Time, k int) []level {
	n := len(lows)
	start := windowStart(times)
	var out []level
	for i := max(k, start); i+k < n; i++ {
		swing := true
		for d := 1; d <= k; d++ {
			if lows[i] >= lows[i-d] || lows[i] >= lows[i+d] {
				swing = false
				break
			}
		}
		if !swing {
			continue
		}
		lv := level{price: lows[i], barIdx: i, confirmIdx: i + k, pierceIdx: -1, reclaimIdx: -1}
		for j := lv.confirmIdx + 1; j < n; j++ {
			if lows[j] < lv.price {
				lv.pierceIdx = j
				break
			}
		}
		if lv.pierceIdx >= 0 {
			lv.sweepLow = lows[lv.pierceIdx]
			for j := lv.pierceIdx; j < n; j++ {
				lv.sweepLow = min(lv.sweepLow, lows[j])
				if closes[j] > lv.price {
					lv.reclaimIdx = j
					break
				}
			}
		}
		out = append(out, lv)
	}
	return out
}

// reclaimCandidate returns the level whose FIRST reclaim lands exactly on bar
// cur within maxBars of its pierce. When several levels reclaim on the same
// bar the deepest sweepLow wins — its stop covers the others.
func reclaimCandidate(levels []level, cur, maxBars int) (level, bool) {
	var best level
	found := false
	for _, lv := range levels {
		if lv.pierceIdx < 0 || lv.reclaimIdx != cur || lv.reclaimIdx-lv.pierceIdx > maxBars {
			continue
		}
		if !found || lv.sweepLow < best.sweepLow {
			best = lv
			found = true
		}
	}
	return best, found
}

// Decide is pure: it computes everything from md and performs no I/O. Times
// are mandatory; when Times is missing or misaligned the strategy is a
// deliberate no-op (can't see the calendar — don't trade).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Kind: model.SignalNone, Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	sig, _ = s.entryCheck(md, sig)
	return sig
}

// manage handles an open position: the intrabar hard stop first (it fires
// earlier in real time), then the take-profit, then the trading-day
// time-stop (close-fill).
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	pos := md.Position
	if pos.StopLoss > 0 && md.Lows[n-1] <= pos.StopLoss {
		sig.Kind = model.SignalSell
		sig.Reason = "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("интрабарный стоп %.4f (low бара %.4f)", pos.StopLoss, md.Lows[n-1])
		return sig
	}
	if tp, ok := takeProfit(pos, s.p.TPR); ok && md.Highs[n-1] >= tp {
		sig.Kind = model.SignalSell
		sig.Reason = "TP"
		sig.TakeProfit = tp
		sig.ExitReason = fmt.Sprintf("тейк-профит %.4f (high бара %.4f)", tp, md.Highs[n-1])
		return sig
	}
	if s.p.MaxHoldDays > 0 && !pos.EntryTime.IsZero() &&
		tradingDaysSince(md.Times, pos.EntryTime) >= s.p.MaxHoldDays {
		sig.Kind = model.SignalSell
		sig.Reason = "TIME"
		sig.ExitReason = fmt.Sprintf("тайм-стоп: позиция старше %d торговых дней", s.p.MaxHoldDays)
		return sig
	}
	return sig
}

// takeProfit derives the frozen TP from the position itself (stateless): the
// entry stop distance times TPR. ok=false when TPR is off or the stop is
// missing/inverted — then no TP exit exists.
func takeProfit(pos *strategy.Position, tpr float64) (float64, bool) {
	if tpr <= 0 || pos.StopLoss <= 0 || pos.PurchasePrice <= pos.StopLoss {
		return 0, false
	}
	return pos.PurchasePrice + tpr*(pos.PurchasePrice-pos.StopLoss), true
}

// entryCheck runs the full entry pipeline. On rejection the second result
// holds a human-readable reason (consumed by Explain). On success
// sig.Kind == model.SignalBuy with the frozen structural stop in sig.StopLoss.
func (s *Strategy) entryCheck(md strategy.MarketData, sig model.Signal) (model.Signal, string) {
	n := len(md.Closes)
	t := md.Times[n-1].In(mskLoc)
	if isWeekend(t) {
		return sig, "выходной (Сб/Вс MSK) — входы запрещены"
	}
	if h := t.Hour(); h < sessionOpenHour {
		return sig, fmt.Sprintf("час бара %d < %d MSK — утренняя сессия без входов", h, sessionOpenHour)
	} else if h >= eveningStartHour {
		return sig, fmt.Sprintf("час бара %d ≥ %d MSK — вечерняя сессия без входов", h, eveningStartHour)
	}
	if s.p.SwingK <= 0 {
		return sig, "SwingK ≤ 0 — вход выключен"
	}
	levels := levelStates(md.Lows, md.Closes, md.Times, s.p.SwingK)
	cand, ok := reclaimCandidate(levels, n-1, s.p.ReclaimBars)
	if !ok {
		return sig, "текущий бар не reclaim-ит ни один уровень в окне ReclaimBars"
	}
	if why, ok := s.passFilters(md, cand); !ok { // no-op до Task 6
		return sig, why
	}
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, atrPeriod)
	if atr <= 0 {
		return sig, "ATR не прогрет — не с чего считать стоп"
	}
	stop := cand.sweepLow - s.p.Buffer*atr
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.Level = cand.price
	if s.p.TPR > 0 {
		sig.TakeProfit = md.Price + s.p.TPR*(md.Price-stop)
	}
	sig.EntryReason = fmt.Sprintf(
		"sweep-и-reclaim уровня %.4f: прокол до %.4f (%d бар(ов) от прокола), close %.4f выше уровня; стоп %.4f",
		cand.price, cand.sweepLow, cand.reclaimIdx-cand.pierceIdx, md.Price, stop)
	return sig, ""
}

// passFilters applies the optional SMC filters to a reclaim candidate.
// Implemented in the filters task; the core entry is filter-free.
func (s *Strategy) passFilters(_ strategy.MarketData, _ level) (string, bool) {
	return "", true
}
