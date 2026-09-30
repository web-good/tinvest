# rsi_pullback: второй вход zone вместо стратегии rsi_zone — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в ядро rsi_pullback второй, включаемый по тикеру вариант входа (zone: крест RSI вниз + close > EMA) с общими выходами и удалить отдельную стратегию rsi_zone вместе с поддержкой нескольких стратегий в живом раннере.

**Architecture:** Ядро `rsi_pullback/strategy/core` получает плоский блок полей `UseZoneEntry`/`Zone*` (дефолты 0); `enter()` сначала проверяет вход pullback, затем — если он молчит и zone включён — zone-вход; стоп, цель и трейл общие. Живой раннер, конфиг, env и `pullparity` возвращаются к состоянию коммита 7d20d99 (до общего счёта), остальной код rsi_zone удаляется.

**Tech Stack:** Go 1.25, `mage` (`./bin/mage ci`), `golangci-lint` v2, `mockery` v2.

**Spec:** `docs/superpowers/specs/2026-09-30-rsi-pullback-zone-entry-design.md`

## Global Constraints

- Ветка `feat/pullback-zone-entry` (уже создана от main f369b65, спека закоммичена).
- При `UseZoneEntry != 1` сигналы ядра обязаны совпадать с текущими бит-в-бит — включая текст `EntryReason` pullback-входа.
- Все четыре zone-поля в `DefaultParams()` — 0. Ни один тикерный пакет `rsi_pullback/strategy/<ticker>/` в этой работе не меняется.
- Гейт дня (`UseDayATRGate`) и объёмный гейт (`UseVolume`) на zone-вход не действуют.
- `EntryReason` zone-входа начинается с `zone:`.
- Откат раннера — новым коммитом (`git checkout 7d20d99 -- <files>`), без переписывания истории main.
- Комментарии в коде — в стиле окружающего файла (в `core.go` — английские; в тестах и доках допустим русский, как в соседних файлах). Коммиты — на русском, как в истории репозитория, с трейлером `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- `go build ./...` падает на `magefiles` — собирать `go build ./internal/... ./pkg/... ./cmd/...`.
- Сеть для прогонов не использовать без разрешения владельца: `pullparity` работает по кэшу `data/candles`.

## Review Focus

1. **Zone-поля заданы, но `UseZoneEntry=0`** — поведение ровно как без них (включая `Lookback`). Тест: `TestZoneOffLeavesSignalsUntouched` (Task 1).
2. **`UseZoneEntry=1` с забытым нулевым полем** — zone молча не входит, а не входит с мусором; тикер с таким литералом ловит сторож реестра. Тесты: `TestZoneEntryGates` (случаи с нулями, Task 1), `TestRSIPullbackZoneEntryFieldsArmedWhenEnabled` (Task 2).
3. **Совпадение обоих входов на одном баре** — одна покупка, причина pullback. Тест: `TestPullbackWinsWhenBothEntriesFire` (Task 1).
4. **Zone-вход при огромном стопе** (`entry − StopDailyATR·ATR ≤ 0`) — отказ, как у pullback, а не голая позиция. Тест: случай `stop at or below zero` в `TestZoneEntryGates` (Task 1).
5. **Сигнал ATR доходит до позиции через движок** — стоп zone-позиции строится от замороженного ATR входа. Тест: `TestEngineRunsAZoneTradeEndToEnd` (Task 2).

---

### Task 1: Zone-вход в ядре rsi_pullback

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (doc пакета строки 1–10; `Params` 30–49; `Lookback` 99–106; `enter`/`entryReason` 373–461; `Explain` 546–622)
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/core/zone_test.go`

**Interfaces:**
- Consumes: существующие тестовые хелперы пакета `core` из `core_test.go` — `entryParams()`, `entryFixture()`, `downtrendFixture()`, `upperCrossFixture()`, `withDay(md, atrWidth, todayHigh, todayLow)`, `withPosition(md, entryPrice, stop, heldBars)`, `shiftTo(&md, last)`, `msk`.
- Produces: поля `Params.UseZoneEntry int`, `Params.ZoneRSIPeriod int`, `Params.ZoneRSILower float64`, `Params.ZoneEMAPeriod int`; неэкспортируемые `(*Strategy).pullbackEntry`, `(*Strategy).zoneEntry`, `(*Strategy).entryLevels(entry, atr float64) (stop, target float64, ok bool)`, `(*Strategy).exitPlan(stop, target float64) string`. Task 2 использует поля.

