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

	hb := k
	if p.ConfirmBars > 0 {
		hb = s.explainConfirm(sb, md, k)
	}
	s.explainHammer(sb, md, hb, md.Closes[k])
}

// explainConfirm reports the confirmation of each hammer candidate in the ConfirmBars bars before
// k and returns the bar whose gates to detail: the confirmed hammer, else the current bar (a new
// hammer that would wait for its confirmation).
func (s *Strategy) explainConfirm(sb *strings.Builder, md strategy.MarketData, k int) int {
	o, c := md.Opens[k], md.Closes[k]
	colour := "не зелёная"
	if c > o {
		colour = "зелёная"
	}
	fmt.Fprintf(sb, "подтверждение (ConfirmBars %d): свеча %s, close %.4f\n", s.p.ConfirmBars, colour, c)
	ind := s.series(md)
	if hb, _, ok := s.confirmedHammer(md, k, ind); ok {
		fmt.Fprintf(sb, "✓ подтверждает молот %s (high %.4f)\n", md.Times[hb].In(mskLoc).Format("01-02 15:04"), md.Highs[hb])
		return hb
	}
	for h := k - 1; h >= max(k-s.p.ConfirmBars, 0); h-- {
		if !isHammer(md.Opens[h], md.Highs[h], md.Lows[h], md.Closes[h]) {
			continue
		}
		_, gates := s.hammerGates(md, h, ind)
		fmt.Fprintf(sb, "✗ кандидат %s: фильтры молота %s, close %.4f > high %.4f %s, не сломан и не подтверждён раньше %s\n",
			md.Times[h].In(mskLoc).Format("01-02 15:04"), mark(gates), c, md.Highs[h], mark(c > md.Highs[h]), mark(s.hammerOpen(md, h, k)))
	}
	sb.WriteString("подтверждённого молота нет; ниже — текущая свеча как новый молот (покупки на ней не будет, только после подтверждения)\n")
	return k
}

// explainHammer details the hammer gates of bar k and the stop under its low against the close c
// of the buy signal's bar.
func (s *Strategy) explainHammer(sb *strings.Builder, md strategy.MarketData, k int, cBuy float64) {
	p := s.p
	o, h, l, c := md.Opens[k], md.Highs[k], md.Lows[k], md.Closes[k]
	rng := h - l
	bottom := min(o, c)
	body := max(o, c) - bottom
	tail, upper := bottom-l, h-max(o, c)
	fmt.Fprintf(sb, "%s молот: диапазон %.4f, тело %.4f (нужно > 0), низ тела %.4f, середина %.4f (нужно ≥), нижняя тень %.4f = %.2f тела (нужно ≥ 2), верхняя тень %.4f = %.2f тела (нужно ≤ 1)\n",
		mark(isHammer(o, h, l, c)), rng, body, bottom, l+rng/2, tail, ratio(tail, body), upper, ratio(upper, body))

	if p.TrendEMA > 0 {
		switch e := ema.Compute(md.Closes, p.TrendEMA)[k]; {
		case e <= 0:
			fmt.Fprintf(sb, "✗ тренд: EMA(%d) не прогрета\n", p.TrendEMA)
		default:
			fmt.Fprintf(sb, "%s тренд: close %.4f над EMA(%d) %.4f\n", mark(c > e), c, p.TrendEMA, e)
		}
	} else {
		sb.WriteString("тренд: фильтр выключен (TrendEMA 0)\n")
	}

	if p.RSIPeriod > 0 && k > p.RSIPeriod {
		rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)[k]
		fmt.Fprintf(sb, "%s RSI(%d) на молоте %.1f (нужно < %.0f)\n", mark(rsi < p.RSIZone), p.RSIPeriod, rsi, p.RSIZone)
	} else {
		fmt.Fprintf(sb, "✗ RSI(%d): не прогрет или выключен\n", p.RSIPeriod)
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

	if atr, ok := s.dailyATR(md); ok {
		switch pdl, ok := s.atPDL(md, k, atr); {
		case p.PDLZoneATR <= 0:
			sb.WriteString("вчерашний минимум: фильтр выключен (PDLZoneATR 0)\n")
		case pdl == 0:
			sb.WriteString("✗ вчерашний минимум: нет будней дневки до даты молота\n")
		default:
			d := p.PDLZoneATR * atr
			fmt.Fprintf(sb, "%s вчерашний минимум %.4f: low молота %.4f в зоне %.4f–%.4f (± %.2f ATR)\n", mark(ok), pdl, l, pdl-d, pdl+d, p.PDLZoneATR)
		}
		stop := l - p.StopBufferATR*atr
		target := "цели нет"
		if p.TargetR > 0 {
			target = fmt.Sprintf("цель %.2f R %.4f", p.TargetR, cBuy+p.TargetR*(cBuy-stop))
		}
		fmt.Fprintf(sb, "%s стоп под молотом: low %.4f − %.2f × ATR(%d) %.4f = %.4f (нужно > 0 и < close %.4f), %s\n",
			mark(stop > 0 && stop < cBuy), l, p.StopBufferATR, p.DailyATRPeriod, atr, stop, cBuy, target)
	} else {
		fmt.Fprintf(sb, "✗ дневной ATR(%d): мало будних дневок\n", p.DailyATRPeriod)
	}
}

