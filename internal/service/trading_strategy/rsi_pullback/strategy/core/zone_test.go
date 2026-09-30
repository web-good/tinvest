package core

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// zoneParams — entryParams() с включённым zone-входом. ZoneRSILower 15 — та же полоса, которую
// RSI(4) entryFixture пересекает на последнем баре (15.6 → 11.1), так что zone-крест на фикстуре
// есть. ZoneEMAPeriod 200 лежит далеко под close: фикстура — 400 баров роста по +0.08%.
func zoneParams() Params {
	p := entryParams()
	p.UseZoneEntry = 1
	p.ZoneRSIPeriod = 4
	p.ZoneRSILower = 15
	p.ZoneEMAPeriod = 200
	return p
}

// blockPullbackTrend закрывает гейт тренда pullback: при EMAFast == EMASlow ema.Compute отдаёт
// бит-в-бит одинаковые ряды, fast[i] == slow[i] не проходит строгое «>». Zone от этих полей не
// зависит.
func blockPullbackTrend(p *Params) { p.EMAFast = p.EMASlow }

// TestZoneEntryBuysWhenPullbackIsBlocked закрывает по одному гейту pullback и требует, чтобы
// (а) с выключенным zone сигнала не было — случай действительно блокирует pullback, и
// (б) с включённым zone была покупка с причиной «zone:». Гейт дня и объёмный гейт здесь же
// доказывают, что на zone-вход они не действуют.
func TestZoneEntryBuysWhenPullbackIsBlocked(t *testing.T) {
	tests := []struct {
		name               string
		tweak              func(p *Params, md *strategy.MarketData)
		todayHigh, todayLo float64
	}{
		{"гейт тренда pullback закрыт", func(p *Params, _ *strategy.MarketData) { blockPullbackTrend(p) }, 101, 100},
		{"RSI pullback не пересёк свою полосу", func(p *Params, _ *strategy.MarketData) { p.RSILower = 0.5 }, 101, 100},
		// used = 5 = 0.5 ATR: между FreshDayATR 0.3 и SpentDayATR 0.8 — мёртвая зона гейта дня.
		{"мёртвая зона гейта дня", func(_ *Params, _ *strategy.MarketData) {}, 105, 100},
		{"объёмный гейт закрыт", func(p *Params, md *strategy.MarketData) {
			p.UseVolume = 1
			md.Volumes[len(md.Volumes)-1] = 1000 // плоский фон: ни один бар не выше слота
		}, 101, 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := zoneParams()
			md := entryFixture()
			tc.tweak(&p, &md)
			md = withDay(md, 10.0, tc.todayHigh, tc.todayLo)

			off := p
			off.UseZoneEntry = 0
			if got := NewWithParams("T", off).Decide(md); got.Kind != model.SignalNone {
				t.Fatalf("zone выключен: Kind = %v, want None — случай должен блокировать pullback сам (reason %q)",
					got.Kind, got.EntryReason)
			}
			got := NewWithParams("T", p).Decide(md)
			if got.Kind != model.SignalBuy {
				t.Fatalf("zone включён: Kind = %v, want Buy", got.Kind)
			}
			if !strings.HasPrefix(got.EntryReason, "zone:") {
				t.Fatalf("EntryReason = %q, want prefix %q", got.EntryReason, "zone:")
			}
		})
	}
}

// TestZoneEntryUsesPullbackStopAndTarget: уровни zone-входа считаются полями pullback и
// совпадают с тем, что DesiredStop построит для позиции.
func TestZoneEntryUsesPullbackStopAndTarget(t *testing.T) {
	p := zoneParams()
	blockPullbackTrend(&p)
	p.StopDailyATR, p.TPDailyATR = 0.7, 1.1
	md := withDay(entryFixture(), 10.0, 101, 100) // дневной ATR фикстуры ровно 10
	got := NewWithParams("T", p).Decide(md)
	if got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", got.Kind)
	}
	entry := md.Closes[len(md.Closes)-1]
	if math.Abs(got.ATR-10) > 1e-9 {
		t.Fatalf("ATR = %v, want 10 (дневной ATR)", got.ATR)
	}
	if want := entry - 0.7*10; math.Abs(got.StopLoss-want) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v (entry − StopDailyATR·ATR)", got.StopLoss, want)
	}
	if want := entry + 1.1*10; math.Abs(got.TakeProfit-want) > 1e-9 {
		t.Fatalf("TakeProfit = %v, want %v (entry + TPDailyATR·ATR)", got.TakeProfit, want)
	}
	if level, _ := DesiredStop(p, entry, got.ATR, entry); math.Abs(level-got.StopLoss) > 1e-9 {
		t.Fatalf("DesiredStop = %v, sig.StopLoss = %v: живой раннер и вход разошлись", level, got.StopLoss)
	}
}

