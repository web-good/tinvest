# Levels: ATR и уровни в отчёте + калибровка под RUAL

- Дата: 2026-06-07
- Ветка: `feat/levels-volume-profile-strategy`
- Статус: согласовано, готово к написанию плана

## Проблема

Три доработки стратегии levels:

1. **ATR в отчёте.** Отчёт бэктеста не показывает волатильность инструмента в
   момент сделки. ATR — «линейка» всей стратегии (все дистанции в ATR), но в
   журнале сделок его нет.
2. **Уровень в каждой сделке.** Вход — это отскок/ретест от HVN-уровня
   (`support`), цель — следующий HVN сверху (`resistance`). Сейчас в журнале
   сделок этих уровней нет: видно только цену входа/выхода, но не «от какого
   уровня вошли и куда целились».
3. **Лучшие параметры под RUAL.** После перехода на внутрибаровое исполнение
   стопов (`min(уровень, open)`) RUAL стал убыточным (−17.65%, PF 0.299):
   жёсткий стоп `support − 1·ATR` слишком узкий, его выбивает интрабарным шумом.
   Текущий калибровочный грид `data/params/rusal/levels_grid.json` **не свипает
   `SLMult` вообще** — то есть не трогает главный подозреваемый.

## Решённые при брейншторминге вопросы

- **Какой уровень показывать:** оба — `support` (уровень входа) и `resistance`
  (цель). Resistance уже едет в существующем `Signal.TakeProfit`.
- **ATR в отчёте:** per-trade ATR на входе (ATR в момент открытия сделки).
  Сводку ATR в шапке не делаем (YAGNI) — per-trade достаточно.
- **Кто гоняет калибровку:** грид готовлю я; прогон `-calibrate` (нужен токен
  T_BANK + сеть) пользователь запускает сам и присылает победителя; победителя
  зашиваем в `rusal.go DefaultParams` отдельным шагом.
- **Колонки для scalping:** scalping volume-профиль не использует, поэтому в его
  отчётах `Level`/`ATR` будут нулевыми (как уже сделано с `Signal.RSI`). Логику
  рендера под это не усложняем (YAGNI).

## Дизайн

Контекст входа (уровни + ATR) известен в `decide()` на Buy, но `Trade`
создаётся только на выходе. Протаскиваем контекст через mock-портфель: сигнал
несёт уровни/ATR → `portfolio.open` их запоминает → `close` штампует в `Trade` →
рендер выводит.

### 1. `internal/service/trading_strategy/scalping/model/signal.go`

Добавляем два поля в `Signal` (resistance уже несёт `TakeProfit`):

```go
type Signal struct {
	Kind           SignalKind
	InstrumentID   string
	InstrumentName string
	Ticker         string
	Price          float64
	TakeProfit     float64
	StopLoss       float64
	RSI            float64
	Level          float64 // entry support level (HVN); 0 when n/a
	ATR            float64 // ATR at entry; 0 when n/a
	Reason         string
}
```

### 2. `internal/service/trading_strategy/levels/strategy/core/core.go` — `decide()`

В ветке Buy дополнительно выставляем уровень входа и ATR (TakeProfit =
resist.Price уже ставится в той же ветке):

```go
if s.entryQualifies(in.price, stop, target, in.atr) {
	sig.Kind, sig.StopLoss, sig.TakeProfit = model.SignalBuy, stop, target
	sig.Level, sig.ATR = support.Price, in.atr
	if in.recentlyBelow {
		sig.Reason = "RETEST"
	} else {
		sig.Reason = "BOUNCE"
	}
}
```

Управление позицией (выходы) не трогаем — там контекст входа уже не нужен.

### 3. `internal/domain/backtest/types.go` — `Trade`

Три новых поля:

```go
type Trade struct {
	EntryTime       time.Time
	EntryPrice      float64
	ExitTime        time.Time
	ExitPrice       float64
	Quantity        int64
	Reason          string
	PnL             float64
	PnLPct          float64
	BarsHeld        int
	SupportLevel    float64 // HVN support the entry bounced off; 0 when n/a
	ResistanceLevel float64 // HVN resistance / target at entry; 0 when n/a
	ATR             float64 // ATR at entry; 0 when n/a
}
```

### 4. `internal/domain/backtest/portfolio.go`

`open` принимает контекст входа и запоминает его; `close` штампует в `Trade`:

```go
type portfolio struct {
	cfg         Config
	cash        float64
	qty         int64
	entryPrice  float64
	entryTime   time.Time
	entryBar    int
	entryLevel  float64 // support level captured at entry
	entryTarget float64 // resistance/target captured at entry
	entryATR    float64 // ATR captured at entry
	bar         int
}

func (p *portfolio) open(price float64, t time.Time, level, target, atr float64) {
	// ... existing lot-sizing / cash math unchanged ...
	p.entryPrice = price
	p.entryTime = t
	p.entryBar = p.bar
	p.entryLevel = level
	p.entryTarget = target
	p.entryATR = atr
}
```

В `close` добавить в собираемый `Trade`:
`SupportLevel: p.entryLevel, ResistanceLevel: p.entryTarget, ATR: p.entryATR`.
После закрытия обнулять `entryLevel/entryTarget/entryATR` вместе с `entryPrice`.

### 5. `internal/domain/backtest/engine.go` — `Run()`, ветка `SignalBuy`

Передаём контекст входа из сигнала:

```go
case model.SignalBuy:
	if p.qty == 0 {
		p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR)
	}
```

### 6. `internal/domain/backtest/report.go` — рендер

**Markdown-журнал** — добавить три колонки **в конце строки** (после «PnL %»),
заголовки «Support», «Resist», «ATR», формат `%.4f`. Хвост строки, чтобы не
ломать визуальный порядок существующих колонок.

**CSV** (`RenderTradesCSV`) — добавить в заголовок и строки колонки
`support_level,resistance_level,atr` (формат `%.6f`).

### 7. `data/params/rusal/levels_grid.json` — переориентированный грид

Свипаем знобы, лечащие интрабарное выбивание (особенно `SLMult`):

```json
{
  "SLMult":      [1.0, 1.5, 2.0, 2.5],
  "TrailArmATR": [0.5, 1.0, 1.5],
  "TrailMult":   [2.0, 2.5, 3.0],
  "MinRR":       [1.2, 1.5, 2.0],
  "RoomATR":     [1.5, 2.0, 2.5]
}
```

4×3×3×3×3 = 324 комбинации. `HVNFactor/MaxExtensionATR/LevelTolATR` остаются на
дефолтах (второй проход потом, при необходимости). Грид — это data-файл, не код;
тестами не покрывается, проверяется тем, что `-calibrate` его парсит и гоняет.

### 8. `docs/levels/strategy.md`

- §6: упомянуть новые колонки журнала (Support / Resist / ATR).
- Добавить команду калибровки (walk-forward), которой пользуется владелец:

```
go run ./cmd/backtest -ticker RUAL -strategy levels -interval Hour1 -months 25 \
  -calibrate data/params/rusal/levels_grid.json -metric expectancy -min-trades 20 -test-months 3
```

## Калибровка — рабочий процесс (ручной шаг владельца)

1. Владелец запускает команду выше (нужен токен T_BANK; кэш `RUAL_Hour1` тёплый).
2. Победитель — в `reports/..._best.md` (метрика `expectancy`, отсев комбинаций
   с < 20 сделок, walk-forward на последние 3 месяца против переобучения).
3. Победителя присылает → его значения зашиваются в
   `internal/service/trading_strategy/levels/strategy/rusal/rusal.go`
   `DefaultParams()`. Это отдельный шаг **после** прогона, вне данного плана кода.

## Тесты

- `core_test.go`: Buy-кейс выставляет `sig.Level == support.Price` и
  `sig.ATR == in.atr` (а также существующий `TakeProfit == resist.Price`).
- `portfolio`/`engine`-тесты: после round-trip `Trade.SupportLevel`,
  `Trade.ResistanceLevel`, `Trade.ATR` равны переданным в `open` значениям;
  не-levels Buy (level/atr=0) даёт нулевые поля.
- `report` (если есть тесты рендера) / иначе golden-строка: CSV-заголовок
  содержит новые колонки; строка сделки печатает значения.

## Ожидаемый эффект

- Журнал каждой сделки показывает: от какого HVN-уровня вошли, куда целились и
  какой был ATR — видно контекст риск/прибыли и волатильность на входе.
- Грид впервые свипает ширину стопа (`SLMult`) и арминг трейла — есть шанс
  вернуть RUAL в плюс за счёт более широкого стопа против интрабарного шума.

## Вне рамок (YAGNI)

- ATR-сводка в шапке отчёта (per-trade достаточно).
- Авто-запуск калибровки из кода / сам прогон grid-search ассистентом.
- Второй проход грида (HVNFactor/MaxExtensionATR/LevelTolATR/ATRPeriod).
- Условное скрытие нулевых колонок для scalping-отчётов.
- Перенос победителя в rusal.go — отдельный шаг после прогона, не в этом плане.
