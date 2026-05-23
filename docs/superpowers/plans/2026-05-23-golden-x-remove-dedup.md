# Golden X: убрать dedup из уведомлений — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Каждый тик стратегии отправляет уведомление со всеми акциями в зонах покупки/продажи, без дедупликации.

**Architecture:** Переносим `alertTier` + tier-константы в `percentile.go`, удаляем dedup-инфраструктуру (`dedup.go`, `dedup_test.go`, `alertState` в `service`), упрощаем цикл в `trade.go`, обновляем документацию.

**Tech Stack:** Go 1.25

---

### Task 1: Перенос `alertTier` и tier-констант в `percentile.go`

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/percentile.go:1-9`

- [ ] **Step 1: Добавить тип и константы в `percentile.go`**

Вставить перед функцией `percentile()` (после блока импортов, строка 9):

```go
type alertTier int

const (
	tierNone       alertTier = iota
	tierYellow               // buy: RSI < p15
	tierGreen                // buy: RSI < p5
	tierSellYellow           // sell (Gold only): RSI > p80
	tierSellOrange           // sell: RSI > p90 (Growth's single tier)
	tierSellRed              // sell (Gold only): RSI > p95
)
```

- [ ] **Step 2: Убедиться, что проект компилируется**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./internal/service/trading_strategy/golden_x/...`
Expected: ошибка дублирования `alertTier` (определён и в `dedup.go`, и в `percentile.go`). Это ожидаемо — удаление `dedup.go` в следующем таске.

---

### Task 2: Удаление dedup-инфраструктуры

**Files:**
- Delete: `internal/service/trading_strategy/golden_x/dedup.go`
- Delete: `internal/service/trading_strategy/golden_x/dedup_test.go`
- Modify: `internal/service/trading_strategy/golden_x/types.go:1-33`

- [ ] **Step 1: Удалить `dedup.go`**

```bash
rm internal/service/trading_strategy/golden_x/dedup.go
```

- [ ] **Step 2: Удалить `dedup_test.go`**

```bash
rm internal/service/trading_strategy/golden_x/dedup_test.go
```

- [ ] **Step 3: Убрать `state` из структуры `service` и конструктора в `types.go`**

Было (`types.go:15-33`):

```go
type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
	state                       *alertState
}

func NewService(
	instrumentsServiceClient grpc.InstrumentsServiceClient,
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	tgClient telegram.Client,
) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
		state:                       newAlertState(),
	}
}
```

Стало:

```go
type service struct {
	instrumentServiceGrpcClient grpc.InstrumentsServiceClient
	marketDataServiceGrpcClient grpc.MarketDataServiceClient
	tgClient                    telegram.Client
}

func NewService(
	instrumentsServiceClient grpc.InstrumentsServiceClient,
	marketDataServiceGrpcClient grpc.MarketDataServiceClient,
	tgClient telegram.Client,
) *service {
	return &service{
		instrumentServiceGrpcClient: instrumentsServiceClient,
		marketDataServiceGrpcClient: marketDataServiceGrpcClient,
		tgClient:                    tgClient,
	}
}
```

- [ ] **Step 4: Убедиться, что проект компилируется**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./internal/service/trading_strategy/golden_x/...`
Expected: ошибка в `trade.go` — `s.state` и `alertSnapshot` больше не существуют.

---

### Task 3: Упрощение цикла в `trade.go`

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/trade.go:92-121`

- [ ] **Step 1: Убрать dedup-логику из цикла**

Было (`trade.go:92-121`):

```go
		buyTier := tierFromAdaptive(sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
		sellTier := sellTierFromAdaptive(sig.RSI, sig.SellThresholds, in.Kind)

		// Buy and sell zones are mutually exclusive on RSI — picks whichever
		// (if any) is non-None.
		finalTier := buyTier
		if sellTier != tierNone {
			finalTier = sellTier
		}

		snap := alertSnapshot{
			tier:       finalTier,
			trendOK:    sig.TrendStatus == dto.TrendWith,
			divergence: sig.DivergenceOK,
			volumeOK:   sig.VolumeOK,
		}
		if !s.state.ShouldAlert(share.ID, snap) {
			continue
		}

		item := domain.Item{
			InstrumentName: share.Name,
			RSIValue:       sig.RSI,
		}
		switch finalTier {
		case tierYellow, tierGreen:
			buyInfo.WriteToMap(share.ID, item)
		case tierSellYellow, tierSellOrange, tierSellRed:
			sellInfo.WriteToMap(share.ID, item)
		}
```

Стало:

