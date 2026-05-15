# Golden X — Этап A (базовые исправления) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Закрыть очевидные баги Golden-X стратегии без изменения торговой логики: убрать шум в Telegram (дедупликация), считать RSI на закрытых неделях, типизировать режим Gold/Growth, логировать паники, эффективно итерироваться только по нужным бумагам, выправить косметику.

**Architecture:** Точечный рефакторинг существующего пакета `internal/service/trading_strategy/golden_x`. Новый enum `StrategyKind` живёт в `dto`-подпакете рядом с `Trade`. State дедупликации хранится in-memory как поле `service`, под `sync.RWMutex`; на каждый инстанс стратегии (Gold/Growth) — отдельный state. Закрытая свеча выбирается фильтрацией массива RSI по дате (граница — понедельник 00:00 MSK), а не индексом.

**Tech Stack:** Go 1.25, sqlx (не задействован в этом этапе), Tinkoff Invest gRPC API, Telegram Bot API, `pkg/logger`.

---

## Контекст: уже выполненные правки в текущей сессии

Эти изменения **уже на диске** и затрагивают часть Task 1. Не повторяйте — проверьте состояние:

- `internal/service/trading_strategy/golden_x/dto/strategy_kind.go` — создан, `StrategyKind` + `Medal()`.
- `internal/service/trading_strategy/golden_x/dto/trade.go` — поле `ShareTip int` заменено на `Kind StrategyKind`.
- `internal/service/trading_strategy/golden_x/notification/notifications.go` — сигнатура `Trade(info, kind)`, формирование шапки через `kind.Medal()`, опечатка «находящтеся» → «находящиеся», удалён пустой `if shareTip == 1 {}` блок.
- `internal/service/trading_strategy/golden_x/notification/rsi_by_shares.go` — сигнатура `RSIList(info, kind)`, медаль через `kind.Medal()`.
- `internal/app/app.go` — все три места создания `goldenx.Trade{…}` обновлены на `Kind: goldenx.StrategyKindDividend/Growth`.

**Не тронуто** (ломает сборку до Task 1 step 1): `internal/service/trading_strategy/golden_x/trade.go:75,84` всё ещё ссылается на `in.ShareTip`.

---

## File Structure

**Создаются:**

- `internal/service/trading_strategy/golden_x/dedup.go` — структура `alertState` (map shareID → tier) с методами `ShouldAlert(id, tier) bool` / `Reset(id)`. Один файл — одна ответственность (state дедупликации).
- `internal/service/trading_strategy/golden_x/dedup_test.go` — table-driven тест на переходы tier.
- `internal/service/trading_strategy/golden_x/candle.go` — функция `lastClosedWeeklyRSI(items, now) (*domain.RSIItemTechAnalyse, bool)` для выбора закрытой свечи. Чистая функция без зависимостей — легко тестируется.
- `internal/service/trading_strategy/golden_x/candle_test.go` — table-driven тест на выбор закрытой недели.

**Модифицируются:**

- `internal/service/trading_strategy/golden_x/types.go` — добавить поле `state *alertState` в `service` и инициализировать в `NewService`.
- `internal/service/trading_strategy/golden_x/trade.go` — основная переработка: itr по `ShareList.All()`, warmup `-80`, фильтр закрытой свечи через `lastClosedWeeklyRSI`, дедуп через `state.ShouldAlert`, лог в `recover`, замена `in.ShareTip` на `in.Kind`, чистка `shareRSI`/`ok == true`.

---

## Задачи

### Task 1: Допилить A4 в `trade.go` (минимум для сборки)

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/trade.go:75,84`

После этой задачи проект собирается. Дальнейшие задачи можно делать инкрементально.

- [ ] **Step 1.1: Заменить `in.ShareTip` на `in.Kind`**

В `trade.go:75` и `trade.go:84` заменить аргумент `in.ShareTip` на `in.Kind`:

```go
err := s.tgClient.SendMessage(notif.Trade(info, in.Kind))
```

```go
err := s.tgClient.SendMessage(notif.RSIList(RSIInfo, in.Kind))
```

- [ ] **Step 1.2: Проверить сборку**

```bash
cd /home/oleg/GolandProjects/tinvest && go build ./...
```

Expected: успешная сборка без ошибок.

- [ ] **Step 1.3: Коммит**

```bash
cd /home/oleg/GolandProjects/tinvest && git add internal/service/trading_strategy/golden_x internal/app/app.go && git commit -m "refactor(golden_x): replace ShareTip int with StrategyKind enum"
```

---

### Task 2: A5 — итерация по `ShareList.All()`

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/trade.go` (тело цикла, ~строки 22, 29–38)

