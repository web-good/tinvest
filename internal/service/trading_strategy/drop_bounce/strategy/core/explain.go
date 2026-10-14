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
	t := md.Times[i].In(mskLoc)
	fill := md.Times[i].Add(s.step).In(mskLoc)
	fmt.Fprintf(&sb, "%s будни: сигнал %s %s, покупка %s %s в ту же дату (в пятницу до %02d:00)\n",
		mark(s.isWeekdayFill(md.Times[i])), t.Weekday(), t.Format("15:04"), fill.Weekday(), fill.Format("15:04"), p.FridayTo)

	hs := s.hourBars(md)
	if p.RSIPeriod > 0 && p.EMAPeriod > 0 && len(hs) >= p.RSIPeriod+2 {
		closes := hourCloses(hs)
		trend := ema.Compute(closes, p.EMAPeriod)
		rsi := indicators.RSISeries(closes, p.RSIPeriod)
		k := len(hs) - 1
		fmt.Fprintf(&sb, "последний закрытый час %s (известен на баре %s): close %.4f, EMA(%d) %.4f, RSI(%d) %.1f → %.1f, сигнал %s\n",
			hs[k].start.In(mskLoc).Format("01-02 15:04"), md.Times[hs[k].done].In(mskLoc).Format("15:04"),
			closes[k], p.EMAPeriod, trend[k], p.RSIPeriod, rsi[k-1], rsi[k], orNone(s.hourSignal(closes, trend, rsi, k)))
		if a := s.armedBy(hs, closes, trend, rsi, i); a >= 0 {
			fmt.Fprintf(&sb, "✓ взведено часом %s (%s), бар %d из %d окна подтверждения\n",
				hs[a].start.In(mskLoc).Format("01-02 15:04"), s.hourSignal(closes, trend, rsi, a), i-hs[a].done+1, p.ConfirmBars)
			hist := macdHistogram(md.Closes, p)
			if hist == nil {
				sb.WriteString("✗ MACD 30m: гистограммы нет\n")
			} else {
				fmt.Fprintf(&sb, "%s MACD 30m: гистограмма %.4f → %.4f (зелёная ≥ 0, на взводе — сразу, дальше — перекраска)\n",
					mark(confirmed(hist, hs[a].done, i)), hist[i-1], hist[i])
			}
		} else {
			fmt.Fprintf(&sb, "✗ взвода нет: ни один час с сигналом не закрылся за последние %d бар(а)\n", p.ConfirmBars)
		}
	} else {
		sb.WriteString("✗ часовые RSI/EMA не прогреты или выключены\n")
	}

	atr := s.dailyATR(md)
	if atr > 0 {
		fmt.Fprintf(&sb, "✓ дневной ATR(%d): %.4f\n", p.DailyATRPeriod, atr)
	} else {
		fmt.Fprintf(&sb, "✗ дневной ATR(%d): не посчитан\n", p.DailyATRPeriod)
	}

	if md.Position != nil {
		s.explainExits(&sb, md, hs)
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

// explainExits reports the exit gates of a held position: the stop and the hourly RSI cross.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData, hs []hourBar) {
	p := s.p
	pos := md.Position
	i := len(md.Closes) - 1
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
	if p.BreakevenATR > 0 {
		if fill := fillIndex(md); fill >= 0 && fill < i {
			atr := pos.EntryATR
			if atr <= 0 {
				atr = s.dailyATR(md)
			}
			best := md.Highs[fill]
			for _, h := range md.Highs[fill+1 : i] {
				best = max(best, h)
			}
			fmt.Fprintf(sb, "безубыток: лучший high с покупки %.4f, порог вход + %.2f ATR = %.4f, взведён %s\n",
				best, p.BreakevenATR, pos.PurchasePrice+p.BreakevenATR*atr, mark(atr > 0 && best >= pos.PurchasePrice+p.BreakevenATR*atr-1e-9))
		} else {
			sb.WriteString("безубыток: свеча покупки не в окне или ещё не закрыта\n")
		}
	}
	k := len(hs) - 1
	if p.RSIExit <= 0 || p.RSIExitPeriod <= 0 || k < p.RSIExitPeriod+1 {
		sb.WriteString("RSI-выход: выключен (RSIExit или RSIExitPeriod 0) или часовой RSI не прогрет\n")
		return
	}
	rsi := indicators.RSISeries(hourCloses(hs), p.RSIExitPeriod)
	fmt.Fprintf(sb, "час %s закрыт на этом баре %s; RSI(%d) %.1f → %.1f, крест %.0f вверх %s\n",
		hs[k].start.In(mskLoc).Format("01-02 15:04"), mark(hs[k].done == i), p.RSIExitPeriod, rsi[k-1], rsi[k],
		p.RSIExit, mark(crossUp(rsi[k-1], rsi[k], p.RSIExit)))
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
