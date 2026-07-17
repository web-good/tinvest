package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
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

// sweepScenario — канонический валидный сетап (k=2): ATR-прогрев (2 дня
// флэта), подтверждённый swing-low 97, прокол до 96.5 и reclaim-close 98.5
// последним баром (среда, основная сессия).
func sweepScenario() []bar {
	bars := flatBars(msk(2026, 7, 6, 10), 16, 100)
	bars = append(bars, bar{t: next(bars), h: 101, l: 97, c: 100, v: 100}) // swing low 97
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100}) // confirm
	bars = append(bars, bar{t: next(bars), h: 99, l: 96.5, c: 96.9, v: 100})     // pierce
	bars = append(bars, bar{t: next(bars), h: 99.4, l: 97.7, c: 98.5, v: 100})   // reclaim
	return bars
}

func defParams() Params {
	return Params{SwingK: 2, ReclaimBars: 4, Buffer: 0.5, TPR: 2, MaxHoldDays: 3}
}

func TestEntryOnSweepReclaim(t *testing.T) {
	md := mkMD(sweepScenario(), nil)
	sig := newStrat(defParams()).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("sig = %+v, want Buy", sig)
	}
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, 14)
	wantStop := 96.5 - 0.5*atr
	if sig.StopLoss != wantStop {
		t.Fatalf("StopLoss = %v, want %v", sig.StopLoss, wantStop)
	}
	if sig.Level != 97 || sig.ATR != atr {
		t.Fatalf("Level/ATR = %v/%v, want 97/%v", sig.Level, sig.ATR, atr)
	}
	wantTP := 98.5 + 2*(98.5-wantStop)
	if sig.TakeProfit != wantTP {
		t.Fatalf("TakeProfit = %v, want %v", sig.TakeProfit, wantTP)
	}
	if sig.EntryReason == "" {
		t.Fatal("EntryReason must be set on Buy")
	}
}

func TestNoEntryWhenReclaimTooLate(t *testing.T) {
	p := defParams()
	p.ReclaimBars = 0 // только однобарный sweep; в сценарии gap = 1
	if sig := newStrat(p).Decide(mkMD(sweepScenario(), nil)); sig.Kind != model.SignalNone {
		t.Fatalf("sig = %+v, want None", sig)
	}
}

func TestNoEntryWhenReclaimOnEarlierBar(t *testing.T) {
	bars := sweepScenario()
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100, v: 100})
	if sig := newStrat(defParams()).Decide(mkMD(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("sig = %+v, want None (reclaim был баром раньше)", sig)
	}
}

func TestNoEntryOutsideMainSession(t *testing.T) {
	// Reclaim-бар в вечернюю сессию (19:00 MSK того же дня).
	bars := sweepScenario()
	ev := bars[len(bars)-1].t.In(mskLoc)
	bars[len(bars)-1].t = time.Date(ev.Year(), ev.Month(), ev.Day(), 19, 0, 0, 0, mskLoc)
	if sig := newStrat(defParams()).Decide(mkMD(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("evening bar: sig = %+v, want None", sig)
	}
	// Reclaim-бар в субботу.
	bars = sweepScenario()
	sat := startOfMSKDay(bars[len(bars)-1].t)
	for sat.Weekday() != time.Saturday {
		sat = sat.AddDate(0, 0, 1)
	}
	bars[len(bars)-1].t = sat.Add(12 * time.Hour)
	if sig := newStrat(defParams()).Decide(mkMD(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("saturday bar: sig = %+v, want None", sig)
	}
}

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

func TestSwingLowConfirmedOnlyAfterK(t *testing.T) {
	base := flatBars(msk(2026, 7, 6, 10), 6, 100)
	dip := bar{t: next(base), h: 101, l: 97, c: 100, v: 100}
	bars := append(append([]bar{}, base...), dip)
	// 1 бар после дна (k=2): уровень ещё не подтверждён.
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	md := mkMD(bars, nil)
	if lvls := levelStates(md.Lows, md.Closes, md.Times, 2); len(lvls) != 0 {
		t.Fatalf("levels before confirmation = %d, want 0", len(lvls))
	}
	// 2 бара после дна: уровень 97 подтверждён.
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100})
	md = mkMD(bars, nil)
	lvls := levelStates(md.Lows, md.Closes, md.Times, 2)
	if len(lvls) != 1 || lvls[0].price != 97 {
		t.Fatalf("levels = %+v, want one level at 97", lvls)
	}
	if lvls[0].pierceIdx != -1 || lvls[0].reclaimIdx != -1 {
		t.Fatalf("fresh level must be untouched, got %+v", lvls[0])
	}
}

