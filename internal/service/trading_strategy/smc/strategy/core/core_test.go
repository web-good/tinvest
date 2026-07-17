package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// bar — компактная спека свечи для тестов: время открытия MSK + H/L/C/V
// (Open в MarketData отсутствует).
type bar struct {
	t       time.Time
	h, l, c float64
	v       int64
}

// mkMD собирает MarketData из баров (oldest-first) и позиции.
func mkMD(bars []bar, pos *strategy.Position) strategy.MarketData {
	md := strategy.MarketData{Position: pos}
	for _, b := range bars {
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, b.v)
		md.Times = append(md.Times, b.t)
	}
	if n := len(bars); n > 0 {
		md.Price = bars[n-1].c
	}
	return md
}

func msk(y int, m time.Month, d, hh int) time.Time {
	return time.Date(y, m, d, hh, 0, 0, 0, mskLoc)
}

// advance — следующий Hour1-бар: +1ч внутри основной сессии (10..17),
// иначе 10:00 следующего буднего дня.
func advance(t time.Time) time.Time {
	nt := t.Add(time.Hour)
	if nt.In(mskLoc).Hour() < 18 && sameMSKDay(nt, t) {
		return nt
	}
	nt = startOfMSKDay(t).AddDate(0, 0, 1).Add(10 * time.Hour)
	for nt.Weekday() == time.Saturday || nt.Weekday() == time.Sunday {
		nt = nt.AddDate(0, 0, 1)
	}
	return nt
}

// flatBars — n «тихих» баров (H=p+1, L=p-1, C=p) с шагом advance от start.
func flatBars(start time.Time, n int, p float64) []bar {
	out := make([]bar, 0, n)
	t := start
	for i := 0; i < n; i++ {
		out = append(out, bar{t: t, h: p + 1, l: p - 1, c: p, v: 100})
		t = advance(t)
	}
	return out
}

// next — время открытия бара, следующего за последним в срезе.
func next(bars []bar) time.Time { return advance(bars[len(bars)-1].t) }

func TestWindowStartKeepsLastNDays(t *testing.T) {
	// 12 торговых дней по 8 баров: окно должно начинаться с первого бара
	// последних levelWindowDays (10) дней = индекс 2*8.
	bars := flatBars(msk(2026, 6, 1, 10), 12*8, 100)
	md := mkMD(bars, nil)
	if got, want := windowStart(md.Times), 16; got != want {
		t.Fatalf("windowStart = %d, want %d", got, want)
	}
	// Короткая история — окно с нулевого бара.
	short := mkMD(flatBars(msk(2026, 6, 1, 10), 5, 100), nil)
	if got := windowStart(short.Times); got != 0 {
		t.Fatalf("windowStart(short) = %d, want 0", got)
	}
}

func TestTradingDaysSince(t *testing.T) {
	// 3 торговых дня по 8 баров; вход в первый день.
	bars := flatBars(msk(2026, 7, 6, 10), 3*8, 100)
	md := mkMD(bars, nil)
	entry := msk(2026, 7, 6, 12)
	if got := tradingDaysSince(md.Times, entry); got != 2 {
		t.Fatalf("tradingDaysSince = %d, want 2 (Tue, Wed)", got)
	}
	// Вход в последний день — 0 полных дней после.
	if got := tradingDaysSince(md.Times, msk(2026, 7, 8, 10)); got != 0 {
		t.Fatalf("tradingDaysSince(last day) = %d, want 0", got)
	}
}