- [ ] **Step 1: Написать падающие тесты**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/core/zone_test.go`:

```go
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
		name     string
		tweak    func(p *Params, md *strategy.MarketData)
		noDaily  bool
	}{
		{"UseZoneEntry=2 — не 1, значит выключен", func(p *Params, _ *strategy.MarketData) { p.UseZoneEntry = 2 }, false},
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
		"вход pullback":          withDay(entryFixture(), 10.0, 101, 100),
		"мёртвая зона дня":       withDay(entryFixture(), 10.0, 105, 100),
		"нисходящий тренд":       withDay(downtrendFixture(), 10.0, 101, 100),
		"открытая позиция, RSI":  withPosition(exit, exit.Closes[last]*0.97, 0, 2),
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
```

- [ ] **Step 2: Запустить — тесты не компилируются**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run 'Zone|PullbackWins' -count=1`
Expected: FAIL — `p.UseZoneEntry undefined (type Params has no field or method UseZoneEntry)`.

- [ ] **Step 3: Поля `Params`**

В `core.go` в конец структуры `Params` (после `TrailDailyATR`) добавить блок:

```go
	// --- zone entry: the second, per-ticker buy variant. Exits stay shared (the block above). ---
	UseZoneEntry  int     // 1 arms the zone entry; any other value disables it (grid; default 0)
	ZoneRSIPeriod int     // zone RSI length (grid; default 0 — set explicitly when armed)
	ZoneRSILower  float64 // a DOWNWARD cross of this band is the zone signal (grid; default 0)
	ZoneEMAPeriod int     // the zone entry needs close > EMA(ZoneEMAPeriod) (grid; default 0)
```

`DefaultParams()` не трогать: нулевые значения — это и есть дефолт (спека §1, почему).

- [ ] **Step 4: `Lookback`**

Заменить тело `Lookback()` (строки 100–105):

```go
func (s *Strategy) Lookback() int {
	need := max(s.p.EMASlow, s.p.EMAFast, s.p.RSIPeriod)
	if s.p.UseZoneEntry == 1 {
		need = max(need, s.p.ZoneEMAPeriod, s.p.ZoneRSIPeriod)
	}
	vol := 0
	if s.p.UseVolume == 1 && s.p.VolBaseDays > 0 {
		vol = (s.p.VolBaseDays + 1) * maxBarsPerDay * 7 / 5
	}
	return max(minLookback, 2*need+20, vol)
}
```

и дописать в doc-комментарий `Lookback` последнее предложение: `The zone entry's periods count only while it is armed: a disabled zone block must not grow the window.`

- [ ] **Step 5: `enter` → `pullbackEntry` + `zoneEntry`**

Заменить функции `enter` и `entryReason` (строки 373–461) целиком на:

```go
// enter checks the weekday, then the pullback entry, then — when armed and pullback stayed
// silent — the zone entry. Both variants share the stop, the target and the trail; only the
// entry conditions differ, and the position does not remember which one opened it. When both
// fire on the same bar the pullback entry wins: its check runs first, and the levels are the
// same either way. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. weekday: any time of a trading day will do, weekends will not.
	if !s.tradingDay(s.barTime(md)) {
		return sig
	}
	if got := s.pullbackEntry(md, sig); got.Kind == model.SignalBuy || s.p.UseZoneEntry != 1 {
		return got
	}
	return s.zoneEntry(md, sig)
}

// pullbackEntry emits a long when a short RSI crosses DOWN through its lower band on the current
// bar while the fast EMA sits above the slow one, the day is either fresh or spent, and the tape
// is busy. The caller has already checked the bar count and the weekday.
func (s *Strategy) pullbackEntry(md strategy.MarketData, sig model.Signal) model.Signal {
	i := len(md.Closes) - 1
	// 2. RSI crosses down through the lower band on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != len(md.Closes) || !crossedDown(rsi, i, s.p.RSILower) {
		return sig
	}
	// 3. trend confirmation: fast EMA above slow EMA (both warmed).
	fast, slow, ok := s.emaPair(md.Closes)
	if !ok || fast[i] <= slow[i] {
		return sig
	}
	// 4. the daily ATR is the unit of both the stop and the target: no ATR, no trade.
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	// 5. the day must be either fresh or spent.
	if !s.dayStateOK(md, atr) {
		return sig
	}
	// 6. the tape must be busier than usual for this time of day.
	if !s.volumeOK(md) {
		return sig
	}
	entry := md.Closes[i]
	stop, target, ok := s.entryLevels(entry, atr)
	if !ok {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], fast[i], slow[i], entry, stop, target, atr, md)
	return sig
}

// zoneEntry emits a long when the zone RSI crosses DOWN through ZoneRSILower on the current bar
// while the close sits strictly above a warmed EMA(ZoneEMAPeriod). The day gate and the volume
// gate do not apply: the zone entry has its own set of conditions. Any zone field at or below
// zero refuses the entry — a forgotten field in a ticker literal must not open trades on a
// zero-length RSI or EMA. The caller has already checked the bar count and the weekday.
//
// The cross uses the same crossedDown helper as the pullback entry. Its value guard
// (series[i-1] > 0) is inert here: with ZoneRSILower > 0 a previous reading of 0 — warm-up or a
// genuine RSI of 0.00 — is never >= the band, so an index-based warm-up check would give the same
// answer.
func (s *Strategy) zoneEntry(md strategy.MarketData, sig model.Signal) model.Signal {
	if s.p.ZoneRSIPeriod <= 0 || s.p.ZoneEMAPeriod <= 0 || s.p.ZoneRSILower <= 0 {
		return sig
	}
	n := len(md.Closes)
	i := n - 1
	rsi := indicators.RSISeries(md.Closes, s.p.ZoneRSIPeriod)
	if len(rsi) != n || !crossedDown(rsi, i, s.p.ZoneRSILower) {
		return sig
	}
	trend := ema.Compute(md.Closes, s.p.ZoneEMAPeriod)
	if len(trend) != n || trend[i] <= 0 || md.Closes[i] <= trend[i] {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	entry := md.Closes[i]
	stop, target, ok := s.entryLevels(entry, atr)
	if !ok {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = fmt.Sprintf(
		"zone: RSI(%d) ушёл под %.0f (%.1f), close %.4f > EMA(%d) %.4f, дневной ATR %.4f; вход %.4f, %s",
		s.p.ZoneRSIPeriod, s.p.ZoneRSILower, rsi[i], entry, s.p.ZoneEMAPeriod, trend[i], atr, entry,
		s.exitPlan(stop, target),
	)
	return sig
}

// entryLevels freezes the stop and the target of a new position off the daily ATR. ok is false
// when an armed stop lands at or below zero: that is not a floor, it is a naked long. manage()
// rebuilds the protective level from entry and EntryATR via DesiredStop, and only ever acts on it
// when it comes back > 0, so a non-positive stop here would silently hold the position with no
// protective exit at all — TP and RSI (when armed) are not a substitute, and RSI can be disabled
// outright via UseRSIExit. The entry must be refused instead.
func (s *Strategy) entryLevels(entry, atr float64) (stop, target float64, ok bool) {
	if s.p.StopDailyATR > 0 {
		stop = entry - s.p.StopDailyATR*atr
		if stop <= 0 {
			return 0, 0, false
		}
	}
	if s.p.TPDailyATR > 0 {
		target = entry + s.p.TPDailyATR*atr
	}
	return stop, target, true
}

// entryReason renders the human-readable rationale of a pullback entry shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, fastNow, slowNow, entry, stop, target, atr float64, md strategy.MarketData) string {
	dayHow := "гейт дня выключен"
	if s.p.UseDayATRGate == 1 && md.TodayHigh > 0 && md.TodayLow > 0 && md.TodayHigh >= md.TodayLow && atr > 0 {
		dayHow = fmt.Sprintf("день прошёл %.2f ATR", (md.TodayHigh-md.TodayLow)/atr)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f) на откате, EMA(%d) %.4f > EMA(%d) %.4f, %s (дневной ATR %.4f); вход %.4f, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, s.p.EMAFast, fastNow, s.p.EMASlow, slowNow,
		dayHow, atr, entry, s.exitPlan(stop, target),
	)
}

// exitPlan renders the stop, the trail and the target of a new position — shared by both entry
// variants, because the exits are.
func (s *Strategy) exitPlan(stop, target float64) string {
	stopHow := "стоп выключен"
	if stop > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (−%.2f ATR)", stop, s.p.StopDailyATR)
	}
	if s.p.UseTrail == 1 && s.p.TrailDailyATR > 0 {
		// Трейл считается от PrevMaxFavorablePrice и уже с первого бара может стоять теснее
		// фиксированного стопа выше — тогда именно он, а не stopHow, был бы реальной защитой
		// сделки. Печатаем его тут же, чтобы запись в журнале не называла один уровень (SL),
		// пока сделку с первого бара мог связывать другой (TRAIL).
		stopHow += fmt.Sprintf(", трейл −%.2f ATR от максимума (с первого бара)", s.p.TrailDailyATR)
	}
	tpHow := "цель выключена"
	if target > 0 {
		tpHow = fmt.Sprintf("цель %.4f (+%.2f ATR)", target, s.p.TPDailyATR)
	}
	return stopHow + ", " + tpHow
}
```

Текст pullback-причины остаётся прежним символ-в-символ: раньше формат заканчивался на `вход %.4f, %s, %s` со `stopHow, tpHow`, теперь — `вход %.4f, %s` с `stopHow + ", " + tpHow`.

- [ ] **Step 6: `Explain`**

В `Explain()` сразу после блока «фон объёмов» (после `if s.p.UseVolume != 1 { ... } else { ... }`) вставить:

```go
	if s.p.UseZoneEntry != 1 {
		sb.WriteString("zone-вход: выключен (UseZoneEntry=0)\n")
	} else {
		s.explainZone(&sb, md)
	}
```

и добавить функцию после `Explain`:

```go
// explainZone reports the zone entry's own gates. The day gate and the volume gate above do not
// apply to it; the stop, the target and the trail below are shared.
func (s *Strategy) explainZone(sb *strings.Builder, md strategy.MarketData) {
	if s.p.ZoneRSIPeriod <= 0 || s.p.ZoneEMAPeriod <= 0 || s.p.ZoneRSILower <= 0 {
		sb.WriteString("zone-вход: поля не заданы (ZoneRSIPeriod/ZoneRSILower/ZoneEMAPeriod ≤ 0) — вход невозможен\n")
		return
	}
	n := len(md.Closes)
	i := n - 1
	if rsi := indicators.RSISeries(md.Closes, s.p.ZoneRSIPeriod); len(rsi) == n {
		fmt.Fprintf(sb, "zone-вход: RSI(%d) пред %.1f тек %.1f; крест вниз через %.0f? %v\n",
			s.p.ZoneRSIPeriod, rsi[i-1], rsi[i], s.p.ZoneRSILower, crossedDown(rsi, i, s.p.ZoneRSILower))
	} else {
		sb.WriteString("zone-вход: RSI — недостаточно истории\n")
	}
	if trend := ema.Compute(md.Closes, s.p.ZoneEMAPeriod); len(trend) == n && trend[i] > 0 {
		fmt.Fprintf(sb, "zone-вход: close %.4f vs EMA(%d) %.4f: выше? %v (гейты дня и объёма не действуют)\n",
			md.Closes[i], s.p.ZoneEMAPeriod, trend[i], md.Closes[i] > trend[i])
	} else {
		fmt.Fprintf(sb, "zone-вход: EMA(%d) не прогрета\n", s.p.ZoneEMAPeriod)
	}
}
```

- [ ] **Step 7: Doc пакета**

В doc-комментарии пакета (строки 1–10) после первого предложения («…on the current bar.») вставить:

```go
// An optional second entry — the zone entry, armed per ticker with UseZoneEntry=1 — buys a
// short RSI crossing DOWN through ZoneRSILower while the close sits above EMA(ZoneEMAPeriod); the
// day and volume gates do not apply to it. Both entries share the stop, the target, the trail
// and the RSI exit below.
```

- [ ] **Step 8: Запустить тесты пакета**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/... -count=1`
Expected: PASS — новые тесты и все старые (включая снимки 40 тикерных пакетов: их литералы не менялись, zone-поля нулевые).

Если падает `TestZoneEntryGates/close ниже EMA` или случай с EMA 200 — фикстура не того вида, что заявлено в комментарии; напечатать `ema.Compute(entryFixture().Closes, N)[404]` и `Closes[404]` и поправить период в тесте, а не код.

- [ ] **Step 9: Мутационная проверка**

По одной, с откатом после каждой (`git stash` не нужен — правка руками и `git checkout -- core.go` после):
1. В `enter` заменить `s.p.UseZoneEntry != 1` на `false` → должен упасть `TestZoneOffLeavesSignalsUntouched` или `TestZoneEntryBuysWhenPullbackIsBlocked` (часть «zone выключен»).
2. В `zoneEntry` удалить `md.Closes[i] <= trend[i] ||` → падает `TestZoneEntryGates/close ниже EMA`.
3. В `zoneEntry` удалить проверку `s.p.ZoneRSILower <= 0 ||` → падает `TestZoneEntryGates/ZoneRSILower=0`.
4. В `zoneEntry` вставить в начало `if !s.dayStateOK(md, s.dailyATR(md)) { return sig }` → падает случай «мёртвая зона гейта дня».
5. В `Lookback` убрать условие `if s.p.UseZoneEntry == 1` (оставить тело) → падает `TestLookbackCountsZoneOnlyWhenArmed`.
6. В `enter` поменять порядок: сначала `zoneEntry`, потом `pullbackEntry` → падает `TestPullbackWinsWhenBothEntriesFire`.

Каждая мутация должна уронить хотя бы указанный тест. Если какая-то не роняет — усилить тест.

- [ ] **Step 10: Lint и коммит**

Run: `gofmt -w internal/service/trading_strategy/rsi_pullback/strategy/core/ && ./bin/mage lint`
Expected: без замечаний.

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/core.go internal/service/trading_strategy/rsi_pullback/strategy/core/zone_test.go
git commit -m "feat(rsi_pullback): второй вход zone — крест RSI вниз и close выше EMA, выходы общие

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Сквозной тест через движок, сторож реестра и документация механики

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/core/engine_zone_test.go`
- Modify: `internal/service/backtest/rsi_pullback_registry_test.go` (после `TestRSIPullbackTickersKeepTheRSIExitArmed`, ~строка 181)
- Modify: `docs/rsi_pullback/strategy.md` (§2 таблица, строки 38–57; конец §3, после строки 109)

**Interfaces:**
- Consumes: поля `UseZoneEntry`, `ZoneRSIPeriod`, `ZoneRSILower`, `ZoneEMAPeriod` из Task 1; `bt.Run(s strategy.Strategy, candles, dailyCandles, htfCandles []bt.Candle, cfg bt.Config) bt.Result` (`internal/domain/backtest/engine.go:218`); `bt.Trade.EntryReason`; хелпер `dailyBars` и `msk` из `core_test.go`; `rsiPullbackRegistry` в пакете `backtest`.
- Produces: ничего нового для кода.

- [ ] **Step 1: Сквозной тест через движок**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/core/engine_zone_test.go`:

```go
// engine_zone_test.go прогоняет zone-сделку через настоящий движок бэктеста (backtest.Run), а не
// через Decide напрямую: так проверяется передача sig.ATR в Position.EntryATR, от которого живёт
// стоп позиции, и то, что причина «zone:» доезжает до журнала сделок.
package core

import (
	"math"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

func TestEngineRunsAZoneTradeEndToEnd(t *testing.T) {
	p := DefaultParams()
	p.EMAFast = p.EMASlow // гейт тренда pullback закрыт: войти может только zone
	p.UseZoneEntry, p.ZoneRSIPeriod, p.ZoneRSILower, p.ZoneEMAPeriod = 1, 4, 25, 200
	s := NewWithParams("TEST", p)
	lookback := s.Lookback()
	if lookback != 420 {
		t.Fatalf("Lookback() = %d, want 420 (тест рассчитан на ZoneEMAPeriod 200)", lookback)
	}

	const width = 10.0                                   // дневной ATR будних дневок
	exitBar := time.Date(2026, 6, 1, 12, 0, 0, 0, msk)   // понедельник
	entryBar := exitBar.Add(-30 * time.Minute)

	// Ровный рост +0.08% за бар и три бара по −0.2%: RSI(4) пересекает 25 вниз ровно на третьем
	// (та же форма, на которой проверялось удалённое ядро rsi_zone), close далеко над EMA(200).
	closes := make([]float64, 0, lookback+3)
	price := 100.0
	for i := 0; i < lookback; i++ {
		price *= 1.0008
		closes = append(closes, price)
	}
	for i := 0; i < 3; i++ {
		price *= 0.998
		closes = append(closes, price)
	}
	entryPrice := closes[len(closes)-1]

	start := entryBar.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	candles := make([]bt.Candle, 0, len(closes)+1)
	for i, c := range closes {
		candles = append(candles, bt.Candle{
			Time: start.Add(time.Duration(i) * 30 * time.Minute),
			Open: c, High: c * 1.003, Low: c * 0.997, Close: c,
		})
	}
	// Бар выхода — гэп вниз под уровень стопа: движок обязан исполнить по min(уровень, open).
	level := entryPrice - p.StopDailyATR*width
	gapOpen := entryPrice - 3*width
	candles = append(candles, bt.Candle{
		Time: exitBar, Open: gapOpen, High: gapOpen, Low: gapOpen - 1, Close: gapOpen - 0.5,
	})

	h, l, c, ts := dailyBars(exitBar, 40, width, width/10)
	daily := make([]bt.Candle, len(c))
	for i := range c {
		daily[i] = bt.Candle{Time: ts[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}

	res := bt.Run(s, candles, daily, nil, bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1 (zone-вход на кресте, SL на гэпе)", len(res.Trades))
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-entryPrice) > 1e-6 {
		t.Fatalf("EntryPrice = %v, want %v: вход не на задуманном баре", tr.EntryPrice, entryPrice)
	}
	if !strings.HasPrefix(tr.EntryReason, "zone:") {
		t.Fatalf("EntryReason = %q, want prefix zone:", tr.EntryReason)
	}
	if math.Abs(tr.ATR-width) > 1e-9 {
		t.Fatalf("ATR сделки = %v, want %v: ATR сигнала должен дойти до Position.EntryATR", tr.ATR, width)
	}
	if tr.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", tr.Reason)
	}
	if want := math.Min(level, gapOpen); math.Abs(tr.ExitPrice-want) > 1e-6 {
		t.Fatalf("ExitPrice = %v, want %v = min(уровень %v, open гэпа %v)", tr.ExitPrice, want, level, gapOpen)
	}
}
```

- [ ] **Step 2: Запустить**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run TestEngineRunsAZoneTradeEndToEnd -count=1 -v`
Expected: PASS. Мутация: в `zoneEntry` заменить `sig.ATR = atr` на `sig.ATR = 0` → тест падает (нет стопа — нет SL-выхода или ATR сделки 0). Вернуть.

- [ ] **Step 3: Сторож реестра**

В `internal/service/backtest/rsi_pullback_registry_test.go` сразу после `TestRSIPullbackTickersKeepTheRSIExitArmed` добавить:

```go
// TestRSIPullbackZoneEntryFieldsArmedWhenEnabled сторожит ловушку нулевого значения во втором
// входе: zone-поля в ядре по умолчанию нулевые, и тикер, включивший UseZoneEntry=1 литералом, но
// забывший поле, молча не торговал бы zone вовсе (ядро отказывает входу при нулевом поле). Пока
// zone не включён ни у одного тикера, тест проходит пусто — он для будущих калибровок.
func TestRSIPullbackZoneEntryFieldsArmedWhenEnabled(t *testing.T) {
	for ticker, b := range rsiPullbackRegistry {
		p, ok := b.DefaultParams().(core.Params)
		if !ok {
			t.Fatalf("%s: DefaultParams вернул %T, want core.Params", ticker, b.DefaultParams())
		}
		if p.UseZoneEntry != 1 {
			continue
		}
		if p.ZoneRSIPeriod <= 0 || p.ZoneRSILower <= 0 || p.ZoneEMAPeriod <= 0 {
			t.Errorf("%s: UseZoneEntry=1, но ZoneRSIPeriod=%d ZoneRSILower=%v ZoneEMAPeriod=%d — поле забыто в литерале",
				ticker, p.ZoneRSIPeriod, p.ZoneRSILower, p.ZoneEMAPeriod)
		}
	}
}
```

Run: `go test ./internal/service/backtest/ -run TestRSIPullback -count=1`
Expected: PASS.

- [ ] **Step 4: Документация механики**

В `docs/rsi_pullback/strategy.md`:

(а) В таблицу §2 после строки `TrailDailyATR` добавить:

```markdown
| `UseZoneEntry` | 0/1 | 0 | да (zone: 0, 1) |
| `ZoneRSIPeriod` | длина RSI, баров | 0 (задаётся при включении) | да (zone) |
| `ZoneRSILower` | пункты RSI | 0 (задаётся при включении) | да (zone) |
| `ZoneEMAPeriod` | баров | 0 (задаётся при включении) | да (zone) |
```

(б) В конец §3 (после абзаца «Оба уходят в сигнал… `sig.ATR` получает дневной, а не внутридневной ATR.») добавить подраздел:

```markdown
### 3.1. Второй вход (zone)

Необязательный второй вариант покупки, включается по тикеру: `UseZoneEntry = 1`. Он
проверяется, только если вход pullback на этом баре сигнала не дал. Условия — свои:

1. **Будний день** — тот же гейт 1.
2. **RSI-крест вниз** своего RSI: `RSI(ZoneRSIPeriod)` пересекает `ZoneRSILower` сверху вниз на
   текущем баре (тот же помощник `crossedDown`, событие, а не состояние).
3. **Цена над EMA:** close текущего бара строго выше прогретой `EMA(ZoneEMAPeriod)`.
4. **Дневной ATR готов** — тот же гейт 4.

Гейт состояния дня (5) и фон объёмов (6) на zone-вход **не действуют**. Стоп и цель
замораживаются теми же формулами и полями (`StopDailyATR`, `TPDailyATR`, `DailyATRPeriod`),
трейл и RSI-выход — общие (§4): позиция не помнит, каким входом открыта. Если на баре
срабатывают оба входа, покупка одна — по pullback. `EntryReason` zone-входа начинается с
`zone:`.

Все четыре поля по умолчанию нулевые — ноль у любого из трёх числовых полей отключает zone-вход
даже при `UseZoneEntry = 1`, поэтому значения задаются явно: в сетке калибровки или в литерале
тикера (сторож — `TestRSIPullbackZoneEntryFieldsArmedWhenEnabled`). `Lookback` учитывает
zone-периоды только при включённом входе.
```

(в) В §7 (`Lookback`) дописать одно предложение в конец раздела: `При UseZoneEntry = 1 в максимум периодов входят ещё ZoneEMAPeriod и ZoneRSIPeriod.`

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/engine_zone_test.go internal/service/backtest/rsi_pullback_registry_test.go docs/rsi_pullback/strategy.md
git commit -m "test(rsi_pullback): zone-сделка через движок, сторож zone-полей; docs: второй вход

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Живой раннер снова ведёт одну стратегию — откат к 7d20d99

Коммиты a9b20f4…f369b65 добавили общий счёт для rsi_zone и больше ничего в перечисленных ниже файлах не меняли (проверено `git diff --name-only 7d20d99 HEAD`). Файлы возвращаются к 7d20d99 целиком. Пакет `rsi_zone/live` удаляется здесь же: он импортирует удаляемый `livecore/adapter`.

**Files:**
- Restore from 7d20d99:
  - `internal/service/trading_strategy/rsi_pullback/live/live.go`
  - `internal/service/trading_strategy/rsi_pullback/live/pass.go`
  - `internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct.go`
  - `internal/service/trading_strategy/livecore/statestore/statestore.go`
  - `internal/service/trading_strategy/livecore/statestore/statestore_test.go`
  - `cmd/pullparity/main.go`, `cmd/pullparity/main_test.go`
  - `internal/service_provider/client.go`, `internal/service_provider/service.go`
  - `internal/app/init_config.go`
  - `internal/config/config.go`, `internal/config/telegram_client.go`
  - `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
  - `.golangci.yml`
  - `docs/rsi_pullback/live.md`
- Delete:
  - `internal/service/trading_strategy/rsi_pullback/live/adapter.go`
  - `internal/service/trading_strategy/rsi_pullback/live/shared_test.go`
  - `internal/service/trading_strategy/livecore/adapter/` (весь каталог)
  - `internal/service/trading_strategy/livecore/rebuild/` (весь каталог)
  - `internal/service/trading_strategy/rsi_zone/live/` (весь каталог)
  - `internal/config/rsi_zone.go`, `internal/config/rsi_zone_test.go`

**Interfaces:**
- Consumes: ничего из Task 1–2 (раннер зовёт `core.Decide`/`core.DesiredStop`, их сигнатуры не менялись).
- Produces: раннер без `adapter.Strategy`; `statestore.Entry` без поля `Strategy`; `pullparity` без флага `-strategy`.

- [ ] **Step 1: Убедиться, что после 7d20d99 в этих файлах нет посторонних правок**

Run:
```bash
git log --oneline 7d20d99..HEAD -- internal/service/trading_strategy/rsi_pullback/live internal/service/trading_strategy/livecore cmd/pullparity internal/service_provider internal/app/init_config.go internal/config env .golangci.yml docs/rsi_pullback/live.md
```
Expected: только коммиты этой ветки zone-live (a9b20f4, 73cb856, 70ca05d, 96aca01, 0b10515, f4fc137, fa21321, b949d40, 8787ff1, 88ed005, f369b65) — и никаких коммитов Task 1–2 (они эти файлы не трогают). Если в списке есть посторонний коммит — остановиться и спросить владельца.

- [ ] **Step 2: Восстановить и удалить**

```bash
git checkout 7d20d99 -- \
  internal/service/trading_strategy/rsi_pullback/live/live.go \
  internal/service/trading_strategy/rsi_pullback/live/pass.go \
  internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct.go \
  internal/service/trading_strategy/livecore/statestore/statestore.go \
  internal/service/trading_strategy/livecore/statestore/statestore_test.go \
  cmd/pullparity/main.go cmd/pullparity/main_test.go \
  internal/service_provider/client.go internal/service_provider/service.go \
  internal/app/init_config.go \
  internal/config/config.go internal/config/telegram_client.go \
  env/prod.env env/prod.env.example env/local.env.example \
  .golangci.yml docs/rsi_pullback/live.md
git rm -q -r \
  internal/service/trading_strategy/rsi_pullback/live/adapter.go \
  internal/service/trading_strategy/rsi_pullback/live/shared_test.go \
  internal/service/trading_strategy/livecore/adapter \
  internal/service/trading_strategy/livecore/rebuild \
  internal/service/trading_strategy/rsi_zone/live \
  internal/config/rsi_zone.go internal/config/rsi_zone_test.go
```

- [ ] **Step 3: Сборка и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/service/trading_strategy/... ./internal/config/... ./internal/service_provider/... ./cmd/pullparity/... -count=1`
Expected: сборка без ошибок, тесты PASS. (Бэктест-код rsi_zone — `rsi_zone/strategy`, `backtest/rsi_zone_*`, `cmd/zonescreen` — ещё на месте и собирается: он от раннера не зависит; удаляется в Task 4.)

Если сборка падает из-за кода вне перечисленных файлов — правка минимальная, в сообщении коммита отдельной строкой «правка вне отката: …».

- [ ] **Step 4: Проверить, что общего счёта не осталось**

Run: `git grep -n -i "adapter\.\|LegacyOwner\|RSI_ZONE\|TopicRSIZone\|rsi_zone/live" -- ':!docs/superpowers'`
Expected: пусто.

- [ ] **Step 5: Моки и lint**

Run: `./bin/mage mockscheck && ./bin/mage lint`
Expected: без дрейфа и замечаний. Если `mocksCheck` показывает дрейф — `./bin/mage mocks`, добавить результат в коммит.

- [ ] **Step 6: Коммит**

```bash
git add -A internal/service/trading_strategy internal/config internal/service_provider internal/app cmd/pullparity env .golangci.yml docs/rsi_pullback/live.md
git commit -m "revert(rsi_pullback): раннер счёта снова ведёт одну стратегию — откат общего счёта к 7d20d99

Второй вход zone живёт в ядре rsi_pullback, гость раннера больше не нужен.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Удалить бэктест-стратегию rsi_zone, скринер, данные, доки и команду калибровки

**Files:**
- Delete:
  - `internal/service/trading_strategy/rsi_zone/` (остаток: `strategy/`)
  - `internal/service/backtest/rsi_zone_registry.go`, `rsi_zone_registry_test.go`, `rsi_zone_grid_test.go`, `rsi_zone_baza_grid_test.go`, `rsi_zone_dias_grid_test.go`, `rsi_zone_lent_grid_test.go`
  - `internal/service/backtest/zone_screen.go`, `zone_screen_report.go`, `zone_screen_report_test.go`, `zone_screen_test.go`
  - `cmd/zonescreen/`
  - `data/params/rsi_zone/`
  - `docs/rsi_zone/`
  - `.claude/commands/rsi-zone-calibrate.md`
- Modify:
  - `cmd/backtest/main.go:41` (help флага `-strategy`), `:168-171` (`case "rsi_zone"` и текст ошибки)
  - `internal/service/backtest/screenrun/screenrun.go:2` (doc-комментарий упоминает cmd/zonescreen)
  - `CLAUDE.md` (строка Layout про `trading_strategy/`)

**Interfaces:**
- Consumes: —
- Produces: `-strategy rsi_zone` в `cmd/backtest` больше нет; `screenrun` остаётся для `cmd/pullscreen`.

- [ ] **Step 1: Удалить файлы**

```bash
git rm -q -r \
  internal/service/trading_strategy/rsi_zone \
  internal/service/backtest/rsi_zone_registry.go internal/service/backtest/rsi_zone_registry_test.go \
  internal/service/backtest/rsi_zone_grid_test.go internal/service/backtest/rsi_zone_baza_grid_test.go \
  internal/service/backtest/rsi_zone_dias_grid_test.go internal/service/backtest/rsi_zone_lent_grid_test.go \
  internal/service/backtest/zone_screen.go internal/service/backtest/zone_screen_report.go \
  internal/service/backtest/zone_screen_report_test.go internal/service/backtest/zone_screen_test.go \
  cmd/zonescreen data/params/rsi_zone docs/rsi_zone .claude/commands/rsi-zone-calibrate.md
```

- [ ] **Step 2: `cmd/backtest/main.go`**

Строка 41 — убрать `|rsi_zone` из help:

```go
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|reversion|scalping_rsimacd|rsi_ema|vwap_rev|rsi_pullback")
```

Строки 168–169 — удалить ветку:

```go
	case "rsi_zone":
		binding = svc.RSIZoneLookupOrGeneric(ticker)
```

Строка 171 (текст ошибки) — убрать `|rsi_zone`:

```go
		return fmt.Errorf("unknown strategy %q (want scalping|reversion|scalping_rsimacd|rsi_ema|vwap_rev|rsi_pullback)", strategyName)
```

- [ ] **Step 3: `screenrun.go` doc**

В строке 2 `internal/service/backtest/screenrun/screenrun.go` заменить `(cmd/pullscreen, cmd/zonescreen)` на `(cmd/pullscreen)`. Если после правки фраза грамматически требует единственного числа (например, «shared by the screeners») — поправить на «used by the screener».

- [ ] **Step 4: `CLAUDE.md`**

В строке Layout `trading_strategy/` (строка 23) удалить предложение целиком, начиная с `` `rsi_zone` — 30m long multi-day RSI critical-zone strategy`` и до `` grids in `data/params/rsi_zone/<ticker>/`.`` включительно. В том же абзаце после `сверка живой сборки с бэктестом — cmd/pullparity.` вставить:

```
Второй, включаемый по тикеру вход zone (`UseZoneEntry`: крест RSI вниз + close > EMA, гейты дня и объёма не действуют, выходы общие) — docs/rsi_pullback/strategy.md §3.1; отдельная стратегия rsi_zone удалена 2026-09-30.
```

- [ ] **Step 5: Сборка, неиспользуемый код, остатки**

Run: `go build ./internal/... ./pkg/... ./cmd/... && ./bin/mage lint`
Expected: сборка без ошибок. Если lint (`unused`) находит в `internal/service/backtest` неэкспортируемые функции, которыми пользовался только удалённый zone-код, — удалить их (только те, что указал lint).

Run: `git grep -n -i "rsi_zone\|rsizone\|RSIZone\|zonescreen\|zone_screen" -- ':!docs/superpowers'`
Expected: пусто.

- [ ] **Step 6: Коммит**

```bash
git add -A
git commit -m "chore(rsi_zone): стратегия удалена — вход zone теперь второй вариант rsi_pullback

Удалены ядро и тикеры rsi_zone, бэктест-реестр, скринер cmd/zonescreen, сетки,
документация и команда /rsi-zone-calibrate. Спеки и планы остаются как история.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Итоговая проверка — CI и паритет живого с бэктестом

**Files:** нет изменений кода (только если проверка что-то найдёт).

- [ ] **Step 1: Полный CI**

Run: `./bin/mage ci`
Expected: lint, `go test -race ./...`, mocks-check — зелёные.

- [ ] **Step 2: Паритет по боевой вселенной**

Список тикеров — значение `RSI_PULLBACK_TICKERS` из `env/prod.env`:

```bash
TICKERS=$(grep -o 'RSI_PULLBACK_TICKERS=.*' env/prod.env | cut -d= -f2)
go run ./cmd/pullparity -tickers "$TICKERS" -months 24
```

Expected: по каждому тикеру ноль расхождений. `pullparity` читает кэш `data/candles`; если какому-то тикеру кэша не хватает и нужен сетевой догруз — остановиться и спросить владельца, а не лезть в сеть.

- [ ] **Step 3: Отчёт**

Сообщить владельцу: хэши коммитов Task 1–4, вывод `mage ci` (итоговая строка), итог `pullparity` по тикерам. Ветку не мержить и не пушить без команды владельца.