Цель: убрать вызов `s.instrumentServiceGrpcClient.Shares(ctx)` и проход по всем акциям биржи. Имя бумаги брать из `Instrument.Name`.

- [ ] **Step 2.1: Заменить цикл**

В `trade.go` найти блок:

```go
t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
loc, _ := time.LoadLocation("Europe/Moscow")
dateNow := time.Now().In(loc)
var shareRSI collection.Instrument
info := domain.NewInfo()
RSIInfo := domain.NewInfo()

for _, share := range t {
    flag := false
    if shareItem, ok := in.ShareList.Get(share.ID); ok == true {
        flag = true
        shareRSI = shareItem
    }

    if !flag {
        continue
    }

    rsi, rsiErr := s.rsi.CalculateRSI(
        ctx,
        share.ID,
        in.Interval,
        utils.TimeStampPbGenerator(dateNow, -20, in.Interval),
        timestamppb.New(dateNow),
        int32(shareRSI.RSILength),
    )
```

И заменить на:

```go
loc, _ := time.LoadLocation("Europe/Moscow")
dateNow := time.Now().In(loc)
info := domain.NewInfo()
RSIInfo := domain.NewInfo()

for _, share := range in.ShareList.All() {
    rsi, rsiErr := s.rsi.CalculateRSI(
        ctx,
        share.ID,
        in.Interval,
        utils.TimeStampPbGenerator(dateNow, -20, in.Interval),
        timestamppb.New(dateNow),
        int32(share.RSILength),
    )
```

Дальше по телу цикла заменить упоминания `share.Name` и `shareRSI.RSILength` на `share.Name` и `share.RSILength` (теперь это один объект `collection.Instrument`).

- [ ] **Step 2.2: Убрать неиспользуемый импорт `collection`**

Если после правок пакет `tinvest/pkg/collection` перестал использоваться в `trade.go` — удалить из import-блока. Проверить: `goimports` или вручную.

- [ ] **Step 2.3: Убрать неиспользуемый импорт `instrumentServiceGrpcClient`**

Поле `instrumentServiceGrpcClient` в `service` остаётся (может пригодиться позже), но если линтер ругается на «unused», пометить `_ = s.instrumentServiceGrpcClient` или удалить. **Решение:** оставить поле — оно часть конструктора `NewService`, удаление сломает DI в `service_provider`.

- [ ] **Step 2.4: Сборка**

```bash
cd /home/oleg/GolandProjects/tinvest && go build ./...
```

Expected: успешная сборка.

- [ ] **Step 2.5: Коммит**

```bash
cd /home/oleg/GolandProjects/tinvest && git add internal/service/trading_strategy/golden_x/trade.go && git commit -m "refactor(golden_x): iterate over ShareList.All() instead of all exchange shares"
```

---

### Task 3: A2 — фильтр RSI по закрытой недельной свече (с тестом)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/candle.go`
- Create: `internal/service/trading_strategy/golden_x/candle_test.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go` (использование функции)

- [ ] **Step 3.1: Написать тест**

Создать `internal/service/trading_strategy/golden_x/candle_test.go`:

