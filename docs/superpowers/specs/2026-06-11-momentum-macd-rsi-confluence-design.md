# Momentum: переработка входа на MACD↔RSI-50 confluence

Дата: 2026-06-11
Ветка: feat/momentum-daily-trend-cooldown (целевой пакет — `internal/service/trading_strategy/momentum`)

## Цель

Упростить momentum-стратегию: удалить пять фильтров и переработать правило входа
так, чтобы оно срабатывало на совпадении двух независимых сигналов — бычьего
MACD-кросса и пересечения RSI уровня 50 снизу вверх — в пределах окна
`SignalValidBars` баров (симметрично: второй сигнал может прийти как до, так и
после первого). Условия «цена выше EMA» и «объём выше среднего» сохраняются.

## Что удаляется

Из `Params` momentum-пакета и всех его дефолтов, калибровочных сеток и тестов
удаляются поля и связанный с ними код:

- `MaxDailyATRUsed` — фильтр запаса дневного хода (daily-ATR room).
- `DailyTrendPeriod` — фильтр наклона дневной EMA (higher-timeframe).
- `MaxDriftATR` — ограничение сноса цены для отложенного входа.
- `MinATRFrac` — anti-churn фильтр.
- `CooldownBars` — пауза на повторный вход после выхода.
- `DailyATRPeriod` — становится мёртвым вместе с дневным ATR (его единственные
  потребители — удалённый daily-room фильтр и текст причины входа).

**Важно:** `MinATRFrac` существует также в пакетах `levels` и `scalping` — это
отдельные структуры `Params`. Их трогать нельзя; удаление ограничено momentum.

Удаляемый код в `core.go`:

- Поля шелла: `barsSinceExit`, `prevInPosition`, `armedCrossPrice`,
  `barsSinceArm`, `consumedArmPrice`, `consumedArmBars`.
- Методы: `trackCooldown`, `advanceArm`, `qualifiesAsTrigger`, `clearArm`.
- Константы: `dailyTrendSlopeBars`, `cooldownSaturate`.
- В `buildInput`: расчёт `dailyATR`, дневной EMA (`dailyEMANow`/`dailyEMAPast`/
  `dailyTrendKnown`), `todayRange`.
- В `decide`/`Explain`: ворота кулдауна, дневного тренда, drift-cap, daily-room,
  anti-churn.
- Импорт `math` (использовался только в drift-cap).

`MarketData` (общий тип из пакета `scalping/strategy`) не меняется — momentum
просто перестаёт читать `DailyHighs/DailyLows/DailyCloses`.

## Что добавляется

- Поле `RSICrossLevel float64` в `Params`, дефолт `50`. Уровень, который RSI
  должен пересечь снизу вверх. Калибруемое (попадает в grid).

## Что меняет смысл

- `SignalValidBars int` — теперь **максимальный зазор в барах** между
  MACD-кроссом и RSI-кроссом. `0` означает, что оба сигнала должны произойти на
  одном баре.

## Что сохраняется

- `MACDBelowZeroOnly` — остаётся опциональным фильтром, применяемым к MACD-кроссу
  (кросс «считается», только если `MACDBelowZeroOnly==0 || macdNow<0`).
- `RSIPeriod` / `RSIOverbought` — RSIPeriod теперь обязателен (>0), питает и вход
  (кросс уровня), и существующий overbought-выход. Если `RSIPeriod==0`, вход
  невозможен (сигнал RSI не вычисляется) — это считается ошибкой конфигурации.
- Весь блок управления позицией: стоп (`SLMult`/`SwingLowWindow`), фиксированный
  TP (`TakeProfitRR`/`MinRR`), chandelier-трейл (`UseTrail`/`TrailMult`/
  `ChandelierWindow`/`TrailArmATR`), MACD-выход (`UseMACDExit`), RSI-overbought
  выход. Без изменений.

## Логика входа (pure core)

Шелл ведёт два счётчика «баров с момента события», стартующих насыщенными
большим числом (чтобы до первого срабатывания не было ложного спаривания):

- `barsSinceMACDCross` — сбрасывается в 0 на баре, где `crossUp && (MACDBelowZeroOnly==0 || macdNow<0)`.
- `barsSinceRSICross` — сбрасывается в 0 на баре, где RSI пересекает уровень:
  `rsiPrev <= RSICrossLevel && rsiNow > RSICrossLevel`.

Счётчики обновляются **каждый бар, независимо от наличия позиции** (события рынка
не зависят от позиции). На баре, где событие не сработало, счётчик инкрементится
(с насыщением).

Условие входа (edge-триггер — требует свежего события на текущем баре, поэтому
срабатывает однократно):

```
macdFiredNow := crossUp && (MACDBelowZeroOnly==0 || macdNow<0)
rsiFiredNow  := rsiPrev <= RSICrossLevel && rsiNow > RSICrossLevel

confluence := (macdFiredNow && barsSinceRSICross <= SignalValidBars)
           || (rsiFiredNow  && barsSinceMACDCross <= SignalValidBars)
```

Вход совершается на баре **второго** из пары сигналов. Симметрия «вперёд/назад»
от MACD-кросса обеспечивается автоматически: если MACD приходит позже — на его
баре проверяется свежесть RSI; если RSI приходит позже — на его баре проверяется
свежесть MACD.

После выполнения `confluence` проверяются оставшиеся ворота (в этом порядке):