// explainExits reports the exit gates of a held position in the order manage() checks them.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	pos := md.Position
	n := len(md.Closes)
	sb.WriteString("--- выходы ---\n")
	fmt.Fprintf(sb, "вход %.4f\n", pos.PurchasePrice)
	stop, target, ok := s.exitLevels(pos)
	if !ok {
		sb.WriteString("✗ уровни выхода не восстановлены (нет стопа или ATR входа)\n")
		return
	}
	low, high := md.Lows[n-1], md.Highs[n-1]
	fmt.Fprintf(sb, "%s стоп %.4f: low %.4f\n", mark(low <= stop), stop, low)
	held := s.rsiHeld(md, n-2)
	switch {
	case s.p.RSIHold <= 0 || s.p.MACDFast <= 0:
		sb.WriteString("RSI-удержание: выключено (RSIHold 0 или MACD выключен)\n")
	case held >= 0:
		fmt.Fprintf(sb, "RSI-удержание: RSI(%d) > %.0f на свече %s при зелёной MACD — цель снята, ждём первую красную MACD\n", p.RSIPeriod, p.RSIHold, md.Times[held].In(mskLoc).Format("01-02 15:04"))
	default:
		fmt.Fprintf(sb, "RSI-удержание: RSI(%d) выше %.0f с покупки не закрывался\n", p.RSIPeriod, p.RSIHold)
	}
	if target > 0 && held < 0 {
		fmt.Fprintf(sb, "%s цель %.4f: high %.4f\n", mark(high >= target), target, high)
	} else if target > 0 {
		fmt.Fprintf(sb, "цель %.4f снята RSI-удержанием\n", target)
	} else {
		sb.WriteString("цель: нет\n")
	}
	if s.macdHist(md.Closes) == nil {
		sb.WriteString("MACD-выход: выключен (MACDFast 0) или не прогрет\n")
		return
	}
	if r, hist, ok := s.rsiExit(md); ok {
		fmt.Fprintf(sb, "✓ RSI-выход: свеча %s закрылась с RSI(%d) > %.0f, гистограмма MACD %.4f не зелёная\n", md.Times[r].In(mskLoc).Format("01-02 15:04"), p.RSIPeriod, p.RSIHold, hist)
		return
	}
	switch hist, green, ok := s.macdTurnedRed(md); {
	case ok:
		after := "после зелёной"
		if s.rsiHeld(md, n-1) >= 0 {
			after = fmt.Sprintf("— первая красная после RSI > %.0f на свече", p.RSIHold)
		}
		fmt.Fprintf(sb, "✓ MACD(%d,%d,%d): гистограмма %.4f < 0 %s %s\n", p.MACDFast, p.MACDSlow, p.MACDSignal, hist, after, green.In(mskLoc).Format("01-02 15:04"))
	case hist < 0:
		fmt.Fprintf(sb, "✗ MACD(%d,%d,%d): гистограмма %.4f красная, но зелёной с молота ещё не было\n", p.MACDFast, p.MACDSlow, p.MACDSignal, hist)
	default:
		h := s.macdHist(md.Closes)
		fmt.Fprintf(sb, "✗ MACD(%d,%d,%d): гистограмма %.4f не красная\n", p.MACDFast, p.MACDSlow, p.MACDSignal, h[n-1])
	}
}

// ratio is a/b, or 0 when b is 0.
func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// mark is ✓ for a passed gate, ✗ otherwise.
func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