```go
package golden_x

import (
	"testing"
	"time"

	"tinvest/internal/domain"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %s: %v", name, err)
	}
	return loc
}

func TestLastClosedWeeklyRSI(t *testing.T) {
	msk := mustLoad(t, "Europe/Moscow")

	// Текущая неделя: 2026-05-11 (Пн) 00:00 MSK — 2026-05-18 (Пн) 00:00 MSK.
	// "Сейчас" — четверг 14 мая 2026, 12:00 MSK.
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, msk)

	// Свечи в порядке возрастания даты.
	prevWeek := time.Date(2026, 5, 4, 0, 0, 0, 0, msk)  // прошлая закрытая неделя
	twoWeeksAgo := time.Date(2026, 4, 27, 0, 0, 0, 0, msk)
	currentWeekOpen := time.Date(2026, 5, 11, 0, 0, 0, 0, msk) // не закрыта

	tests := []struct {
		name      string
		items     []*domain.RSIItemTechAnalyse
		wantOK    bool
		wantDate  time.Time
	}{
		{
			name: "массив включает текущую открытую неделю — берём предыдущую",
			items: []*domain.RSIItemTechAnalyse{
				{Date: twoWeeksAgo},
				{Date: prevWeek},
				{Date: currentWeekOpen},
			},
			wantOK:   true,
			wantDate: prevWeek,
		},
		{
			name: "массив без текущей недели — последний элемент уже закрыт",
			items: []*domain.RSIItemTechAnalyse{
				{Date: twoWeeksAgo},
				{Date: prevWeek},
			},
			wantOK:   true,
			wantDate: prevWeek,
		},
		{
			name:   "пустой массив",
			items:  nil,
			wantOK: false,
		},
		{
			name: "только текущая неделя — нет закрытых",
			items: []*domain.RSIItemTechAnalyse{
				{Date: currentWeekOpen},
			},
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := lastClosedWeeklyRSI(tc.items, now, msk)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if !got.Date.Equal(tc.wantDate) {
				t.Fatalf("date = %v, want %v", got.Date, tc.wantDate)
			}
		})
	}
}
```

- [ ] **Step 3.2: Запустить тест — упасть с «undefined»**

```bash
cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/ -run TestLastClosedWeeklyRSI
```

Expected: FAIL — `undefined: lastClosedWeeklyRSI`.

- [ ] **Step 3.3: Реализовать `lastClosedWeeklyRSI`**

Создать `internal/service/trading_strategy/golden_x/candle.go`:

```go
package golden_x

import (
	"time"

	"tinvest/internal/domain"
)

// startOfWeekMSK возвращает понедельник 00:00 в указанной локации для даты t.
func startOfWeekMSK(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	// time.Monday = 1, time.Sunday = 0. Нужно сдвинуть так, чтобы понедельник = 0.
	weekday := (int(t.Weekday()) + 6) % 7
	year, month, day := t.Date()
	return time.Date(year, month, day-weekday, 0, 0, 0, 0, loc)
}

// lastClosedWeeklyRSI возвращает последнюю точку RSI, чья дата принадлежит
// неделе строго раньше текущей (граница — понедельник 00:00 MSK).
// Если такой точки нет — возвращает (nil, false).
func lastClosedWeeklyRSI(items []*domain.RSIItemTechAnalyse, now time.Time, loc *time.Location) (*domain.RSIItemTechAnalyse, bool) {
	currentWeekStart := startOfWeekMSK(now, loc)
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] == nil {
			continue
		}
		if items[i].Date.Before(currentWeekStart) {
			return items[i], true
		}
	}
	return nil, false
}
```

- [ ] **Step 3.4: Запустить тест — должен пройти**

```bash
cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/ -run TestLastClosedWeeklyRSI -v
```

Expected: PASS — все четыре подтеста зелёные.

- [ ] **Step 3.5: Использовать `lastClosedWeeklyRSI` в `trade.go`**

В `trade.go` найти место, где раньше брался `rsi[0]`:

```go
RSIInfo.WriteToMap(
    share.ID,
    domain.Item{
        InstrumentName: share.Name,
        RSILength:      share.RSILength,
        RSIValue:       utils.CombinePrice(rsi[0].SignalLine.Units, rsi[0].SignalLine.Nano),
    })
if utils.CombinePrice(rsi[0].SignalLine.Units, rsi[0].SignalLine.Nano) > 40 {
    continue
}
```

Заменить на:

```go
closedRSI, ok := lastClosedWeeklyRSI(rsi, dateNow, loc)
if !ok {
    logger.InfoContext(ctx, "no closed weekly RSI candle for share", "share", share.Name)
    continue
}
rsiValue := utils.CombinePrice(closedRSI.SignalLine.Units, closedRSI.SignalLine.Nano)

RSIInfo.WriteToMap(
    share.ID,
    domain.Item{
        InstrumentName: share.Name,
        RSILength:      share.RSILength,
        RSIValue:       rsiValue,
    })
if rsiValue > 40 {
    continue
}
```

Имя локали `loc` уже доступно — оно создаётся в начале функции.

- [ ] **Step 3.6: Сборка + полный прогон тестов пакета**

```bash
cd /home/oleg/GolandProjects/tinvest && go build ./... && go test ./internal/service/trading_strategy/golden_x/...
```

Expected: успешная сборка, тест зелёный.

