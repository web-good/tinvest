// Package core implements a long-only hourly trend-momentum strategy: enter when
// price is above a long EMA (uptrend), MACD just crossed up (optionally below
// zero), volume is above its recent average, and the day still has room left
// within its typical daily-ATR range. Exits on a frozen structural ATR stop, a
// fixed reward-multiple take-profit, or an optional chandelier trail. The decision
// logic is pure and ticker-agnostic; per-share packages supply ticker + Params.
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const (
	// dailyTrendSlopeBars is the fixed horizon (in completed daily candles) over which
	// the daily-EMA slope filter measures direction. Held as an in-package policy
	// constant (not a grid knob) to keep calibration combinatorics small.
	dailyTrendSlopeBars = 3
	// cooldownSaturate seeds and caps barsSinceExit so it never overflows and starts
	// "cooldown satisfied" (no entry blocked before the first exit).
	cooldownSaturate = 1 << 30
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1)
// so reflection grid calibration can sweep them.
type Params struct {
	EMAPeriod         int     // long EMA trend filter (hourly)
	MACDFast          int     // MACD fast EMA period
	MACDSlow          int     // MACD slow EMA period
	MACDSignal        int     // MACD signal EMA period
	MACDBelowZeroOnly int     // 1 = require macd<0 at the cross; 0 = any bullish cross
	VolLookback       int     // SMA window for the volume baseline
	VolMultiplier     float64 // last volume must exceed VolMultiplier*SMA(volume)
	DailyATRPeriod    int     // ATR period over completed daily candles
	MaxDailyATRUsed   float64 // block entry if today's range >= MaxDailyATRUsed*dailyATR
	ATRPeriod         int     // hourly ATR period (stops, anti-churn)
	SwingLowWindow    int     // bars scanned for the structural low anchoring the stop
	SLMult            float64 // stop = swingLow - SLMult*ATR
	TakeProfitRR      float64 // TP = entry + TakeProfitRR*(entry-stop)
	MinRR             float64 // reject entry if (TP-price) < MinRR*risk; <=0 disables
	MinATRFrac        float64 // reject entry if ATR < MinATRFrac*price; <=0 disables
	UseTrail          int     // 1 = trail instead of fixed TP
	TrailMult         float64 // chandelier = recentHigh(ChandelierWindow) - TrailMult*ATR
	ChandelierWindow  int     // window for the chandelier high
	TrailArmATR       float64 // trail arms after MaxFavorable >= entry + TrailArmATR*EntryATR
	CooldownBars      int     // bars to block re-entry after a position exits; 0 disables
	DailyTrendPeriod  int     // daily-EMA period for the higher-timeframe slope filter (0 disables)
}

// Strategy trades a single instrument with the momentum rules. Ticker-agnostic.
// It carries the cooldown counter as mutable state in the impure shell; the pure
// decide() core stays a function of its input. Not safe for concurrent use; the
// backtest and live runners drive Decide sequentially, one bar at a time.
type Strategy struct {
	ticker         string
	p              Params
	barsSinceExit  int  // bars elapsed since the last exit; gates re-entry
	prevInPosition bool // whether the previous Decide saw an open position
}

// NewWithParams returns the momentum strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p, barsSinceExit: cooldownSaturate}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer.
func (s *Strategy) Lookback() int {
	m := s.p.EMAPeriod
	for _, c := range []int{
		s.p.MACDSlow + s.p.MACDSignal,
		s.p.VolLookback + 1,
		s.p.ATRPeriod + 1,
		s.p.SwingLowWindow,
		s.p.ChandelierWindow,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}

// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price           float64
	atr             float64
	dailyATR        float64
	emaTrend        float64
	macdNow         float64
	crossUp         bool
	volumeOK        bool
	todayRange      float64
	barHigh         float64
	barLow          float64
	recentLow       float64
	recentHigh      float64
	dailyEMANow     float64 // last daily-EMA value (0 if unavailable)
	dailyEMAPast    float64 // daily-EMA value dailyTrendSlopeBars back (0 if unavailable)
	dailyTrendKnown bool    // true when daily history sufficed to compute both points
	barsSinceExit   int     // bars since the last exit, for the cooldown gate
	pos             *strategy.Position
}