func TestSweepReclaimLifecycle(t *testing.T) {
	base := flatBars(msk(2026, 7, 6, 10), 6, 100)
	bars := append(append([]bar{}, base...),
		bar{t: next(base), h: 101, l: 97, c: 100, v: 100}, // swing low 97
	)
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100}) // confirm
	bars = append(bars, bar{t: next(bars), h: 99, l: 96.5, c: 96.9, v: 100})     // pierce, close under
	md := mkMD(bars, nil)
	lvls := levelStates(md.Lows, md.Closes, md.Times, 2)
	if len(lvls) != 1 || lvls[0].pierceIdx != len(bars)-1 || lvls[0].reclaimIdx != -1 {
		t.Fatalf("after pierce: %+v", lvls)
	}
	bars = append(bars, bar{t: next(bars), h: 99.4, l: 97.7, c: 98.5, v: 100}) // reclaim close
	md = mkMD(bars, nil)
	lvls = levelStates(md.Lows, md.Closes, md.Times, 2)
	lv := lvls[0]
	if lv.reclaimIdx != len(bars)-1 {
		t.Fatalf("reclaimIdx = %d, want %d", lv.reclaimIdx, len(bars)-1)
	}
	if lv.sweepLow != 96.5 {
		t.Fatalf("sweepLow = %v, want 96.5", lv.sweepLow)
	}
	// Однобарный sweep: прокол и reclaim одной свечой.
	sb := append(append([]bar{}, base...),
		bar{t: next(base), h: 101, l: 97, c: 100, v: 100},
	)
	sb = append(sb, bar{t: next(sb), h: 100.6, l: 97.9, c: 100.3, v: 100})
	sb = append(sb, bar{t: next(sb), h: 100.5, l: 97.8, c: 100.1, v: 100})
	sb = append(sb, bar{t: next(sb), h: 99, l: 96.5, c: 98.2, v: 100}) // wick under, close above
	md = mkMD(sb, nil)
	lv = levelStates(md.Lows, md.Closes, md.Times, 2)[0]
	if lv.pierceIdx != lv.reclaimIdx || lv.reclaimIdx != len(sb)-1 {
		t.Fatalf("same-bar sweep: %+v", lv)
	}
}

func TestReclaimCandidateWindowAndDepth(t *testing.T) {
	lvls := []level{
		{price: 97, pierceIdx: 10, reclaimIdx: 12, sweepLow: 96.5},
		{price: 96, pierceIdx: 11, reclaimIdx: 12, sweepLow: 95.8},
	}
	// Оба reclaim-ятся баром 12 — берём с более глубоким sweepLow.
	cand, ok := reclaimCandidate(lvls, 12, 4)
	if !ok || cand.sweepLow != 95.8 {
		t.Fatalf("cand = %+v ok=%v, want deepest sweepLow 95.8", cand, ok)
	}
	// Просроченный reclaim (gap 2 > maxBars 1) не кандидат.
	if _, ok := reclaimCandidate(lvls[:1], 12, 1); ok {
		t.Fatal("stale reclaim (gap 2 > 1) must not be a candidate")
	}
	// Текущий бар не совпадает с reclaim — не кандидат.
	if _, ok := reclaimCandidate(lvls, 13, 4); ok {
		t.Fatal("reclaim on an earlier bar must not be a candidate")
	}
}

// heldMD — 2 торговых дня флэта + текущий бар с заданными H/L; позиция
// открыта в первый день по 100 со стопом 95.
func heldMD(h, l, c float64) strategy.MarketData {
	bars := flatBars(msk(2026, 7, 6, 10), 16, 100)
	bars = append(bars, bar{t: next(bars), h: h, l: l, c: c, v: 100})
	return mkMD(bars, &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 95,
		EntryTime: msk(2026, 7, 6, 11),
	})
}

func newStrat(p Params) *Strategy { return NewWithParams("TEST", p) }

func TestManageStopBeforeTP(t *testing.T) {
	s := newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 5})
	// TP = 100 + 2*(100-95) = 110; бар задевает и стоп, и тейк — стоп первым.
	sig := s.Decide(heldMD(111, 94, 100))
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("sig = %+v, want Sell/SL", sig)
	}
	if sig.StopLoss != 95 {
		t.Fatalf("StopLoss = %v, want 95", sig.StopLoss)
	}
}

func TestManageTP(t *testing.T) {
	s := newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 5})
	sig := s.Decide(heldMD(110.5, 99, 109))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 110 {
		t.Fatalf("sig = %+v, want Sell/TP@110", sig)
	}
	// TPR <= 0 выключает тейк.
	s = newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 0, MaxHoldDays: 5})
	if sig := s.Decide(heldMD(120, 99, 119)); sig.Kind != model.SignalNone {
		t.Fatalf("TPR=0: sig = %+v, want None", sig)
	}
}

func TestManageTimeStop(t *testing.T) {
	s := newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 2})
	// heldMD: вход Пн, текущий бар Ср → 2 полных торговых дня после входа.
	sig := s.Decide(heldMD(101, 99, 100))
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("sig = %+v, want Sell/TIME", sig)
	}
	// Без EntryTime тайм-стоп деградирует в no-op.
	md := heldMD(101, 99, 100)
	md.Position.EntryTime = time.Time{}
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("zero EntryTime: sig = %+v, want None", sig)
	}
	// MaxHoldDays=5 ещё не истёк.
	s = newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 5})
	if sig := s.Decide(heldMD(101, 99, 100)); sig.Kind != model.SignalNone {
		t.Fatalf("not expired: sig = %+v, want None", sig)
	}
}