- [ ] **Step 3.7: Коммит**

```bash
cd /home/oleg/GolandProjects/tinvest && git add internal/service/trading_strategy/golden_x && git commit -m "feat(golden_x): use last closed weekly candle for RSI signal"
```

---

### Task 4: A1 — дедупликация алертов (с тестом)

**Files:**
- Create: `internal/service/trading_strategy/golden_x/dedup.go`
- Create: `internal/service/trading_strategy/golden_x/dedup_test.go`
- Modify: `internal/service/trading_strategy/golden_x/types.go` (добавить поле + инициализация в `NewService`)
- Modify: `internal/service/trading_strategy/golden_x/trade.go` (использовать state)

- [ ] **Step 4.1: Написать тест для `alertState`**

Создать `internal/service/trading_strategy/golden_x/dedup_test.go`:

```go
package golden_x

import "testing"

func TestAlertState_ShouldAlert(t *testing.T) {
	s := newAlertState()
	const id = "share-1"

	tests := []struct {
		name string
		tier alertTier
		want bool
	}{
		{"первый Brown — алерт", tierBrown, true},
		{"повторный Brown — нет", tierBrown, false},
		{"переход Brown→Yellow — алерт", tierYellow, true},
		{"повторный Yellow — нет", tierYellow, false},
		{"переход Yellow→Green — алерт", tierGreen, true},
		{"повторный Green — нет", tierGreen, false},
		{"откат Green→None (RSI > 40) — нет (молчим)", tierNone, false},
		{"снова Brown после None — алерт", tierBrown, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.ShouldAlert(id, tc.tier)
			if got != tc.want {
				t.Fatalf("ShouldAlert(%v) = %v, want %v", tc.tier, got, tc.want)
			}
		})
	}
}

func TestAlertState_IndependentShares(t *testing.T) {
	s := newAlertState()
	if !s.ShouldAlert("a", tierBrown) {
		t.Fatal("first Brown for a must alert")
	}
	if !s.ShouldAlert("b", tierBrown) {
		t.Fatal("first Brown for b must alert")
	}
	if s.ShouldAlert("a", tierBrown) {
		t.Fatal("repeat Brown for a must not alert")
	}
}
```

- [ ] **Step 4.2: Запустить тест — упасть с «undefined»**

```bash
cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/ -run TestAlertState
```

Expected: FAIL — `undefined: newAlertState, alertTier, tier...`.

- [ ] **Step 4.3: Реализовать `dedup.go`**

Создать `internal/service/trading_strategy/golden_x/dedup.go`:

```go
package golden_x

import "sync"

type alertTier int

const (
	tierNone alertTier = iota
	tierBrown
	tierYellow
	tierGreen
)

// tierFromRSI — классификация по тем же порогам, что в notification.
func tierFromRSI(rsi float64) alertTier {
	switch {
	case rsi < 31:
		return tierGreen
	case rsi < 35:
		return tierYellow
	case rsi <= 40:
		return tierBrown
	default:
		return tierNone
	}
}

// alertState хранит последний отправленный tier по shareID и решает,
// нужно ли слать новый алерт. Алерт шлётся только при изменении tier
// и только если новый tier != tierNone (откат RSI > 40 — молчим).
type alertState struct {
	mu   sync.RWMutex
	last map[string]alertTier
}

func newAlertState() *alertState {
	return &alertState{last: make(map[string]alertTier)}
}

// ShouldAlert возвращает true, если для shareID нужно слать алерт с данным tier.
// Сайд-эффект: при возврате true (или при tier == tierNone) обновляет внутреннее
// состояние, чтобы следующие вызовы с тем же tier возвращали false.
func (s *alertState) ShouldAlert(shareID string, tier alertTier) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.last[shareID]
	if tier == tierNone {
		s.last[shareID] = tier
		return false
	}
	if prev == tier {
		return false
	}
	s.last[shareID] = tier
	return true
}
```

- [ ] **Step 4.4: Запустить тест — должен пройти**

```bash
cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/ -run TestAlertState -v
```

Expected: PASS — обе функции зелёные.

- [ ] **Step 4.5: Добавить state в `service`**

В `types.go` модифицировать `service` и `NewService`:

```go
type service struct {
	rsi                         rsiInstrument
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
	state                       *alertState
}

func NewService(rsi rsiInstrument, instrumentsServiceClient grpc.InstrumentsServiceClient, marketDataServiceGrpcClient grpc.MarketDataServiceClient, tgClient telegram.Client) *service {
	return &service{
		rsi:                         rsi,
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
		state:                       newAlertState(),
	}
}
```

**Важно:** один `service` — один state. Но в `app.go` стратегия запускается двумя
горутинами через **один и тот же** `service_provider.GetGoldenXTradingService()`,
который кэширует сервис как синглтон. Значит state делится между Gold и Growth.
**Это OK**, потому что `shareID` уникален между списками (бумаги не пересекаются —
проверить!). Если пересекаются — добавить префикс в ключ. См. Step 4.7.

- [ ] **Step 4.6: Использовать state в `trade.go`**

В `trade.go` перед записью в `info` (которая попадает в Trade-нотификацию):

Найти:

```go
if rsiValue > 40 {
    continue
}

info.WriteToMap(
    share.ID,
    domain.Item{
        InstrumentName: share.Name,
        RSIValue:       rsiValue,
    })
```

Заменить на:

```go
tier := tierFromRSI(rsiValue)
if !s.state.ShouldAlert(share.ID, tier) {
    continue
}

info.WriteToMap(
    share.ID,
    domain.Item{
        InstrumentName: share.Name,
        RSIValue:       rsiValue,
    })
```

`tierFromRSI` для `rsiValue > 40` вернёт `tierNone`, `ShouldAlert` для `tierNone`
вернёт false, бумага в `info` не попадёт — поведение эквивалентно старому
`if > 40 { continue }` плюс дедупликация.

`RSIInfo` (список «промежуточных значений») — **не дедуплицируем**. Это
обзорный отчёт по всем бумагам в коллекции, его получаем как есть.

- [ ] **Step 4.7: Проверка пересечений списков**

```bash
cd /home/oleg/GolandProjects/tinvest && grep -E "^\s+ID:" internal/app/init_collection.go | sort | uniq -d
```

Expected: пустой вывод. Если что-то выведется — добавить префикс в ключ state
(`fmt.Sprintf("%d:%s", kind, shareID)`).

- [ ] **Step 4.8: Сборка + тесты**

```bash
cd /home/oleg/GolandProjects/tinvest && go build ./... && go test ./internal/service/trading_strategy/golden_x/...
```

Expected: всё зелёное.

- [ ] **Step 4.9: Коммит**

```bash
cd /home/oleg/GolandProjects/tinvest && git add internal/service/trading_strategy/golden_x && git commit -m "feat(golden_x): in-memory deduplication of buy alerts by tier"
```

---

### Task 5: A3 — логирование recover + расширение warmup до 80 недель

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/trade.go` (recover + параметр TimeStampPbGenerator)

- [ ] **Step 5.1: Логировать панику в `recover()`**

В `trade.go:17-21` найти:

```go
defer func() {
    if r := recover(); r != nil {
        err = fmt.Errorf("panic: %v", r)
    }
}()
```

Заменить на:

```go
defer func() {
    if r := recover(); r != nil {
        logger.ErrorContext(ctx, fmt.Sprintf("panic in golden_x.Trade: %v", r))
        err = fmt.Errorf("panic: %v", r)
    }
}()
```

- [ ] **Step 5.2: Увеличить окно истории до 80 недель**

В `trade.go` найти вызов:

```go
utils.TimeStampPbGenerator(dateNow, -20, in.Interval),
```

Заменить на:

```go
utils.TimeStampPbGenerator(dateNow, -80, in.Interval),
```

- [ ] **Step 5.3: Сборка**

```bash
cd /home/oleg/GolandProjects/tinvest && go build ./...
```

Expected: успешная сборка.

- [ ] **Step 5.4: Коммит**

```bash
cd /home/oleg/GolandProjects/tinvest && git add internal/service/trading_strategy/golden_x/trade.go && git commit -m "fix(golden_x): log panic in recover, extend RSI warmup to 80 weeks"
```

---

### Task 6: A6 — финальная косметика

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/trade.go`

К этому моменту опечатка и пустой `if shareTip == 1 {}` уже вычищены (см. блок «уже выполнено» сверху). Остались `shareRSI` и `ok == true` — но они **уже устранены** в Task 2 step 2.1 (полная замена цикла). Проверяем, что ничего не пропустили.

- [ ] **Step 6.1: Найти оставшийся `== true`**

