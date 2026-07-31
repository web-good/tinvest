# rsi_pullback: выход по ATR-трейлингу

Дата: 2026-07-31
Стратегия: `internal/service/trading_strategy/rsi_pullback` (backtest-only, 30m, лонг)

## Задача

Стратегия закрывает позицию первой из трёх причин: SL, TP, восходящий крест RSI через
`RSIUpper`. На откалиброванном под T наборе (отчёт
`reports/T/T_rsi_pullback_Minutes30_20260731_134407.md`, 67 сделок за 4 года) это даёт
структурно перекошенный выход:

- RSI-выход сработал 45 раз из 67, средний результат +740;
- TP (1.5 дневного ATR) сработал 2 раза из 67 — цель практически недостижима, RSI закрывает раньше;
- SL сработал 20 раз, средний результат −1676.

Средний выигрыш 962 против среднего убытка 1606: реальное отношение прибыли к риску около
0.6:1, и вся система держится на win rate 68.7%. При падении WR до 62% profit factor
становится 1.0. Это дефект конструкции выходов, а не вопрос подбора чисел: RSI-выход
срезает движение в самом начале, тогда как стоп забирает полную дистанцию.

Трейлинг-стоп даёт калибровке инструмент удерживать движение вместо фиксации по первому
касанию перекупленности.

## Границы работы

В объёме:

- три новых поля `Params`, метущиеся гридом;
- функция `desiredStop` и её встраивание в `manage()`;
- отключаемость RSI-выхода;
- фаза `trail` в `grid.json` и тематический `cal_trail.json`;
- тесты, обновление `Explain`, шапки пакета и `docs/rsi_pullback/strategy.md`.

Вне объёма:

- модель проскальзывания на стопах (обсуждалась отдельно; движок уже учитывает гэп
  открытия через `min(level, open)`, но не внутрибарные проколы) — отдельная задача,
  затрагивающая все стратегии;
- живой слой: `rsi_pullback` остаётся backtest-only;
- перекалибровка тикеров — она следует за реализацией, но не входит в неё.

## Параметры

```go
UseRSIExit    int     // 1 = RSI-выход включён (default 1); любое другое значение отключает
UseTrail      int     // 1 = ATR-трейл включён (default 0); любое другое значение отключает
TrailDailyATR float64 // трейл = maxFav - TrailDailyATR*dailyATR; 0 отключает
```

`DefaultParams()` получает `UseRSIExit: 1, UseTrail: 0, TrailDailyATR: 0`.

**Дефолты обязаны сохранять текущее поведение побайтово.** Уже откалиброванные наборы по
тикерам (`data/params/rsi_pullback/gazp`, `.../t`) не содержат новых ключей и потому получат
дефолты; если те изменят смысл выходов, все прошлые прогоны станут несопоставимыми молча.
Это проверяется тестом, а не соглашением.

Единица трейла — **дневной ATR, замороженный на входе** (`Position.EntryATR`, куда движок
кладёт `sig.ATR` из `enter()`). Стоп, цель и трейл меряются одной линейкой, зафиксированной
в момент решения. Текущий дневной ATR не годится: уровень трейла дышал бы вместе с ним уже
после входа, и сделка перестала бы быть воспроизводимой из своих же входных данных.
`EntryATR <= 0` отключает трейл целиком — тот же guard, что в `reversion.DesiredStop`.

## Логика выхода

Новая чистая функция в `rsi_pullback/strategy/core`:

```go
// desiredStop returns the protective stop level for an open position and the reason of the
// binding component ("SL" | "TRAIL"), or (0, "") when no stop is enabled. maxFav is the
// monotonic max of closes the trail may trail from; callers pass PrevMaxFavorablePrice (see
// "Анти-lookahead"). dailyATR<=0 disables every price stop outright. Among the active
// components the numerically GREATEST level binds — it is the closest to price, and therefore
// the first one price would touch as it falls.
func desiredStop(p Params, entry, dailyATR, maxFav float64) (float64, string) {
	if dailyATR <= 0 {
		return 0, ""
	}
	level, reason := 0.0, ""
	if p.StopDailyATR > 0 {
		level, reason = entry-p.StopDailyATR*dailyATR, "SL"
	}
	if p.UseTrail == 1 && p.TrailDailyATR > 0 && maxFav > 0 {
		if l := maxFav - p.TrailDailyATR*dailyATR; l > level {
			level, reason = l, "TRAIL"
		}
	}
	if level <= 0 {
		return 0, ""
	}
	return level, reason
}
```

Форма зеркалит проверенную `reversion.DesiredStop`, но работает на своих `Params` и на
дневном ATR.

Порядок выходов в `manage()`: **STOP(SL|TRAIL) → TP → RSI**.

- STOP триггерится внутрибарно по `low <= level` (как сейчас SL) и выигрывает ничью с целью
  на одном баре: внутрибарный порядок двух касаний из OHLC неизвестен, честный выбор —
  худший из двух. `sig.StopLoss` выставляется в уровень, чтобы движок залил по
  `min(level, open)`, а не по close.
- `sig.Reason` — `"SL"` или `"TRAIL"` в зависимости от связавшего компонента.
  `model.IsStopReason` уже содержит `"TRAIL"`, правок движка не требуется.
- RSI-ветка оборачивается в `if s.p.UseRSIExit == 1`.

`ExitReason` для трейла рендерится в том же стиле, что остальные: уровень, максимум, от
которого он отсчитан, и цена входа.

Конфигурация `UseRSIExit=0, UseTrail=0, TPDailyATR=0` оставляет единственным выходом стоп.
Позиция всё равно закрывается (стоп обязателен: `StopDailyATR=0` в гриды не входит и
входить не должен), поэтому отдельный guard не вводится, но комбинация упоминается в
комментарии к фазе `trail`.

## Анти-lookahead

Единственное место, где ошибка даёт незаметно завышенный результат.

Движок в `Run()` (`engine.go:167-173`) вызывает `p.mark(candles[i].Close)` **до** `Decide`.
`MaxFavorablePrice`, который видит стратегия, уже включает close текущего бара. Стоп при
этом триггерится по `low` того же бара, моделируя биржевой стоп-ордер.

Порядок `low` и `close` внутри бара из OHLC неизвестен. Если close случился после low,
реальный ордер в момент прохода low стоял на уровне, ничего не знавшем про этот close.
Считать трейл от `MaxFavorablePrice` — значит выходить по уровню, которого тогда не
существовало.

Смещение систематическое: `MaxFavorablePrice >= PrevMaxFavorablePrice` всегда, поэтому
уровень трейла от MaxFav никогда не ниже честного. Он срабатывает не реже и фиксирует по
цене не хуже — завышает результат в обе стороны сразу.

Пример: entry 100, dailyATR 10, `TrailDailyATR=0.5`, до бара maxFav = 100. Бар: low 96,
close 105.

| источник | уровень | low 96 | итог |
|---|---|---|---|
| `MaxFavorablePrice` = 105 | 100 | 96 <= 100, сработал | выход по 100 |
| `PrevMaxFavorablePrice` = 100 | 95 | 96 > 95, нет | держим, бар закрылся на 105 |

Поэтому трейл читает **`Position.PrevMaxFavorablePrice`**, как задокументировано в
`reversion/strategy/core/core.go:456-462`. Фиксированному SL лаг не нужен: он отсчитывается
от цены входа и не двигается.

## Грид

Новая фаза `trail` в `data/params/rsi_pullback/grid.json`, после `risk` и вместо текущей
финальной фазы `exit` (её `RSIUpper` уходит внутрь новой фазы, чтобы RSI-порог и его
отключение мелись вместе, а не в двух разных фазах):