// TestPullbackWinsWhenBothEntriesFire: на entryFixture срабатывают оба входа (zone — см. первый
// случай TestZoneEntryBuysWhenPullbackIsBlocked на той же фикстуре), покупка одна и по pullback.
func TestPullbackWinsWhenBothEntriesFire(t *testing.T) {
	md := withDay(entryFixture(), 10.0, 101, 100)
	got := NewWithParams("T", zoneParams()).Decide(md)
	if got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", got.Kind)
	}
	if strings.HasPrefix(got.EntryReason, "zone:") {
		t.Fatalf("EntryReason = %q: при совпадении входит pullback", got.EntryReason)
	}
}

// TestZoneEntryGates ломает по одному условию zone-входа при закрытом pullback и требует
// отсутствия сигнала. Базовый случай проверяется первым: без него таблица была бы пустой
// проверкой «None по любой причине».
func TestZoneEntryGates(t *testing.T) {
	base := zoneParams()
	blockPullbackTrend(&base)
	if got := NewWithParams("T", base).Decide(withDay(entryFixture(), 10.0, 101, 100)); got.Kind != model.SignalBuy {
		t.Fatalf("базовый случай: Kind = %v, want Buy — таблица ниже была бы пустой", got.Kind)
	}
	tests := []struct {
		name    string
		tweak   func(p *Params, md *strategy.MarketData)
		noDaily bool
	}{
		{"UseZoneEntry=3 — неизвестное значение, значит выключен", func(p *Params, _ *strategy.MarketData) { p.UseZoneEntry = 3 }, false},
		{"ZoneRSIPeriod=0", func(p *Params, _ *strategy.MarketData) { p.ZoneRSIPeriod = 0 }, false},
		{"ZoneRSILower=0", func(p *Params, _ *strategy.MarketData) { p.ZoneRSILower = 0 }, false},
		{"ZoneEMAPeriod=0", func(p *Params, _ *strategy.MarketData) { p.ZoneEMAPeriod = 0 }, false},
		{"RSI не пересёк zone-полосу", func(p *Params, _ *strategy.MarketData) { p.ZoneRSILower = 0.5 }, false},
		// Пять баров подряд вниз: EMA(3) — взвешенное среднее более высоких close, она выше текущего.
		{"close ниже EMA", func(p *Params, _ *strategy.MarketData) { p.ZoneEMAPeriod = 3 }, false},
		// 405 баров при периоде 1000: ema.Compute отдаёт нули — «не прогрета».
		{"EMA не прогрета", func(p *Params, _ *strategy.MarketData) { p.ZoneEMAPeriod = 1000 }, false},
		{"суббота", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 6, 12, 0, 0, 0, msk)) // 2026-06-06 — суббота
		}, false},
		{"нет дневного ATR", func(_ *Params, _ *strategy.MarketData) {}, true},
		// ATR 10 · 1000 намного больше цены фикстуры (~137): стоп ниже нуля — голый лонг.
		{"стоп на нуле или ниже", func(p *Params, _ *strategy.MarketData) { p.StopDailyATR = 1000 }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			md := entryFixture()
			tc.tweak(&p, &md)
			if !tc.noDaily {
				md = withDay(md, 10.0, 101, 100)
			}
			if got := NewWithParams("T", p).Decide(md); got.Kind != model.SignalNone {
				t.Fatalf("Kind = %v, want None (reason %q)", got.Kind, got.EntryReason)
			}
		})
	}
}

// TestZoneOffLeavesSignalsUntouched — сторож критерия 1 спеки: заполненные zone-поля при
// UseZoneEntry=0 не меняют ни одного сигнала и окна свечей.
func TestZoneOffLeavesSignalsUntouched(t *testing.T) {
	plain := entryParams()
	filled := entryParams()
	filled.ZoneRSIPeriod, filled.ZoneRSILower, filled.ZoneEMAPeriod = 4, 15, 200 // UseZoneEntry остаётся 0

	exit := upperCrossFixture()
	last := len(exit.Closes) - 1
	fixtures := map[string]strategy.MarketData{
		"вход pullback":         withDay(entryFixture(), 10.0, 101, 100),
		"мёртвая зона дня":      withDay(entryFixture(), 10.0, 105, 100),
		"нисходящий тренд":      withDay(downtrendFixture(), 10.0, 101, 100),
		"открытая позиция, RSI": withPosition(exit, exit.Closes[last]*0.97, 0, 2),
	}
	for name, md := range fixtures {
		t.Run(name, func(t *testing.T) {
			a := NewWithParams("T", plain).Decide(md)
			b := NewWithParams("T", filled).Decide(md)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("сигналы разошлись:\nбез zone-полей: %+v\nс полями, UseZoneEntry=0: %+v", a, b)
			}
		})
	}
	if a, b := NewWithParams("T", plain).Lookback(), NewWithParams("T", filled).Lookback(); a != b {
		t.Fatalf("Lookback = %d без полей и %d с полями при UseZoneEntry=0", a, b)
	}
}

