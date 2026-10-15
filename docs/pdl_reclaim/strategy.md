# pdl_reclaim — возврат над вчерашним лоем на 5m

Бэктест-стратегия, только лонг, внутридневная. Прогон **только на 5m** (`-interval Minutes5`, другие cmd/backtest отклоняет). Уровни — лой (PDL) и хай (PDH) вчерашнего буднего дня: PDL — поддержка, у которой ищется вход, PDH — цель. Позиция через ночь не переносится. Код: `internal/service/trading_strategy/pdl_reclaim/strategy/core`, привязка к бэктесту — `internal/service/backtest/pdl_reclaim_registry.go`. Спека: `docs/superpowers/specs/2026-10-15-pdl-reclaim-design.md`.

## Уровни

- **PDL / PDH** — low и high последней завершённой будней дневки (выходные дневки отброшены; в понедельник это пятница). Дневка биржевая — со всеми сессиями дня.
- **Дневной ATR** — Уайлдер `DailyATRPeriod` по завершённым будним дневкам; уходит в сигнал и замораживается на позицию.
- Нет дневок для уровней или ATR — входов нет.

## Вход

На закрытии 5m свечи без открытой позиции, общие условия:

- будни МСК, свеча начинается не раньше `EntryFromHour:EntryFromMinute` (07:00);
- свеча покупки (следующая) начинается раньше `EntryToHour:EntryToMinute` (18:00) в ту же дату — последний сигнал на свече 17:50;
- при включённой цели close свечи ниже PDH;
- **RSI-фильтр:** RSI(`RSIPeriod`) по 5m close на сигнальной свече не выше `RSIEntryMax` (50); свечей для RSI не хватает — входа нет;
- стоп `close − StopDailyATR · ATR` выше нуля.

**Вход A — ложный пробой** (`EntryA = 1`): свеча зелёная, low < PDL, close > PDL.

**Вход B — поглощение под PDL** (`EntryB = 1`): предыдущая свеча той же даты красная и закрылась ниже PDL; текущая зелёная, её close выше open красной, её low ниже low красной.

Обе ветки сразу — причина `A`. Покупка по open следующей 5m свечи (`Config.EntryAtNextOpen`, `FillBy` = время свечи + 5m). Сигнал несёт стоп (для `-risk-pct`), цель PDH и ATR. После выхода перевход в тот же день разрешён.

## Выход

По приоритету на каждой 5m свече:

1. **SL** — `low ≤ вход − StopDailyATR · ATR` (стоп от цены исполнения, ATR замороженный). Исполнение `min(стоп, open)`.
2. **TP** — `high ≥ PDH` (заморожен на входе). Исполнение `max(PDH, open)`. Стоп и цель в одной свече — стоп.
3. **RSI** — RSI(`RSIPeriod`) по 5m close ≥ `RSIExit`. Продажа по close.
4. **MACD** — гистограмма MACD(`MACDFast`, `MACDSlow`, `MACDSignal`) по 5m close была зелёной (`≥ 0`) хотя бы на одной свече от свечи входа до предыдущей, а на текущей красная. Продажа по close. Зелёная уже на свече входа — считается.
5. **EOD** — первая свеча, начавшаяся не раньше `ExitHour:ExitMinute` (18:30), продаётся по close; первая свеча новой даты или свеча даты позже входа — тоже.

RSI в `pkg/indicators` на отрезке без единого падения даёт 50, а не 100: на ровном росте RSI-выход не сработает, пока не будет хоть одного отката.

## Параметры

| поле | по умолчанию | смысл |
|---|---|---|
| `EntryA` | 1 | вход A; 0 — выключен |
| `EntryB` | 1 | вход B; 0 — выключен |
| `StopDailyATR` | 0.5 | стоп ниже входа в дневных ATR; 0 — входов нет |
| `DailyATRPeriod` | 14 | длина дневного ATR Уайлдера по будням |
| `TakeProfitPDH` | 1 | цель PDH; 0 — без цели |
| `RSIPeriod` | 6 | длина RSI на 5m — общая для фильтра входа и выхода; 0 — оба выключены |
| `RSIEntryMax` | 50 | RSI сигнальной свечи выше этого — входа нет; 0 — фильтр выключен |
| `RSIExit` | 70 | уровень RSI-выхода; 0 — выключен |
| `MACDFast` / `MACDSlow` / `MACDSignal` | 12 / 26 / 9 | MACD выхода на 5m; некорректные периоды — выключен |
| `EntryFromHour` / `EntryFromMinute` | 7 / 0 | сигнальная свеча не раньше |
| `EntryToHour` / `EntryToMinute` | 18 / 0 | свеча покупки раньше |
| `ExitHour` / `ExitMinute` | 18 / 30 | выход в конце дня |

`Lookback = max(64, 3 · (MACDSlow + MACDSignal), 4 · RSIPeriod + 2) + 139` 5m свечей (139 — свечи от 07:00 до 18:30): при дефолтах 244.

## Пакеты тикеров

Реестр `pdlReclaimRegistry` держит по одному литералу `Params()` на откалиброванный тикер (`internal/service/trading_strategy/pdl_reclaim/strategy/<ticker>/`); тикер без пакета работает на `core.DefaultParams()`; JSON `-params` / `-calibrate` ложится поверх. Стартовая сетка — `data/params/pdl_reclaim/grid_example.json`.

## Вселенная для live

Живого раннера нет; вселенная под него — `PDL_RECLAIM_TICKERS_M5` (`config.PDLReclaimConfig`). Тест `TestPDLReclaimLiveTickersAreRegistered` требует пакет в `pdlReclaimRegistry` у каждого тикера из `env/prod.env`.

## Запуск

```bash
go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes5 -ticker GAZP -months 36
go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes5 -ticker GAZP -months 36 \
  -calibrate data/params/pdl_reclaim/grid_example.json -metric profit_factor
# почему на свече был (или не был) вход/выход
go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes5 -ticker GAZP -months 36 -explain "2026-09-03 10:00"
```
Для калибровочных прогонов — собранный бинарь на одном ядре.