```json
{
  "name": "trail",
  "grid": {
    "UseRSIExit": [0, 1],
    "RSIUpper": [60, 70, 80],
    "UseTrail": [0, 1],
    "TrailDailyATR": [0.5, 0.8, 1.2]
  }
}
```

При `UseTrail=0` значения `TrailDailyATR` схлопываются в один контроль, при `UseRSIExit=0`
— значения `RSIUpper`; дубли в лидерборде ожидаемы, ровно как в фазе `volume`.

Плюс тематический `data/params/rsi_pullback/cal_trail.json` по образцу остальных `cal_*.json`
(комментарий с назначением, командой запуска и предупреждениями). Он метёт ту же тему
отдельно, поверх выживших после `cal_risk.json` наборов.

Существующий `cal_exit.json` остаётся как есть — он метёт только `RSIUpper` и от новых полей
не зависит.

## Тесты

TDD: красный тест до кода.

Таблица на `desiredStop`:

- только SL (`UseTrail=0`);
- трейл ниже SL — связывает SL;
- трейл выше SL — связывает TRAIL;
- `TrailDailyATR=0` при `UseTrail=1`;
- `dailyATR <= 0` — возвращает `(0, "")`;
- `maxFav = 0` — трейл не участвует;
- `StopDailyATR=0` и трейл выключен — возвращает `(0, "")`.

Поведенческие на `manage()`:

- `TestExitTrailFiresOnLow` — срабатывание по low, `Reason="TRAIL"`, `sig.StopLoss` равен
  уровню;
- `TestTrailReadsPrevMaxFavorable` — регресс ровно на сценарий из раздела «Анти-lookahead»;
  падает при подмене на `MaxFavorablePrice`;
- `TestTrailNeverGoesBelowSL` — при широком `TrailDailyATR` уровень не опускается ниже
  фиксированного стопа;
- `TestTrailDisabledWithoutEntryATR` — `EntryATR <= 0` отключает трейл;
- `TestTrailWinsOverTakeProfitOnTheSameBar` — приоритет стопа над целью сохраняется и для
  трейла;
- `TestUseRSIExitZeroKeepsPosition` — RSI пересёк `RSIUpper` вверх, выхода нет;
- `TestDefaultsPreserveLegacyExits` — на `DefaultParams()` выходы ведут себя как до правки;
- `TestTrailIsAStopReason` — `model.IsStopReason("TRAIL")` истинно; связь между стратегией и
  моделью фила неявная, и её легко сломать.

Обновляются: `TestDefaultParams` (три новых поля), `TestExplainMentionsEveryGate` (трейл в
`Explain`).

## Документация

- шапка пакета `core.go`: сейчас сказано «closed on the first of: the stop, the target, or
  RSI crossing UP» — добавить трейл и отключаемость RSI-выхода;
- `docs/rsi_pullback/strategy.md`: секция выходов, таблица параметров, описание фазы `trail`.

## Критерий готовности

- `./bin/mage ci` зелёный (lint + `go test -race ./...` + проверка дрейфа моков);
- прогон по T на наборе параметров из отчёта
  `reports/T/T_rsi_pullback_Minutes30_20260731_134407.md` (RSIPeriod=5, RSILower=20,
  RSIUpper=65, EMAFast=20, EMASlow=100, StopDailyATR=0.5, TPDailyATR=1.5, UseVolume=0 —
  новые поля не заданы и потому берут дефолты) воспроизводит его побайтово: 67 сделок,
  PF 1.312, ни одного выхода `TRAIL`. Это доказательство, что дефолты не сдвинули поведение
  ранее откалиброванных наборов;
- прогон с `UseTrail=1` даёт отличающийся журнал сделок с непустым числом выходов `TRAIL`.

Калибровка и walk-forward идут отдельной задачей после того, как это станет зелёным.
Планка приёмки прежняя: pooled OOS profit factor >= 1.5 при >= 30 сделках, иначе стратегия
закрывается, как `orb` и `vwap_rev`.