// TestLookbackCountsZoneOnlyWhenArmed: zone-периоды растят окно только при включённом входе.
func TestLookbackCountsZoneOnlyWhenArmed(t *testing.T) {
	p := DefaultParams() // EMASlow 100 → 2·100+20 = 220
	p.ZoneEMAPeriod, p.ZoneRSIPeriod = 200, 4
	if got := NewWithParams("T", p).Lookback(); got != 220 {
		t.Fatalf("zone выключен: Lookback = %d, want 220", got)
	}
	p.UseZoneEntry = 1
	if got := NewWithParams("T", p).Lookback(); got != 420 {
		t.Fatalf("zone включён, ZoneEMAPeriod 200: Lookback = %d, want 420", got)
	}
	p.ZoneEMAPeriod, p.ZoneRSIPeriod = 50, 300
	if got := NewWithParams("T", p).Lookback(); got != 620 {
		t.Fatalf("zone включён, ZoneRSIPeriod 300: Lookback = %d, want 620", got)
	}
}

// TestZonePositionExitsByPullbackRules: у позиции нет «варианта входа», выход — RSI pullback
// (RSIUpper 70) и при включённом zone.
func TestZonePositionExitsByPullbackRules(t *testing.T) {
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.97, 0, 2)
	got := NewWithParams("T", zoneParams()).Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI", got.Kind, got.Reason)
	}
}

func TestExplainReportsZoneEntry(t *testing.T) {
	md := withDay(entryFixture(), 10.0, 101, 100)
	if got := NewWithParams("T", entryParams()).Explain(md); !strings.Contains(got, "zone-вход: выключен") {
		t.Fatalf("Explain при UseZoneEntry=0 не говорит, что zone выключен:\n%s", got)
	}
	got := NewWithParams("T", zoneParams()).Explain(md)
	for _, want := range []string{"zone-вход: RSI(4)", "EMA(200)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Explain при UseZoneEntry=1 не упоминает %q:\n%s", want, got)
		}
	}
}

// TestZoneOnlyNeverEntersByPullback: при UseZoneEntry=2 вход pullback не проверяется вовсе.
// Первый случай — только pullback (zone-полоса 0.5 фикстурой не пересекается): режим 1 покупает,
// режим 2 молчит. Второй — срабатывают оба: режим 1 входит pullback, режим 2 — zone.
func TestZoneOnlyNeverEntersByPullback(t *testing.T) {
	md := withDay(entryFixture(), 10.0, 101, 100)

	pullbackOnly := zoneParams()
	pullbackOnly.ZoneRSILower = 0.5
	if got := NewWithParams("T", pullbackOnly).Decide(md); got.Kind != model.SignalBuy || strings.HasPrefix(got.EntryReason, "zone:") {
		t.Fatalf("режим 1, сигнал только pullback: Kind/Reason = %v/%q, want Buy по pullback", got.Kind, got.EntryReason)
	}
	pullbackOnly.UseZoneEntry = ZoneEntryOnly
	if got := NewWithParams("T", pullbackOnly).Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("режим 2, сигнал только pullback: Kind = %v, want None (reason %q)", got.Kind, got.EntryReason)
	}

	both := zoneParams()
	both.UseZoneEntry = ZoneEntryOnly
	got := NewWithParams("T", both).Decide(md)
	if got.Kind != model.SignalBuy || !strings.HasPrefix(got.EntryReason, "zone:") {
		t.Fatalf("режим 2, срабатывают оба: Kind/Reason = %v/%q, want Buy с причиной zone:", got.Kind, got.EntryReason)
	}
}

// TestZoneOnlyBuysWhatZoneBuys: закрытые гейты pullback (тренд, день, объём) режиму 2 не мешают —
// те же случаи, что у режима 1 в TestZoneEntryBuysWhenPullbackIsBlocked.
func TestZoneOnlyBuysWhatZoneBuys(t *testing.T) {
	tweaks := map[string]func(p *Params, md *strategy.MarketData){
		"гейт тренда pullback закрыт": func(p *Params, _ *strategy.MarketData) { blockPullbackTrend(p) },
		"объёмный гейт закрыт": func(p *Params, md *strategy.MarketData) {
			p.UseVolume = 1
			md.Volumes[len(md.Volumes)-1] = 1000
		},
	}
	for name, tweak := range tweaks {
		t.Run(name, func(t *testing.T) {
			p := zoneParams()
			p.UseZoneEntry = ZoneEntryOnly
			md := entryFixture()
			tweak(&p, &md)
			got := NewWithParams("T", p).Decide(withDay(md, 10.0, 105, 100)) // 0.5 ATR — мёртвая зона дня
			if got.Kind != model.SignalBuy || !strings.HasPrefix(got.EntryReason, "zone:") {
				t.Fatalf("Kind/Reason = %v/%q, want Buy с причиной zone:", got.Kind, got.EntryReason)
			}
		})
	}
}

