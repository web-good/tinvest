package core

import (
	"fmt"
	"strings"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace (-explain).
// It recomputes the values enter() and manage() use and ends with the signal Decide actually
// returns on this bar, so the verdict can never disagree with a real run.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	p := s.p
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return "недостаточно свечей или нет времени/open свечей\n"
	}
	i := n - 1
	t := md.Times[i].In(mskLoc)
	fmt.Fprintf(&sb, "%s окно: сигнал %s %s, покупка %s (сигнал с %02d:%02d, покупка до %02d:%02d, Пн–Пт)\n",
		mark(s.inEntryWindow(md.Times[i])), t.Weekday(), t.Format("15:04"), md.Times[i].Add(s.step).In(mskLoc).Format("15:04"),
		p.EntryFromHour, p.EntryFromMinute, p.EntryToHour, p.EntryToMinute)

	o, h, l, c := md.Opens[i], md.Highs[i], md.Lows[i], md.Closes[i]
	fmt.Fprintf(&sb, "свеча: open %.4f high %.4f low %.4f close %.4f\n", o, h, l, c)
	if pdl, pdh, ok := s.levels(md); ok {
		fmt.Fprintf(&sb, "уровни вчера: PDL %.4f, PDH %.4f\n", pdl, pdh)
		fmt.Fprintf(&sb, "%s A (включён %d): зелёная %s, low < PDL %s, close > PDL %s\n",
			mark(isA(md, i, pdl)), p.EntryA, mark(c > o), mark(l < pdl), mark(c > pdl))
		po, pc, pl := md.Opens[i-1], md.Closes[i-1], md.Lows[i-1]
		fmt.Fprintf(&sb, "%s B (включён %d): одна дата %s, пред. красная %s, её close %.4f < PDL %s, текущая зелёная %s, close > open пред. %.4f %s, low < low пред. %.4f %s\n",
			mark(isB(md, i, pdl)), p.EntryB, mark(dayKey(md.Times[i]) == dayKey(md.Times[i-1])), mark(pc < po), pc, mark(pc < pdl),
			mark(c > o), po, mark(c > po), pl, mark(l < pl))
		if p.TakeProfitPDH > 0 {
			fmt.Fprintf(&sb, "%s цель: close %.4f < PDH %.4f\n", mark(c < pdh), c, pdh)
		}
	} else {
		sb.WriteString("✗ уровни: нет будней дневки\n")
	}
	if p.RSIEntryMax > 0 && p.RSIPeriod > 0 {
		rsi, ok := s.entryRSI(md)
		fmt.Fprintf(&sb, "%s RSI(%d) сигнальной свечи %.1f ≤ %.0f\n", mark(ok), p.RSIPeriod, rsi, p.RSIEntryMax)
	} else {
		sb.WriteString("RSI-фильтр входа: выключен\n")
	}
	if atr := s.dailyATR(md); atr > 0 {
		fmt.Fprintf(&sb, "✓ дневной ATR(%d): %.4f, стоп от close %.4f\n", p.DailyATRPeriod, atr, c-p.StopDailyATR*atr)
	} else {
		fmt.Fprintf(&sb, "✗ дневной ATR(%d): не посчитан\n", p.DailyATRPeriod)
	}

	if md.Position != nil {
		s.explainExits(&sb, md)
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

// explainExits reports every exit gate of a held position in priority order.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	pos := md.Position
	i := len(md.Closes) - 1
	sb.WriteString("--- выходы ---\n")
	stop := s.stop(md, pos)
	fmt.Fprintf(sb, "%s стоп: вход %.4f, стоп %.4f, low %.4f\n", mark(stop > 0 && md.Lows[i] <= stop), pos.PurchasePrice, stop, md.Lows[i])
	if pos.TakeProfit > 0 {
		fmt.Fprintf(sb, "%s цель PDH %.4f, high %.4f\n", mark(md.Highs[i] >= pos.TakeProfit), pos.TakeProfit, md.Highs[i])
	} else {
		sb.WriteString("цель: нет\n")
	}
	if p.RSIPeriod > 0 && p.RSIExit > 0 && len(md.Closes) > p.RSIPeriod {
		rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)[i]
		fmt.Fprintf(sb, "%s RSI(%d) %.1f ≥ %.0f\n", mark(rsi >= p.RSIExit), p.RSIPeriod, rsi, p.RSIExit)
	} else {
		sb.WriteString("RSI-выход: выключен или не прогрет\n")
	}
	if hist := histogram(md.Closes, p); hist != nil {
		e := entryIndex(md.Times, pos.EntryTime)
		fmt.Fprintf(sb, "%s MACD: гистограмма %.4f, свеча входа в окне %s\n", mark(macdExit(hist, e, i)), hist[i], mark(e >= 0))
	} else {
		sb.WriteString("MACD-выход: выключен\n")
	}
	why := s.eodReason(md, pos)
	fmt.Fprintf(sb, "%s конец дня: %s\n", mark(why != ""), orNone(why))
}

// orNone is s, or "нет" when empty.
func orNone(s string) string {
	if s == "" {
		return "нет"
	}
	return s
}

// mark is ✓ for a passed gate and ✗ for a failed one.
func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