```go
		buyTier := tierFromAdaptive(sig.RSI, sig.Thresholds.P5, sig.Thresholds.P15)
		sellTier := sellTierFromAdaptive(sig.RSI, sig.SellThresholds, in.Kind)

		item := domain.Item{
			InstrumentName: share.Name,
			RSIValue:       sig.RSI,
		}
		switch {
		case buyTier != tierNone:
			buyInfo.WriteToMap(share.ID, item)
		case sellTier != tierNone:
			sellInfo.WriteToMap(share.ID, item)
		}
```

- [ ] **Step 2: Убедиться, что проект компилируется и тесты проходят**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./internal/service/trading_strategy/golden_x/... && go test ./internal/service/trading_strategy/golden_x/...`
Expected: PASS

- [ ] **Step 3: Коммит**

```bash
git add internal/service/trading_strategy/golden_x/percentile.go \
       internal/service/trading_strategy/golden_x/types.go \
       internal/service/trading_strategy/golden_x/trade.go
git rm internal/service/trading_strategy/golden_x/dedup.go \
      internal/service/trading_strategy/golden_x/dedup_test.go
git commit -m "refactor(golden_x): remove dedup — notify all shares in buy/sell zones every tick"
```

---

### Task 4: Обновление документации

**Files:**
- Modify: `docs/golden_x/strategy.md:1-6,27-28,39,123-132`
- Modify: `docs/golden_x/alerts.md:155-165`

- [ ] **Step 1: Обновить `strategy.md` — диаграмму жизненного цикла**

В строке 3 убрать упоминание дедупликации:

Было:
```
Ядро стратегии — чистая функция `Detect()` в `internal/service/trading_strategy/golden_x/detector.go:19`. Никакого I/O, никакого времени, никакой телеметрии: на вход — массив недельных свечей (закрытые + текущая формирующаяся), на выходе — `dto.Signal`. Всю обвязку (загрузку данных, дедупликацию, отправку в Telegram) делает обёртка `service.Trade`.
```

Стало:
```
Ядро стратегии — чистая функция `Detect()` в `internal/service/trading_strategy/golden_x/detector.go:19`. Никакого I/O, никакого времени, никакой телеметрии: на вход — массив недельных свечей (закрытые + текущая формирующаяся), на выходе — `dto.Signal`. Всю обвязку (загрузку данных, отправку в Telegram) делает обёртка `service.Trade`.
```

- [ ] **Step 2: Убрать `[дедупликация ShouldAlert()]` из ASCII-диаграммы**

В блоке строк 7-33: удалить строки 27-29:

```
[dto.Signal] → service.Trade
                ↓
        [дедупликация ShouldAlert()]
                ↓
        [форматирование notification.Trade()]
```

Заменить на:

```
[dto.Signal] → service.Trade
                ↓
        [форматирование notification.Trade()]
```

- [ ] **Step 3: Убрать trade-off про дедуп из строки 39**

Удалить целиком:

```
**Trade-off:** значения индикаторов «дышат» с ценой в течение недели — tier может смениться и обратно. Дедуп `ShouldAlert()` срабатывает только на смену tier, поэтому без дополнительной защиты возможны дублирующие алёрты внутри одной недели. Антифлап вынесен в отдельную задачу.
```

- [ ] **Step 4: Заменить §7 (Дедупликация) на §7 (Отправка)**

Убрать строки 123-132 (весь раздел «Шаг 7. Дедупликация»). Заменить содержимым:

```markdown
### Шаг 7. Формирование уведомления

Все акции с tier ≠ `tierNone` попадают в уведомление — покупочные (`tierYellow`, `tierGreen`) в секцию «Сигналы на покупку», продажные (`tierSellYellow`, `tierSellOrange`, `tierSellRed`) в секцию «Сигналы на продажу». Дедупликации нет: каждый тик стратегии отправляет полный список всех акций, находящихся в зонах покупки или продажи.
```

- [ ] **Step 5: Обновить «Шаг 8» → «Шаг 8» (заголовок тот же, убрать упоминание «одной партией»)**

Строка 134: оставить как есть — «Шаг 8. Отправка в Telegram» не упоминает dedup.

- [ ] **Step 6: Обновить `alerts.md` — удалить секцию «Дедупликация»**

Удалить строки 155-165 (целиком секция «## Дедупликация — что важно знать про повторы»).

- [ ] **Step 7: Коммит**

```bash
git add docs/golden_x/strategy.md docs/golden_x/alerts.md
git commit -m "docs(golden_x): remove dedup references from strategy and alerts docs"
```