// Decide computes every indicator from md, packs them, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	s.trackCooldown(md.Position)
	sig := s.decide(s.buildInput(md))
	sig.Ticker = s.ticker
	return sig
}

// trackCooldown advances the post-exit bar counter. It detects the in-position ->
// flat edge to reset the counter, then increments while flat. Called once per bar
// from Decide (the trading path); Explain never mutates this state.
func (s *Strategy) trackCooldown(pos *strategy.Position) {
	switch {
	case pos != nil:
		// In a position: cooldown is irrelevant, leave the counter as-is.
	case s.prevInPosition:
		s.barsSinceExit = 0 // just exited this bar
	case s.barsSinceExit < cooldownSaturate:
		s.barsSinceExit++
	}
	s.prevInPosition = pos != nil
}

// buildInput computes every indicator from md and packs them for the pure core.
func (s *Strategy) buildInput(md strategy.MarketData) decideInput {
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	dailyATR := indicators.ATR(md.DailyHighs, md.DailyLows, md.DailyCloses, s.p.DailyATRPeriod)

	emaTrend := 0.0
	if e := ema.Compute(md.Closes, s.p.EMAPeriod); len(e) > 0 {
		emaTrend = e[len(e)-1]
	}

	macdNow, crossUp := 0.0, false
	if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); len(m) >= 2 {
		prevDiff := m[len(m)-2] - sg[len(sg)-2]
		currDiff := m[len(m)-1] - sg[len(sg)-1]
		macdNow = m[len(m)-1]
		crossUp = prevDiff <= 0 && currDiff > 0
	}

	var barHigh, barLow float64
	if n := len(md.Highs); n > 0 {
		barHigh = md.Highs[n-1]
	}
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	dailyEMANow, dailyEMAPast, dailyTrendKnown := 0.0, 0.0, false
	if s.p.DailyTrendPeriod > 0 {
		if de := ema.Compute(md.DailyCloses, s.p.DailyTrendPeriod); len(de) >= s.p.DailyTrendPeriod+dailyTrendSlopeBars {
			n := len(de)
			dailyEMANow, dailyEMAPast, dailyTrendKnown = de[n-1], de[n-1-dailyTrendSlopeBars], true
		}
	}

	return decideInput{
		price:           md.Price,
		atr:             atr,
		dailyATR:        dailyATR,
		emaTrend:        emaTrend,
		macdNow:         macdNow,
		crossUp:         crossUp,
		volumeOK:        indicators.VolumeConfirmed(md.Volumes, s.p.VolLookback, s.p.VolMultiplier),
		todayRange:      md.TodayHigh - md.TodayLow,
		barHigh:         barHigh,
		barLow:          barLow,
		recentLow:       recentLow(md.Lows, s.p.SwingLowWindow),
		recentHigh:      recentHigh(md.Highs, s.p.ChandelierWindow),
		dailyEMANow:     dailyEMANow,
		dailyEMAPast:    dailyEMAPast,
		dailyTrendKnown: dailyTrendKnown,
		barsSinceExit:   s.barsSinceExit,
		pos:             md.Position,
	}
}

