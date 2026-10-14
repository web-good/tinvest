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
	p := s.p
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n {
		sb.WriteString("недостаточно свечей или нет времени свечей\n")
		return sb.String()
	}
	i := n - 1
	c := md.Closes[i]
	t := md.Times[i].In(mskLoc)

	fill := md.Times[i].Add(s.step).In(mskLoc)
	weekday := !isWeekend(fill)
	sameDay := dayKey(fill) == dayKey(t)
	hours := s.inBuyHours(fill.Hour())
	fmt.Fprintf(&sb, "%s окно: сигнал %s %s, покупка в %s — будний %s, исполнение в ту же дату %s, часы покупки [%02d:00, %02d:00) или [%02d:00, %02d:00) %s, пятница до %02d:00 %s\n",
		mark(s.inWindow(md.Times[i])), t.Weekday(), t.Format("15:04"), fill.Format("15:04"), mark(weekday), mark(sameDay),
		p.MorningFrom, p.MorningTo, p.EveningFrom, p.EveningTo, mark(hours), p.FridayTo, mark(s.fridayOK(fill, fill.Hour())))

	if p.EMAPeriod > 0 {
		trend := ema.Compute(md.Closes, p.EMAPeriod)[i]
		if trend > 0 {
			fmt.Fprintf(&sb, "%s тренд: close %.4f > EMA(%d) %.4f\n", mark(trendUp(c, trend)), c, p.EMAPeriod, trend)
		} else {
			fmt.Fprintf(&sb, "✗ тренд: EMA(%d) не прогрета\n", p.EMAPeriod)
		}
	} else {
		sb.WriteString("✗ тренд: EMAPeriod 0 — входы выключены\n")
	}

	if p.RSIPeriod > 0 && n >= p.RSIPeriod+2 {
		rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)
		prev, cur := rsi[i-1], rsi[i]
		zone := cur < p.RSIZone
		cross := crossUp(prev, cur, p.RSIZone) && cur <= p.RSICap
		branch := "нет"
		switch {
		case zone:
			branch = "ZONE"
		case cross:
			branch = "CROSS"
		}
		fmt.Fprintf(&sb, "%s RSI(%d): %.1f → %.1f; ZONE (< %.0f) %s, CROSS (крест %.0f вверх, ≤ %.0f) %s; ветка %s\n",
			mark(zone || cross), p.RSIPeriod, prev, cur, p.RSIZone, mark(zone), p.RSIZone, p.RSICap, mark(cross), branch)
	} else {
		fmt.Fprintf(&sb, "✗ RSI(%d): не прогрет\n", p.RSIPeriod)
	}

	atr := s.dailyATR(md)
	if atr > 0 {
		fmt.Fprintf(&sb, "✓ дневной ATR(%d): %.4f\n", p.DailyATRPeriod, atr)
	} else {
		fmt.Fprintf(&sb, "✗ дневной ATR(%d): не посчитан\n", p.DailyATRPeriod)
	}

	switch {
	case p.StopDailyATR <= 0:
		sb.WriteString("✓ стоп: выключен\n")
	case atr > 0:
		stop := c - p.StopDailyATR*atr
		fmt.Fprintf(&sb, "%s стоп от close: %.4f − %.2f×ATR = %.4f > 0\n", mark(stop > 0), c, p.StopDailyATR, stop)
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

// explainExits reports the exit gates of a held position: the stop and the RSI(RSIExitPeriod)
// cross.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	pos := md.Position
	n := len(md.Closes)
	i := n - 1
	sb.WriteString("--- выходы ---\n")
	if p.StopDailyATR > 0 {
		atr := pos.EntryATR
		if atr <= 0 {
			atr = s.dailyATR(md)
		}
		stop := pos.PurchasePrice - p.StopDailyATR*atr
		fmt.Fprintf(sb, "стоп: вход %.4f − %.2f×ATR %.4f = %.4f; low %.4f, задет %s\n",
			pos.PurchasePrice, p.StopDailyATR, atr, stop, md.Lows[i], mark(atr > 0 && stop > 0 && md.Lows[i] <= stop+1e-9))
	}
	if p.RSIExit <= 0 || p.RSIExitPeriod <= 0 || n < p.RSIExitPeriod+2 {
		sb.WriteString("RSI-выход: выключен (RSIExit или RSIExitPeriod 0) или RSI не прогрет\n")
		return
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIExitPeriod)
	fmt.Fprintf(sb, "RSI(%d) %.1f → %.1f, крест %.0f вверх %s\n",
		p.RSIExitPeriod, rsi[i-1], rsi[i], p.RSIExit, mark(crossUp(rsi[i-1], rsi[i], p.RSIExit)))
}

// mark is ✓ for a passed gate and ✗ for a failed one.
func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
