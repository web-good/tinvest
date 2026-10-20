package core

import (
	"fmt"
	"strings"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
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
	fmt.Fprintf(&sb, "%s день: сигнал %s %s %s, покупка %s (Пн–Пт, покупка в ту же дату)\n",
		mark(s.entryAllowed(md)), s.tf(), t.Weekday(), t.Format("15:04"), md.Times[i].Add(s.step).In(mskLoc).Format("15:04"))
	fmt.Fprintf(&sb, "свеча %s: open %.4f high %.4f low %.4f close %.4f\n", s.tf(), md.Opens[i], md.Highs[i], md.Lows[i], md.Closes[i])
	s.explainEntry(&sb, md)
	if atr := s.dailyATR(md); atr > 0 {
		fmt.Fprintf(&sb, "✓ дневной ATR(%d): %.4f, стоп %.2f ATR\n", p.DailyATRPeriod, atr, p.StopDailyATR)
		if p.MaxDayRangeATR > 0 {
			spent := dayRange(barsOf(md), n-1) / atr
			fmt.Fprintf(&sb, "%s диапазон дня %.2f ATR < %.2f (запас хода)\n", mark(s.roomOK(spent)), spent, p.MaxDayRangeATR)
		}
	} else {
		fmt.Fprintf(&sb, "✗ дневной ATR(%d): не посчитан\n", p.DailyATRPeriod)
	}
	if why, ok := s.indexOK(md.Times[i]); why != "" {
		fmt.Fprintf(&sb, "%s тренд индекса вниз: %s\n", mark(ok), why)
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

// explainEntry reports the entry gates of the current bar: levels, the setups of this bar, the
// held PDH breakout and the target.
func (s *Strategy) explainEntry(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	pdl, pdh, ok := s.levels(md)
	if !ok {
		sb.WriteString("✗ уровни: нет будней дневки\n")
		return
	}
	fmt.Fprintf(sb, "уровни вчера: PDL %.4f, PDH %.4f\n", pdl, pdh)
	ser, k := barsOf(md), len(md.Closes)-1
	fmt.Fprintf(sb, "%s возврат над PDL на этой свече (включён %d): зелёная, close > PDL после красного закрытия ниже PDL\n",
		mark(isReclaim(ser, k, pdl)), p.EntryPDL)
	fmt.Fprintf(sb, "%s пробой PDH на этой свече (включён %d): зелёная, первое close > PDH\n", mark(isBreakout(ser, k, pdh)), p.EntryPDH)
	if s.volumeOn() && isBreakout(ser, k, pdh) {
		vol, avg, okv := s.volumeOK(ser, k)
		fmt.Fprintf(sb, "%s объём пробоя %.0f > %.2f × среднее %d предыдущих дневных свечей %.0f\n", mark(okv), vol, p.VolumeMult, p.VolBars, avg)
	}
	if st, minClose, ok := s.held(ser, k, pdh); ok {
		fmt.Fprintf(sb, "✓ пробой PDH на свече %s держится над PDH %d св. (мин. close %.4f)\n",
			ser.t[st.bar].In(mskLoc).Format("15:04"), k-st.bar, minClose)
		explainTarget(sb, st, pdl, pdh, ser.c[k])
		return
	}
	fmt.Fprintf(sb, "✗ пробоя PDH %d св. назад, удержанного над PDH, нет\n", p.HoldBars)
	if p.EntryPDL <= 0 || !isReclaim(ser, k, pdl) {
		return
	}
	if s.patternsOn() {
		found := "нет"
		name, at, ok := s.pdlPattern(ser, k, pdl)
		if ok {
			found = fmt.Sprintf("%s %s", name, ser.t[at].In(mskLoc).Format("15:04"))
		}
		fmt.Fprintf(sb, "%s свечной паттерн между пробоем PDL и возвратом (поглощение %d, молот %d): %s\n", mark(ok), p.PDLEngulf, p.PDLHammer, found)
		if !ok {
			return
		}
	}
	if s.dipVolumeOn() {
		dip, avg, ok := s.dipVolumeOK(ser, k, pdl)
		fmt.Fprintf(sb, "%s объём серии под PDL %.0f > %.2f × обычный %.0f (свечи основной сессии от пробоя до возврата против %d до него)\n",
			mark(ok), dip, p.PDLDipVolMult, avg, p.VolBars)
		if !ok {
			return
		}
	}
	explainTarget(sb, setup{bar: k, name: "PDL", level: pdl}, pdl, pdh, ser.c[k])
}

// explainTarget reports whether the close of the trigger bar is below the target of setup st.
func explainTarget(sb *strings.Builder, st setup, pdl, pdh, closeP float64) {
	tg := targetOf(st, pdl, pdh)
	fmt.Fprintf(sb, "%s цель %.4f (уровень + половина вчерашнего диапазона): close %.4f ниже\n", mark(closeP < tg), tg, closeP)
}

// explainExits reports every exit gate of a held position in priority order.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData) {
	pos := md.Position
	i := len(md.Closes) - 1
	sb.WriteString("--- выходы ---\n")
	stop := s.stop(md, pos)
	fmt.Fprintf(sb, "%s стоп: вход %.4f, стоп %.4f, low %.4f\n", mark(stop > 0 && md.Lows[i] <= stop), pos.PurchasePrice, stop, md.Lows[i])
	if pos.TakeProfit > 0 {
		fmt.Fprintf(sb, "%s цель %.4f, high %.4f\n", mark(md.Highs[i] >= pos.TakeProfit), pos.TakeProfit, md.Highs[i])
	} else {
		sb.WriteString("цель: нет\n")
	}
}

// mark is ✓ for a passed gate and ✗ for a failed one.
func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
