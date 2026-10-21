package core

import (
	"fmt"
	"strings"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace (-explain).
// It recomputes the values enter() and manage() use, marks each gate, and ends with the signal
// Decide actually returns on this bar, so the verdict can never disagree with a real run.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		sb.WriteString("недостаточно свечей или серии не выровнены\n")
		return sb.String()
	}
	if md.Position != nil {
		s.explainExits(&sb, md)
	} else {
		s.explainEntry(&sb, md)
	}
	sig := s.Decide(md)
	switch sig.Kind {
	case model.SignalBuy:
		fmt.Fprintf(&sb, "ИТОГ: ПОКУПКА по open следующей свечи — %s\n", sig.EntryReason)
	case model.SignalSell:
		fmt.Fprintf(&sb, "ИТОГ: ПРОДАЖА %s — %s\n", sig.Reason, sig.ExitReason)
	default:
		if md.Position != nil {
			sb.WriteString("ИТОГ: держим (в позиции новые входы не ищутся)\n")
		} else {
			sb.WriteString("ИТОГ: входа нет\n")
		}
	}
	return sb.String()
}

// explainEntry reports the entry gates in the order enter() checks them, without stopping at the
// first failure.
func (s *Strategy) explainEntry(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	k := len(md.Closes) - 1
	t := md.Times[k].In(mskLoc)
	buy := md.Times[k].Add(s.step).In(mskLoc)
	fmt.Fprintf(sb, "%s календарь: сигнал %s %s, покупка %s %s (будни, та же дата)\n",
		mark(!isWeekend(t) && dayKey(buy) == dayKey(t)), t.Weekday(), t.Format("15:04"), buy.Weekday(), buy.Format("01-02 15:04"))

	o, h, l, c := md.Opens[k], md.Highs[k], md.Lows[k], md.Closes[k]
	rng := h - l
	body := max(o, c) - min(o, c)
	bodyShare, bottomPos := 0.0, 0.0
	if rng > 0 {
		bodyShare, bottomPos = body/rng, (min(o, c)-l)/rng
	}
	fmt.Fprintf(sb, "%s молот: диапазон %.4f, тело %.4f (%.0f%% диапазона, нужно > 0 и ≤ 33%%), низ тела на %.0f%% диапазона (нужно ≥ 67%%)\n",
		mark(isHammer(o, h, l, c)), rng, body, 100*bodyShare, 100*bottomPos)

	under := false
	if p.TrendEMA > 0 {
		e := ema.Compute(md.Closes, p.TrendEMA)[k]
		switch {
		case e <= 0:
			fmt.Fprintf(sb, "✗ тренд: EMA(%d) не прогрета\n", p.TrendEMA)
		case c > e:
			fmt.Fprintf(sb, "✓ тренд: close %.4f над EMA(%d) %.4f\n", c, p.TrendEMA, e)
		case c < e && p.BelowEMA > 0:
			under = true
			fmt.Fprintf(sb, "✓ тренд: close %.4f под EMA(%d) %.4f — вход под EMA (BelowEMA)\n", c, p.TrendEMA, e)
		default:
			fmt.Fprintf(sb, "✗ тренд: close %.4f не над EMA(%d) %.4f, вход под EMA выключен (BelowEMA 0)\n", c, p.TrendEMA, e)
		}
	} else {
		sb.WriteString("тренд: фильтр выключен (TrendEMA 0)\n")
	}

	period, zone := p.RSIPeriod, p.RSIZone
	if under {
		period, zone = p.BelowRSIPeriod, p.BelowRSIZone
	}
	if period > 0 && k > period {
		rsi := indicators.RSISeries(md.Closes, period)[k]
		fmt.Fprintf(sb, "%s RSI(%d) на молоте %.1f (нужно < %.0f)\n", mark(rsi < zone), period, rsi, zone)
	} else {
		fmt.Fprintf(sb, "✗ RSI(%d): не прогрет или выключен\n", period)
	}

	switch vol, avg, ok := s.volumeOK(md, k); {
	case p.VolumeMult <= 0 || p.VolBars <= 0:
		sb.WriteString("объём: фильтр выключен (VolumeMult 0)\n")
	case avg <= 0:
		fmt.Fprintf(sb, "✗ объём: нет объёмов или меньше %d свечей основной сессии в окне\n", p.VolBars)
	default:
		fmt.Fprintf(sb, "%s объём %.0f, среднее %d свечей основной сессии %.0f, порог %.1f× = %.0f (%.2f×)\n",
			mark(ok), vol, p.VolBars, avg, p.VolumeMult, p.VolumeMult*avg, vol/avg)
	}

	if atr, dr, ok := s.dailyLevels(md); ok {
		target := "цели нет"
		if p.TargetRangeMult > 0 && !under {
			target = fmt.Sprintf("цель %.4f", c+p.TargetRangeMult*dr)
		}
		fmt.Fprintf(sb, "✓ дневные уровни: ATR(%d) %.4f, вчерашний диапазон %.4f; от close: стоп %.4f, %s\n",
			p.DailyATRPeriod, atr, dr, c-p.StopDailyATR*atr, target)
	} else {
		fmt.Fprintf(sb, "✗ дневные уровни: мало будних дневок для ATR(%d) или нулевой диапазон\n", p.DailyATRPeriod)
	}
}

// explainExits reports the exit gates of a held position in the order manage() checks them.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	pos := md.Position
	n := len(md.Closes)
	sb.WriteString("--- выходы ---\n")
	under := s.boughtUnder(pos)
	mode := "над EMA (или без тренда)"
	if under {
		mode = fmt.Sprintf("под EMA %.4f", pos.EntryLevel)
	}
	fmt.Fprintf(sb, "режим: %s, вход %.4f\n", mode, pos.PurchasePrice)
	stop, target, ok := s.exitLevels(pos)
	if !ok {
		sb.WriteString("✗ уровни выхода не восстановлены (нет стопа или ATR входа)\n")
		return
	}
	low, high := md.Lows[n-1], md.Highs[n-1]
	fmt.Fprintf(sb, "%s стоп %.4f: low %.4f\n", mark(low <= stop), stop, low)
	if target > 0 {
		fmt.Fprintf(sb, "%s цель %.4f: high %.4f\n", mark(high >= target), target, high)
	} else {
		sb.WriteString("цель: нет\n")
	}
	if under {
		x, per := p.BelowRSIExit, p.BelowRSIPeriod
		if x <= 0 || per <= 0 || n <= per+1 {
			sb.WriteString("RSI-выход: выключен (BelowRSIExit 0) или не прогрет\n")
			return
		}
		r := indicators.RSISeries(md.Closes, per)
		fmt.Fprintf(sb, "%s RSI(%d) %.1f → %.1f, пересечение %.0f снизу вверх\n",
			mark(r[n-2] <= x && r[n-1] > x), per, r[n-2], r[n-1], x)
		return
	}
	if p.RSIExit <= 0 || p.RSIPeriod <= 0 || n <= p.RSIPeriod {
		sb.WriteString("RSI-выход: выключен (RSIExit 0) или не прогрет\n")
		return
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)[n-1]
	fmt.Fprintf(sb, "%s RSI(%d) %.1f (выход > %.0f)\n", mark(rsi > p.RSIExit), p.RSIPeriod, rsi, p.RSIExit)
}

// mark is ✓ for a passed gate, ✗ otherwise.
func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