// Explain re-runs the entry gates over md and reports each gate's value and
// verdict (✓ pass / ✗ block) in entry order, stopping the chain at the first
// blocker — the same short-circuit order decide uses. Diagnostic only; never
// part of the trading path.
func (s *Strategy) Explain(md strategy.MarketData) string {
	in := s.buildInput(md)

	if in.pos != nil {
		return "позиция уже открыта — вход не рассматривается"
	}

	var b strings.Builder
	pass := func(format string, args ...any) { fmt.Fprintf(&b, "✓ "+format+"\n", args...) }
	block := func(format string, args ...any) string {
		fmt.Fprintf(&b, "✗ "+format+"\n", args...)
		fmt.Fprintf(&b, "→ ВХОДА НЕТ: заблокировал этот фильтр")
		return b.String()
	}

	// 0. Cooldown.
	if s.p.CooldownBars > 0 {
		if in.barsSinceExit < s.p.CooldownBars {
			return block("Кулдаун: после выхода прошло %d бар(ов) из %d", in.barsSinceExit, s.p.CooldownBars)
		}
		pass("Кулдаун: после выхода прошло %d бар(ов) ≥ %d", in.barsSinceExit, s.p.CooldownBars)
	}

	// 1. Uptrend.
	if !(in.emaTrend > 0 && in.price > in.emaTrend) {
		return block("Тренд: close %.4f ≤ EMA%d %.4f (нужно выше)", in.price, s.p.EMAPeriod, in.emaTrend)
	}
	pass("Тренд: close %.4f > EMA%d %.4f", in.price, s.p.EMAPeriod, in.emaTrend)

	// 1b. Daily trend slope.
	if s.p.DailyTrendPeriod > 0 {
		switch {
		case !in.dailyTrendKnown:
			pass("Дневной тренд: недостаточно дневной истории — фильтр пропущен")
		case !(in.dailyEMANow > in.dailyEMAPast):
			return block("Дневной тренд: EMA%d не растёт (%.4f ≤ %.4f, %d дн назад)",
				s.p.DailyTrendPeriod, in.dailyEMANow, in.dailyEMAPast, dailyTrendSlopeBars)
		default:
			pass("Дневной тренд: EMA%d растёт (%.4f > %.4f)", s.p.DailyTrendPeriod, in.dailyEMANow, in.dailyEMAPast)
		}
	}

	// 2. MACD bullish cross.
	if !in.crossUp {
		return block("MACD: нет бычьего кросса на этом баре (MACD=%.4f)", in.macdNow)
	}
	pass("MACD: бычий кросс (MACD=%.4f)", in.macdNow)

	// 3. Below-zero requirement.
	if s.p.MACDBelowZeroOnly == 1 && in.macdNow >= 0 {
		return block("MACD: кросс над нулём (%.4f), а требуется под нулём (MACDBelowZeroOnly=1)", in.macdNow)
	}
	if s.p.MACDBelowZeroOnly == 1 {
		pass("MACD: кросс под нулём (%.4f)", in.macdNow)
	}

	// 4. Volume.
	if !in.volumeOK {
		return block("Объём: ниже %.2g×ср(%d) — подтверждения нет", s.p.VolMultiplier, s.p.VolLookback)
	}
	pass("Объём: выше %.2g×ср(%d)", s.p.VolMultiplier, s.p.VolLookback)

	// 5. Daily-ATR room.
	if in.dailyATR > 0 && in.todayRange >= s.p.MaxDailyATRUsed*in.dailyATR {
		roomPct := (1 - in.todayRange/in.dailyATR) * 100
		return block("Запас дневного ATR: день уже прошёл %.4f из %.4f (осталось %.0f%%, лимит входа %.0f%%)",
			in.todayRange, in.dailyATR, roomPct, (1-s.p.MaxDailyATRUsed)*100)
	}
	if in.dailyATR > 0 {
		roomPct := (1 - in.todayRange/in.dailyATR) * 100
		pass("Запас дневного ATR: прошло %.4f из %.4f (осталось %.0f%%)", in.todayRange, in.dailyATR, roomPct)
	} else {
		pass("Запас дневного ATR: дневных данных нет — фильтр пропущен")
	}

	// 6. Anti-churn.
	if s.p.MinATRFrac > 0 && in.atr < s.p.MinATRFrac*in.price {
		return block("Анти-черн: ATR(ч) %.4f < %.4g×цена %.4f", in.atr, s.p.MinATRFrac, in.price)
	}
	pass("Анти-черн: ATR(ч) %.4f ≥ порога", in.atr)

	// 7. Risk / RR sanity.
	stop := in.recentLow - s.p.SLMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return block("Риск: SL=%.4f ≥ цены %.4f (risk ≤ 0)", stop, in.price)
	}
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return block("RR: цель %.4f даёт %.2gR < MinRR %.2g", target, (target-in.price)/risk, s.p.MinRR)
	}
	pass("Риск: SL=%.4f, TP=%.4f, %.2gR", stop, target, (target-in.price)/risk)

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price}

	if in.pos != nil {
		return s.manage(in, sig)
	}

	if s.p.CooldownBars > 0 && in.barsSinceExit < s.p.CooldownBars {
		return sig // still cooling down after the last exit
	}

	// Entry gates (all must pass).
	if !(in.emaTrend > 0 && in.price > in.emaTrend) {
		return sig // not an uptrend
	}
	if s.p.DailyTrendPeriod > 0 && in.dailyTrendKnown && !(in.dailyEMANow > in.dailyEMAPast) {
		return sig // daily trend not rising
	}
	if !in.crossUp {
		return sig
	}
	if s.p.MACDBelowZeroOnly == 1 && in.macdNow >= 0 {
		return sig
	}
	if !in.volumeOK {
		return sig
	}
	// Daily-ATR room: pass when daily data is absent (dailyATR<=0), else require room.
	if in.dailyATR > 0 && in.todayRange >= s.p.MaxDailyATRUsed*in.dailyATR {
		return sig
	}
	if s.p.MinATRFrac > 0 && in.atr < s.p.MinATRFrac*in.price {
		return sig
	}

	stop := in.recentLow - s.p.SLMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return sig
	}
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return sig
	}

	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = in.atr
	sig.EntryReason = s.entryReason(in, stop, target, risk)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, target, risk float64) string {
	zero := "над нулём"
	if in.macdNow < 0 {
		zero = "под нулём"
	}
	roomPct := 0.0
	if in.dailyATR > 0 {
		roomPct = (1 - in.todayRange/in.dailyATR) * 100
	}
	return fmt.Sprintf(
		"Тренд↑ (close %.4f > EMA%d %.4f); MACD бычий кросс %s (%.4f); объём > %.2g×ср(%d); дневной ATR-запас %.0f%% (прошло %.4f из %.4f); ATR(ч)=%.4f, ATR(д)=%.4f; SL=%.4f (-%.4f); TP=%.4f (+%.4f, %.2gR)",
		in.price, s.p.EMAPeriod, in.emaTrend, zero, in.macdNow,
		s.p.VolMultiplier, s.p.VolLookback,
		roomPct, in.todayRange, in.dailyATR,
		in.atr, in.dailyATR,
		stop, risk, target, target-in.price, s.p.TakeProfitRR,
	)
}

// manage handles an open long: frozen hard stop, fixed take-profit (reconstructed
// from the frozen entry stop), or an optional armed chandelier trail.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	entry := in.pos.PurchasePrice
	hardSL := in.pos.StopLoss
	// TP is reconstructed from the frozen entry stop (Position carries no target):
	// risk and stop are both fixed at entry, so this is deterministic.
	risk := entry - hardSL
	tp := entry + s.p.TakeProfitRR*risk

	sig.StopLoss = hardSL
	sig.TakeProfit = tp

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
	case s.p.UseTrail == 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
	case s.p.UseTrail == 1:
		chandelier := in.recentHigh - s.p.TrailMult*in.atr
		armed := s.p.TrailArmATR <= 0 || in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR
		if armed && in.barLow <= chandelier {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
	}
	return sig
}

// recentLow returns the lowest low over the last window bars (all if fewer);
// a non-positive window is clamped to the last bar.
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

// recentHigh returns the highest high over the last window bars (all if fewer);
// a non-positive window is clamped to the last bar.
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
