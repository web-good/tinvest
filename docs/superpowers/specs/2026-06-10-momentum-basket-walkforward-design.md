# Momentum: индивидуальные параметры на пул бумаг + walk-forward корзина

Дата: 2026-06-10
Ветка: feat/momentum-daily-trend-cooldown
Стратегия: momentum (long-only EMA-trend + MACD bullish cross + volume + daily-ATR-room, Hour1)

## Проблема и цель

Momentum — селективная стратегия (~1 сделка/месяц на бумагу). За 6-месячное
OOS-окно одна бумага даёт 3–5 сделок — статистически незначимо, по ним нельзя
судить, переобучены ли подобранные параметры. План пользователя: торговать пул
бумаг, у **каждой свои индивидуальные параметры**. Это правильно и не меняется.

Корзина — **только инструмент валидации**, не способ торговли: она пулит
OOS-сделки нескольких индивидуально-откалиброванных бумаг в одну выборку
(~18–20 сделок), чтобы получить статистически осмысленный агрегат и честно
ответить «обобщается ли процесс калибровки на новых данных». Каждая бумага при
этом сохраняет и торгует свои параметры — пул сделок собирается из прогонов с
индивидуальными параметрами каждой бумаги.

Параллельно расширяем пул: добавляем 4 новые бумаги первоклассными (как rusal).

## Часть A — добавить 4 бумаги (SBER, GAZP, NVTK, MDMG)

Повторяет существующий паттерн (rusal/afks/ydex/plzl) один-в-один.

Для каждого из тикеров **SBER, GAZP, NVTK, MDMG**:

1. **Пакет** `internal/service/trading_strategy/momentum/strategy/<pkg>/<pkg>.go`
   с экспортируемой `Ticker` const и `DefaultParams() core.Params`. Стартовые
   значения = текущий `genericMomentumDefaults` (плейсхолдер, как было у rusal до
   калибровки). Комментарий пакета: «откалибруй `-calibrate` и захардкодь
   победителя сюда».
   - Пакеты: `sber` (SBER), `gazp` (GAZP), `nvtk` (NVTK), `mdmg` (MDMG).
2. **Регистрация** в `internal/service/backtest/momentum_registry.go` — добавить
   4 записи в `momentumRegistry` + 4 теста по образцу
   `momentum_registry_test.go` (`TestMomentumLookupRegistered<TICKER>`),
   проверяющих, что lookup отдаёт зарегистрированный binding и `Ticker` совпадает.
3. **Грид** `data/params/<lower(ticker)>/momentum_grid.json` в фазовом формате
   (как `data/params/afks/momentum_grid.json`: фаза `core` с keepTop, затем фаза
   `gates`), заточенный под профиль бумаги:
   - **SBER** — голубая фишка, ровные тренды: `SLMult [0.8,1.0,1.5]`,
     `DailyTrendPeriod [0,10,20]`, `CooldownBars [0,12,24]`,
     `MaxDailyATRUsed [0.5,0.7]`.
   - **GAZP** — тяжёлая, дивгэпы/новости: шире стопы `SLMult [1.0,1.5,2.0]`,
     упор на трендовый фильтр `DailyTrendPeriod [10,20,50]`,
     `CooldownBars [12,24]`.
   - **NVTK** — волатильная, санкционные шоки: `SLMult [1.0,1.5,2.0]`, теснее
     ATR-запас `MaxDailyATRUsed [0.5,0.6]`, `CooldownBars [12,24,48]`.
   - **MDMG** — малоликвид, гэпы, **короткая история**: консервативный грид
     `SLMult [1.0,1.5]`, `TakeProfitRR [1.5,2,3]`, `DailyTrendPeriod [10,20]`,
     `CooldownBars [12,24]`.

   Остальные оси (EMAPeriod, MACDFast/Slow, VolMultiplier, MACDBelowZeroOnly,
   SignalValidBars, MaxDriftATR) — по образцу afks-грида.

### Данные по новым тикерам

- SBER, GAZP, NVTK — ликвидны, без недавних сплитов → полное окно `-months 24` ок.
- **MDMG** — торгуется на MOEX ~с окт-2024 после редомициляции (символ новый,
  данных до этого под MDMG нет). Под `-months 24` провайдер просто вернёт ~20
  месяцев чистых свечей — провал-гэпа нет, спецобработка не нужна; следствие
  только меньшее число сделок у MDMG. Дневной lead-in (`dailyFrom = from - 1y`)
  для warmup EMA так же даст лишь доступную историю — это нормально.

## Часть B — walk-forward корзина