```bash
cd /home/oleg/GolandProjects/tinvest && grep -rn "== true" internal/service/trading_strategy/golden_x/
```

Expected: пустой вывод. Если осталось — заменить `if ok == true` на `if ok`.

- [ ] **Step 6.2: Найти оставшиеся `shareTip` / `находящтеся`**

```bash
cd /home/oleg/GolandProjects/tinvest && grep -rn "shareTip\|находящтеся" internal/service/trading_strategy/golden_x/
```

Expected: пустой вывод.

- [ ] **Step 6.3: Прогон gofmt**

```bash
cd /home/oleg/GolandProjects/tinvest && gofmt -l internal/service/trading_strategy/golden_x/
```

Expected: пустой вывод (нет неформатированных файлов).

- [ ] **Step 6.4: Коммит (только если были правки в 6.1/6.3)**

```bash
cd /home/oleg/GolandProjects/tinvest && git status
# если есть незакоммиченное:
git add internal/service/trading_strategy/golden_x && git commit -m "style(golden_x): final cleanup"
```

---

### Task 7: Финальная верификация всего этапа A

**Files:** (только проверка)

- [ ] **Step 7.1: Полный билд**

```bash
cd /home/oleg/GolandProjects/tinvest && go build ./...
```

Expected: успешная сборка.

- [ ] **Step 7.2: Все тесты**

```bash
cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x/... -v
```

Expected: PASS для `TestLastClosedWeeklyRSI` (4 подтеста), `TestAlertState_ShouldAlert` (8 подтестов), `TestAlertState_IndependentShares`.

- [ ] **Step 7.3: Vet**

```bash
cd /home/oleg/GolandProjects/tinvest && go vet ./internal/service/trading_strategy/golden_x/...
```

Expected: пустой вывод.

- [ ] **Step 7.4: Проверка goverter-генерации (если применимо)**

```bash
cd /home/oleg/GolandProjects/tinvest && grep -rn "dto.Trade\|ShareTip" internal/converter/ 2>/dev/null
```

Если что-то нашлось — выполнить `make generate` и закоммитить регенерированные файлы.

- [ ] **Step 7.5: Smoke-test в dev (опционально, если есть env/local.env)**

```bash
cd /home/oleg/GolandProjects/tinvest && APP_ENV=dev go run ./cmd/main
```

Ожидаемое поведение в логах:
- Один лог `Воркер Golden RSI начал работу` каждую минуту (cron `* * * * *` для dev).
- Для каждой бумаги в `GrowthShare`: либо запись в `info`/`RSIInfo`, либо лог `no closed weekly RSI candle for share`.
- Никаких `panic` без сопроводительного `logger.ErrorContext`.
- Telegram-сообщения приходят, при повторных запусках с тем же tier — **не** дублируются.

Остановить через Ctrl-C после 2–3 минут наблюдения.

- [ ] **Step 7.6: Финальный коммит (если что-то осталось)**

```bash
cd /home/oleg/GolandProjects/tinvest && git status
```

Если чистый — этап A завершён.

---

## Self-Review

**1. Spec coverage:**
- A1 (дедуп) — Task 4 ✅
- A2 (закрытая свеча) — Task 3 ✅
- A3 (recover-лог + warmup 80) — Task 5 ✅
- A4 (StrategyKind enum) — «уже выполнено» в верхнем блоке + Task 1 (хвост в `trade.go`) ✅
- A5 (итерация по `ShareList.All()`) — Task 2 ✅
- A6 (косметика) — частично в «уже выполнено», финальная проверка в Task 6 ✅

**2. Placeholder scan:** Все шаги содержат конкретный код или конкретные команды. TBD/TODO отсутствуют.

**3. Type consistency:**
- `alertTier` определён в Task 4 step 4.3, используется в `tierFromRSI` (Task 4 step 4.3) и `ShouldAlert` (Task 4 step 4.6). Совпадает.
- `lastClosedWeeklyRSI(items, now, loc)` — сигнатура одинакова в тесте (Task 3 step 3.1) и реализации (Task 3 step 3.3). Совпадает.
- `dto.StrategyKind` / `StrategyKindDividend` / `StrategyKindGrowth` — использованы в `app.go` (блок «уже выполнено»), `notifications.go` (через `.Medal()`), `trade.go` (через `in.Kind` в Task 1). Совпадает.

Issues: нет.
