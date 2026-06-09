# Интеграция YDEX и PLZL в momentum-стратегию

- Дата: 2026-06-09
- Ветка: feat/momentum-daily-trend-cooldown
- Статус: согласован

## Цель

Подключить тикеры **YDEX** (Яндекс) и **PLZL** (Полюс) к hourly momentum-стратегии
тем же способом, что уже сделан для RUAL и AFKS: отдельный пакет-тикер с
константой `Ticker` и `DefaultParams()`, регистрация в реестре momentum и
калибровочная сетка в `data/params/<ticker>/`. После мержа пользователь
прогоняет `-calibrate` и хардкодит winner — калибровка вне scope этой итерации.

## Контекст / как устроено сейчас

- Тикер резолвится в инструмент через API `Shares()` по имени
  (`cmd/backtest/main.go:resolveShare`) — хардкода FIGI нет, бэктест уже сейчас
  гоняется на любом тикере через generic-fallback.
- Каждый калиброванный тикер — отдельный пакет:
  `internal/service/trading_strategy/momentum/strategy/{rusal,afks}/` с
  `const Ticker` и `func DefaultParams() core.Params`.
- Реестр `internal/service/backtest/momentum_registry.go` мапит
  `Ticker -> Binding`; незарегистрированные тикеры получают
  `genericMomentumDefaults()` через `MomentumLookupOrGeneric`.
- Калибровочная сетка лежит в `data/params/<ticker>/momentum_grid.json`.
  Все существующие гриды (rusal, afks, ydex) идентичны — стандартная сетка на
  11664 комбинации.

## Архитектура

Ядро движка (`core`), резолвинг тикера, кэш свечей и пламбинг реестра —
**без изменений**. Добавляем два пакета-тикера, регистрируем их и докладываем
недостающий грид для PLZL.

## Компоненты

1. **`momentum/strategy/ydex/ydex.go`** (новый)
   - `const Ticker = "YDEX"`
   - `func DefaultParams() core.Params` — пока некалиброванные значения, равные
     generic-бейзлайну (`genericMomentumDefaults`). Doc-комментарий помечает их
     как uncalibrated и предупреждает про релистинг Яндекса (~авг 2024):
     калибровать на чистом окне после релистинга, иначе гэп исказит метрики.

2. **`momentum/strategy/plzl/plzl.go`** (новый)
   - `const Ticker = "PLZL"`
   - `func DefaultParams() core.Params` — то же, generic-бейзлайн. Комментарий
     предупреждает про сплит Полюса 1:10 (2024): калибровать на чистом окне
     после сплита.

3. **`internal/service/backtest/momentum_registry.go`** (правка)
   - импорт `momentumydex` и `momentumplzl`
   - две строки в `momentumRegistry`:
     `momentumydex.Ticker: momentumBindingFor(...)`, аналогично для plzl.

4. **`data/params/plzl/momentum_grid.json`** (новый)
   - копия стандартной сетки (грид `ydex` уже существует).

## Поток данных / поведение

Не меняется. До калибровки YDEX/PLZL отдают захардкоженный generic-бейзлайн;
позже пользователь заменяет `DefaultParams` на winner. `-calibrate` можно
гонять сразу после мержа (сетки на месте).

## Замечание по качеству данных (переносится в doc-комментарии кода)

И YDEX (релистинг ~авг 2024), и PLZL (сплит 1:10 2024) имеют разрывы
непрерывности свечей. Калибровать обязательно на окне, начинающемся ПОСЛЕ
события (рантайм-флаг `-months`), иначе движок посчитает искусственный гэп за
реальное движение. Это прямо отражено в комментариях `DefaultParams`.

## Тестирование

- `go build ./...` и `go vet ./...`.
- Расширить `internal/service/backtest/momentum_registry_test.go`: два теста
  по образцу RUAL/AFKS — проверяют, что `MomentumLookupOrGeneric("YDEX"/"PLZL")`
  возвращает ожидаемые `DefaultParams` и что `Build(...).Ticker()` совпадает.
- Существующий `TestGenericMomentumDefaultsAreFrozenBaseline` остаётся зелёным
  (generic-бейзлайн не трогаем).

## Вне scope

- Запуск `-calibrate` и хардкод winner-параметров.
- Выгрузка/проверка свечей на гэпы.
- Любой рефакторинг ядра или реестра.
