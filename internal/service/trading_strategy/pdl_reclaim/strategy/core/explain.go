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
	ser, k, tf, ok := s.signalBar(md)
	if !ok {
		fmt.Fprintf(&sb, "✗ таймфрейм 30m (с %02d:%02d): на свече %s получас не закрылся — сигнала нет\n",
			p.SwitchHour, p.SwitchMinute, md.Times[i].In(mskLoc).Format("15:04"))
	} else {
		t := ser.t[k].In(mskLoc)
		fmt.Fprintf(&sb, "%s окно: сигнал %s %s %s, покупка %s (сигнал с %02d:%02d, покупка до %02d:%02d, 30m с %02d:%02d, Пн–Пт)\n",
			mark(s.entryAllowed(md, ser, k, tf)), tf, t.Weekday(), t.Format("15:04"), md.Times[i].Add(s.step).In(mskLoc).Format("15:04"),
			p.EntryFromHour, p.EntryFromMinute, p.EntryToHour, p.EntryToMinute, p.SwitchHour, p.SwitchMinute)
		fmt.Fprintf(&sb, "%s входы по %s включены (Entry5m %d, Entry30m %d)\n", mark(s.tfEnabled(tf)), tf, p.Entry5m, p.Entry30m)
		fmt.Fprintf(&sb, "свеча %s: open %.4f high %.4f low %.4f close %.4f\n", tf, ser.o[k], ser.h[k], ser.l[k], ser.c[k])
		closeSig, pb, isBreakout := ser.c[k], k, false
		if pdl, pdh, ok := s.levels(md); ok {
			fmt.Fprintf(&sb, "уровни вчера: PDL %.4f, PDH %.4f\n", pdl, pdh)
			if p.GapOnly > 0 {
				fmt.Fprintf(&sb, "%s гэп: день открылся ниже PDL\n", mark(s.gapDown(md, pdl)))
			}
			if p.Confirm > 0 {
				var okc bool
				pb, okc = s.patternBar(ser, k, tf, pdl)
				fmt.Fprintf(&sb, "%s подтверждение: свеча закрылась %.4f > PDL, паттерн на предыдущей свече той же даты\n", mark(okc), ser.c[k])
				if !okc {
					pb = k - 1
				}
			}
		} else {
			sb.WriteString("✗ уровни: нет будней дневки\n")
		}
		if pdl, pdh, ok := s.levels(md); ok && pb >= 0 {
			k := pb // the bar the pattern is read on (the signal bar, or the one before it with Confirm)
			o, l, c := ser.o[k], ser.l[k], ser.c[k]
			fmt.Fprintf(&sb, "%s A (включён %d): зелёная %s, low < PDL %s, close > PDL %s\n",
				mark(isA(ser, k, pdl)), p.EntryA, mark(c > o), mark(l < pdl), mark(c > pdl))
			if k >= 1 {
				po, pc, pl := ser.o[k-1], ser.c[k-1], ser.l[k-1]
				fmt.Fprintf(&sb, "%s B (включён %d): одна дата %s, пред. красная %s, её close %.4f < PDL %s, текущая зелёная %s, close > open пред. %.4f %s, low < low пред. %.4f %s\n",
					mark(isB(ser, k, pdl)), p.EntryB, mark(dayKey(ser.t[k]) == dayKey(ser.t[k-1])), mark(pc < po), pc, mark(pc < pdl),
					mark(c > o), po, mark(c > po), pl, mark(l < pl))
			}
			if k >= 1 {
				fmt.Fprintf(&sb, "%s C (включён %d): одна дата %s, зелёная %s, close %.4f > PDH %s, пред. close %.4f ≤ PDH %s\n",
					mark(isC(ser, k, pdh)), p.EntryC, mark(dayKey(ser.t[k]) == dayKey(ser.t[k-1])), mark(c > o), c, mark(c > pdh),
					ser.c[k-1], mark(ser.c[k-1] <= pdh))
			}
			kind := s.pattern(ser, k, pdl, pdh)
			if p.BodyATR > 0 {
				level, name := pdl, "PDL"
				if kind == "C" {
					level, name = pdh, "PDH"
				}
				atr := s.dailyATR(md)
				body, past, okb := s.strength(ser, k, level, atr)
				fmt.Fprintf(&sb, "%s сила свечи: тело %.2f > %.2f, отрыв от %s %.2f > %.2f (BodyATR %.2f × ATR)\n",
					mark(okb), body, p.BodyATR*atr, name, past, p.BodyATR*atr, p.BodyATR)
			}
			if s.volumeOn() {
				vol, avg, okv := s.volumeOK(ser, k)
				fmt.Fprintf(&sb, "%s объём свечи паттерна %.0f > %.2f × среднее слота %s за %d будних дней %.0f\n",
					mark(okv), vol, p.VolumeMult, ser.t[k].In(mskLoc).Format("15:04"), p.VolDays, avg)
			}
			if tg := s.target(pdl, pdh); tg > 0 && kind != "C" {
				fmt.Fprintf(&sb, "%s цель: close %.4f < цели %.4f\n", mark(closeSig < tg), closeSig, tg)
			}
			isBreakout = kind == "C"
		}
		switch {
		case isBreakout:
			sb.WriteString("RSI-фильтр входа: не для C (пробой PDH), цели нет\n")
		case pb >= 0 && tf == "30m" && s.hourRSIOn():
			rsi, label, ok := s.signalRSI(md, ser, pb, tf)
			fmt.Fprintf(&sb, "%s %s часа до закрытия получаса паттерна %.1f < %.0f\n", mark(ok), label, rsi, p.HourRSIMax)
		case p.RSIEntryMax > 0 && p.RSIPeriod > 0 && pb >= 0:
			rsi, ok := s.entryRSI(ser.c[:pb+1])
			fmt.Fprintf(&sb, "%s RSI(%d) сигнальной свечи %.1f ≤ %.0f\n", mark(ok), p.RSIPeriod, rsi, p.RSIEntryMax)
		default:
			sb.WriteString("RSI-фильтр входа: выключен\n")
		}
	}
	if atr := s.dailyATR(md); atr > 0 {
		fmt.Fprintf(&sb, "✓ дневной ATR(%d): %.4f, стоп %.2f ATR\n", p.DailyATRPeriod, atr, p.StopDailyATR)
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
	closes, from, k, e, tf, ok := s.exitSeries(md, pos)
	if !ok {
		fmt.Fprintf(sb, "RSI/MACD %s: получас не закрылся на этой свече\n", tf)
	} else {
		if from < k {
			fmt.Fprintf(sb, "разрыв: на этой свече закрылись %d получаса, выход проверен на каждом, ниже — последний\n", k-from+1)
		}
		switch {
		case s.breakoutHeld(md, pos):
			sb.WriteString("RSI-выход: не для пробоя PDH (вход выше PDH)\n")
		case p.RSIPeriod > 0 && p.RSIExit > 0 && len(closes) > p.RSIPeriod:
			rsi := indicators.RSISeries(closes, p.RSIPeriod)[k]
			fmt.Fprintf(sb, "%s RSI %s(%d) %.1f ≥ %.0f\n", mark(rsi >= p.RSIExit), tf, p.RSIPeriod, rsi, p.RSIExit)
		default:
			sb.WriteString("RSI-выход: выключен или не прогрет\n")
		}
		if hist := histogram(closes, p); hist != nil {
			fmt.Fprintf(sb, "%s MACD %s: гистограмма %.4f, свеча входа в окне %s\n", mark(macdExit(hist, e, k)), tf, hist[k], mark(e >= 0))
		} else {
			sb.WriteString("MACD-выход: выключен\n")
		}
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