**Новый режим в `cmd/backtest`** через флаг
`-basket "AFKS,PLZL,RUAL,YDEX,SBER,GAZP,NVTK,MDMG"` (comma-separated). Когда
`-basket` непустой: `-ticker` игнорируется, `-explain`/`-params`/`-calibrate`
несовместимы (ошибка). Переиспользует `-months`, `-test-months` (обязателен > 0
для корзины — без OOS-окна пулить нечего), `-metric`, `-min-trades`, `-cash`,
`-fraction`, `-commission`, `-refresh`.

Новый флаг `-grid-dir` (default `data/params`): директория с гридами; путь грида
бумаги = `<grid-dir>/<lower(ticker)>/momentum_grid.json`.

### Поток (на каждую бумагу — то же, что `runCalibration` с `-test-months`, в цикле)

1. Резолв share, загрузка часовых + дневных свечей (с тем же дневным lead-in).
2. `SplitByTime` по границе `to.AddDate(0, -testMonths, 0)`.
3. `ParsePhases` грида бумаги → `RunPhases` на **раннем** окне (`gridDays`).
4. Победитель (`results[0].Params`) → `backtest.Run` на **OOS-хвосте** →
   `backtest.Compute` для per-stock метрик; сохраняем хвостовые `Trade`-ы,
   per-stock `Metrics` и параметры-победителя.
5. Бумаги, у которых нет грид-файла или 0 сделок на хвосте, помечаются в отчёте
   (пропущены / без OOS-сделок), но не валят прогон.

### Агрегация и отчёт

Чистая логика в `internal/service/backtest/basket.go`:

- `PooledMetrics(trades []backtest.Trade) backtest.Metrics` — трейд-основанные
  поля (TotalTrades, Wins/Losses, WinRate, GrossProfit/Loss, ProfitFactor,
  Expectancy, Sortino, AvgWin/Loss, Best/Worst). Equity-поля (MaxDrawdown, CAGR,
  NetPnLPct, ExposurePct) остаются нулевыми — они не пулятся между разными
  счетами и не имеют смысла на объединённой выборке.
- `BasketSummary` struct: агрегатные `PooledMetrics` + срез per-stock записей
  (ticker, #OOS-сделок, PF, NetPnL, NetPnL%, MaxDrawdown%, WinRate, сводка
  параметров-победителя, флаг skipped).

Рендер `RenderBasketMarkdown(summary, metric, meta) string` — две таблицы:
1. **Пул сделок** (агрегат): всего сделок, win-rate, PF, expectancy, sortino,
   gross profit/loss, лучшая/худшая. Главный вывод о генерализации.
2. **Разбивка по бумагам**: тикер | #OOS | PF | net % | max-DD % | win-rate |
   параметры-победителя.

Сознательно **без** склеенной equity-кривой и агрегатной max-DD (YAGNI:
портфель не торгуется одним счётом, склейка кривых с разными бар-таймстемпами
сомнительна; equity-измерение покрыто per-stock колонками).

### Размещение кода (повторяет текущий split «I/O в cmd, pure в service»)

- `cmd/backtest/main.go` — флаги `-basket`, `-grid-dir`; диспетчеризация на
  `runBasket(...)`; весь gRPC/файловый I/O, цикл по тикерам, загрузка свечей и
  гридов.
- `internal/service/backtest/basket.go` — `PooledMetrics`, `BasketSummary`,
  `RenderBasketMarkdown` (чистые, без I/O).
- Вывод: `reports/basket/basket_momentum_<stamp>.md` + пулённый
  `basket_momentum_<stamp>_trades.csv` (через существующий `RenderTradesCSV`).

## Тестирование

- Unit: `PooledMetrics` на синтетических сделках — детерминированно проверяет
  PF/WR/expectancy/agg по известным PnL (вкл. краевые: все выигрыши → PF=GrossProfit;
  пустой срез → нули).
- Unit: 4 новых registry-теста (lookup отдаёт зарегистрированный binding).
- Полный прогон корзины ходит в сеть (фетч свечей) → ручной/e2e шаг (как текущая
  калибровка), юнит-тестом не покрываем. e2e-проверка: команда отрабатывает и
  пишет отчёт с непустым пулом.

## Команда (пример запуска)

```
go run ./cmd/backtest -strategy momentum \
  -basket "AFKS,PLZL,RUAL,YDEX,SBER,GAZP,NVTK,MDMG" \
  -months 24 -test-months 6 -metric profit_factor -min-trades 18 \
  -out ./reports
```

## Вне объёма (YAGNI)

- Склеенная portfolio-equity и агрегатная max-DD.
- Fixed-params режим корзины (прогон захардкоженных параметров без рекалибровки) —
  возможное позднее расширение; сейчас делаем walk-forward.
- Автоподстановка победителя в пакеты — остаётся ручным шагом (пользователь
  хардкодит вручную после просмотра калибровки).