// TestZoneOnlyPositionExitsByPullbackRules: у позиции режима 2 выходы те же — RSI pullback.
func TestZoneOnlyPositionExitsByPullbackRules(t *testing.T) {
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.97, 0, 2)
	p := zoneParams()
	p.UseZoneEntry = ZoneEntryOnly
	got := NewWithParams("T", p).Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI", got.Kind, got.Reason)
	}
}

// TestLookbackZoneOnlyDropsPullbackEntryWindows: в режиме 2 окно не держит периоды EMA pullback и
// объёмный фон — они на вход не влияют. RSIPeriod остаётся: на нём RSI-выход.
func TestLookbackZoneOnlyDropsPullbackEntryWindows(t *testing.T) {
	p := DefaultParams() // EMASlow 100
	p.UseVolume = 1      // (14+1)·48·7/5 = 1008
	p.UseZoneEntry, p.ZoneEMAPeriod, p.ZoneRSIPeriod = ZoneEntryOnly, 50, 4
	if got := NewWithParams("T", p).Lookback(); got != 120 {
		t.Fatalf("режим 2, ZoneEMAPeriod 50: Lookback = %d, want 120 (2·50+20)", got)
	}
	p.ZoneEMAPeriod = 200
	if got := NewWithParams("T", p).Lookback(); got != 420 {
		t.Fatalf("режим 2, ZoneEMAPeriod 200: Lookback = %d, want 420", got)
	}
	p.ZoneEMAPeriod, p.RSIPeriod = 10, 100
	if got := NewWithParams("T", p).Lookback(); got != 220 {
		t.Fatalf("режим 2, RSIPeriod 100: Lookback = %d, want 220 — RSI-выход обязан прогреться", got)
	}
}

func TestExplainReportsZoneOnly(t *testing.T) {
	p := zoneParams()
	p.UseZoneEntry = ZoneEntryOnly
	got := NewWithParams("T", p).Explain(withDay(entryFixture(), 10.0, 101, 100))
	for _, want := range []string{"вход pullback: выключен", "zone-вход: RSI(4)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Explain при UseZoneEntry=2 не упоминает %q:\n%s", want, got)
		}
	}
}

func TestParamsZoneArmed(t *testing.T) {
	for mode, want := range map[int]bool{0: false, ZoneEntryAlso: true, ZoneEntryOnly: true, 3: false, -1: false} {
		if got := (Params{UseZoneEntry: mode}).ZoneArmed(); got != want {
			t.Errorf("UseZoneEntry=%d: ZoneArmed = %v, want %v", mode, got, want)
		}
	}
}

// TestPullbackEntryReasonGolden пинит точный текст причины pullback-входа — он уходит в живой
// Telegram и журнал сделок. Эталон снят с кода main (f369b65), до выделения exitPlan: рефакторинг
// не должен молча менять ни слова.
func TestPullbackEntryReasonGolden(t *testing.T) {
	cases := []struct {
		name  string
		trail bool
		want  string
	}{
		{"без трейла", false, "RSI(4) ушёл под 15 (11.1) на откате, EMA(10) 136.9239 > EMA(100) 132.8734, день прошёл 0.10 ATR (дневной ATR 10.0000); вход 136.3237, стоп 131.3237 (−0.50 ATR), цель 142.3237 (+0.60 ATR)"},
		{"с трейлом", true, "RSI(4) ушёл под 15 (11.1) на откате, EMA(10) 136.9239 > EMA(100) 132.8734, день прошёл 0.10 ATR (дневной ATR 10.0000); вход 136.3237, стоп 131.3237 (−0.50 ATR), трейл −0.50 ATR от максимума (с первого бара), цель 142.3237 (+0.60 ATR)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := entryParams()
			if tc.trail {
				p.UseTrail = 1
				p.TrailDailyATR = 0.5
			}
			got := NewWithParams("T", p).Decide(withDay(entryFixture(), 10.0, 101, 100))
			if got.Kind != model.SignalBuy {
				t.Fatalf("Kind = %v, want Buy", got.Kind)
			}
			if got.EntryReason != tc.want {
				t.Fatalf("EntryReason =\n%q\nwant\n%q", got.EntryReason, tc.want)
			}
		})
	}
}