1. **Тренд:** `emaTrend > 0 && price > emaTrend`.
2. **Объём:** `volumeOK` (`VolumeConfirmed` по `VolLookback`/`VolMultiplier`).
3. **Риск/RR:** `stop = recentLow - SLMult*ATR`; `risk = price - stop > 0`;
   `target = price + TakeProfitRR*risk`; при `TakeProfitRR>0 && MinRR>0` отклонить,
   если `(target-price) < MinRR*risk`.

Если любое из ворот не пройдено на баре confluence — входа нет, и он **не
откладывается** (механизма ожидания подтверждения объёма/EMA больше нет; окно
`SignalValidBars` относится только к спариванию MACD↔RSI). Это прямое следствие
правила «вход на баре второго сигнала».

## EntryReason (для отчёта)

Текст причины входа перестраивается: убирается дневной ATR-запас, добавляется
описание двух сигналов с их возрастом в барах. Пример:

```
Тренд↑ (close X > EMA200 Y); MACD бычий кросс (под нулём, M) N1 бар(ов) назад;
RSI пересёк 50↑ (A→B) N2 бар(ов) назад, зазор ≤ SignalValidBars;
объём > 1.0×ср(20); SL=…(-…); TP=…(+…, 2R)
```

где `N1 = barsSinceMACDCross`, `N2 = barsSinceRSICross` (один из них равен 0 —
«сейчас»), `A→B = rsiPrev→rsiNow`.

`Explain` переписывается под новый порядок ворот: тренд → MACD/RSI confluence →
объём → риск/RR. Секции кулдауна, дневного тренда, drift-cap, daily-room и
anti-churn удаляются.

## Радиус правок

Код:
- `internal/service/trading_strategy/momentum/strategy/core/core.go` — основная
  переработка (Params, шелл-состояние, `buildInput`, `decide`, `Explain`,
  `entryReason`).
- `internal/service/trading_strategy/momentum/strategy/core/core_test.go` —
  переписать тесты входа (удалить кулдаун/daily/drift/anti-churn, добавить
  confluence-кейсы: пары в окне, вне окна, на одном баре, edge-однократность,
  блок по EMA/объёму/RR).

Дефолты и реестр:
- `internal/service/backtest/momentum_registry.go` — `genericMomentumDefaults`:
  убрать удалённые поля, добавить `RSICrossLevel`, гарантировать `RSIPeriod>0`.
- `internal/service/backtest/momentum_registry_test.go` — обновить ожидания.

Per-ticker дефолты (8 файлов): `rusal`, `gazp`, `nvtk`, `mdmg`, `afks`, `sber`,
`plzl`, `ydex` — убрать удалённые поля, добавить `RSICrossLevel`, проставить
`RSIPeriod>0`.

Калибровочные сетки (`data/params/<ticker>/momentum_grid.json`) — переписывает
пользователь отдельно (см. раздел «Калибровочные сетки (grid)»).

Отчётность:
- `internal/service/backtest/basket.go` — в `paramSummary` заменить ключи
  `CooldownBars`/`DailyTrendPeriod` на `SignalValidBars`/`RSICrossLevel`.

## Калибровочные сетки (grid)

Сам механизм перебора параметров через grid **сохраняется без изменений**:
запуск `-calibrate data/params/<ticker>/momentum_grid.json`, структура фаз
(`phases`/`name`/`keepTop`), параллельный перебор (`calibWorkers`) и логика
отбора лучших комбинаций остаются как есть. Перебор по-прежнему свипает все поля
`Params`, перечисленные в сетке.

Меняется лишь *множество свипаемых ключей* и их значения:

- Удалённые поля (`CooldownBars`, `MaxDailyATRUsed`, `MaxDriftATR`,
  `DailyTrendPeriod`, `MinATRFrac`, `DailyATRPeriod`) больше не существуют в
  `Params`, поэтому не могут присутствовать в сетках — ни одна сетка
  `data/params/<ticker>/momentum_grid.json` не должна на них ссылаться.
- Новые входные ключи (`RSICrossLevel`, обязательный `RSIPeriod`) и
  переосмысленный `SignalValidBars` становятся свипаемыми.

Конкретные диапазоны свипа и значения по каждому тикеру задаёт пользователь и
переписывает сетки сам — эта спека их не фиксирует.

## Тестирование

Table-driven тесты в `core_test.go` покрывают:

- MACD и RSI на одном баре (зазор 0) при `SignalValidBars>=0` → вход.
- RSI раньше, MACD позже в пределах окна → вход на баре MACD.
- MACD раньше, RSI позже в пределах окна → вход на баре RSI.
- Зазор больше `SignalValidBars` → входа нет.
- `SignalValidBars==0` и зазор 1 → входа нет.
- Edge-однократность: после входа повторного входа на следующем баре без свежего
  события нет.
- confluence есть, но цена ≤ EMA → блок.
- confluence есть, но объём ниже среднего → блок.
- confluence есть, но `risk<=0` или RR ниже `MinRR` → блок.
- `MACDBelowZeroOnly==1` и MACD-кросс над нулём → MACD-сигнал не взводится.

Команда проверки: `go test ./internal/service/trading_strategy/momentum/... ./internal/service/backtest/...`
плюс `go build ./...`.

## Явное поведенческое следствие

Вход стал edge-триггером на баре второго сигнала. Старое поведение «взвести
сигнал по MACD-кроссу и ждать подтверждения объёма до `SignalValidBars` баров»
заменено. Теперь `SignalValidBars` — это только окно спаривания MACD↔RSI, а
объём/EMA/RR проверяются строго на баре confluence.
