# rsi_pullback: второй вариант входа (zone) вместо отдельной стратегии rsi_zone

Дата: 2026-09-30. Статус: утверждён в диалоге, ждёт ревью текста владельцем.

## Зачем

rsi_zone и rsi_pullback — почти одна стратегия: обе покупают крест короткого RSI вниз через нижнюю
полосу в восходящем тренде и продают по стопу в дневных ATR или по кресту RSI вверх. Отдельная
стратегия потребовала отдельного live-адаптера, владения тикером на общем счёте, приоритета входа и
своей калибровки — и ни на одном тикере не показала устойчивого преимущества (сводка — в памяти
проекта, `project_rsi_zone.md`).

Владелец решил: zone становится **вторым вариантом покупки** внутри rsi_pullback. Каждый тикер
сам решает, включён ли у него этот вариант. Настройки входа pullback и настройки входа zone — два
отдельных блока параметров тикера. Zone отвечает только за вход; выходы у любой позиции —
выходы rsi_pullback, как сейчас.

## Решения владельца

- Отдельная стратегия rsi_zone удаляется целиком: ядро, тикерные пакеты, live-адаптер, механизм
  гостей в раннере, бэктест-регистрация, скринер, команда калибровки, конфиг и env.
- В этой работе zone не включается ни на одном тикере: только механика. Включение — отдельной
  калибровкой по тикеру (следующая спека). Прежние калибровки rsi_zone не переносятся: они
  подбирались под другие выходы (стоп 1.0 ATR, без цели, RSI-выход 75).

## Критерии готовности

1. При `UseZoneEntry=0` сигналы ядра совпадают с текущими бит-в-бит; `pullparity` по всей боевой
   вселенной — ноль расхождений.
2. При `UseZoneEntry=1` ядро открывает позицию по zone-условиям там, где вход pullback молчит, со
   стопом, целью и трейлом по полям pullback.
3. В репозитории не остаётся кода, конфигурации и env стратегии rsi_zone; живой раннер снова
   ведёт одну стратегию.
4. `./bin/mage ci` зелёный.

## 1. Ядро `rsi_pullback/strategy/core`

### Параметры

`Params` остаётся плоской структурой: калибратор (`internal/service/backtest/calibrate.go`) свипает
поля через `reflect` по имени верхнего уровня. Новый блок идёт последним, под комментарием-заголовком:

```go
// --- вход zone: второй вариант покупки; выходы у позиции общие (блок выше) ---
UseZoneEntry  int     // 1 включает zone-вход; иначе выключен (grid; default 0)
ZoneRSIPeriod int     // длина RSI zone-входа (grid; default 0 — задаётся явно при включении)
ZoneRSILower  float64 // крест RSI вниз через эту полосу — сигнал (grid; default 0)
ZoneEMAPeriod int     // вход только при close > EMA(ZoneEMAPeriod) (grid; default 0)
```

Все четыре поля в `DefaultParams()` нулевые. Причина — тикерные литералы: 33 пакета сверяют
литерал снимком, а восемь (TGKA, CNRU, отвергнутые AFKS, HEAD, RTKM, RTKMP, TRNFP, UWGN)
сравнивают параметры с `core.DefaultParams()` — на равенство или неравенство. Ненулевые
zone-дефолты сломали бы их или потребовали переписать литералы без пользы. Значения задаются явно там, где zone
включают: в сетке калибровки или в литерале тикера. Ориентир для первой сетки — бывшие дефолты
rsi_zone: RSI 4, полоса 25, EMA 200.

Ловушка нулевого значения закрыта дважды:
- ядро отказывает zone-входу, если `ZoneRSIPeriod <= 0`, `ZoneEMAPeriod <= 0` или `ZoneRSILower <= 0`;
- сторожевой тест реестра `internal/service/backtest/rsi_pullback_registry_test.go` (по образцу
  `TestRSIPullbackTickersKeepTheRSIExitArmed`): у любого тикера с `UseZoneEntry == 1` все три
  поля положительны.

### Вход

`enter()` сначала проверяет вход pullback — код и порядок гейтов не меняются. Если pullback
сигнала не дал и `UseZoneEntry == 1`, проверяется zone-вход:

1. будний день (тот же `tradingDay`);
2. `RSISeries(closes, ZoneRSIPeriod)` пересекает `ZoneRSILower` вниз на текущем баре;
3. close текущего бара строго выше прогретой `EMA(ZoneEMAPeriod)` (прогрев: значение EMA > 0);
4. дневной ATR (`DailyATRPeriod`, будние дни) посчитан — иначе нет ни стопа, ни цели;
5. стоп `entry − StopDailyATR·ATR` при включённом стопе выше нуля — та же проверка, что у pullback.

Гейт дня (`UseDayATRGate`) и объёмный гейт (`UseVolume`) на zone-вход **не действуют**: у zone
свой набор условий.

Стоп, цель, `sig.ATR` считаются теми же выражениями и полями, что у pullback (`StopDailyATR`,
`TPDailyATR`, `DailyATRPeriod`); трейл работает от `TrailDailyATR` в `manage()` как обычно.
Сигнал zone-входа ставит `sig.RSI` в значение zone-RSI.

Совпадение на одном баре: вход один, по pullback (его проверка идёт первой). Уровни стопа и
цели у обоих вариантов одинаковы, так что приоритет меняет только текст причины.

Крест zone-входа проверяется тем же помощником `crossedDown`, что у pullback. Разница с
индексным прогревом удаляемого ядра rsi_zone для креста ВНИЗ несущественна: при
`ZoneRSILower > 0` предыдущее значение 0 (прогрев или настоящий RSI 0.00) никогда не бывает
`>= полосы`, так что оба варианта дают одинаковый ответ. Индексный прогрев в rsi_zone был важен
для креста ВВЕРХ (выход), а выход у zone-позиций — выход pullback.

`EntryReason` zone-входа начинается с `zone:` и описывает свои условия (RSI, полоса, close vs
EMA, дневной ATR, вход, стоп, цель, трейл) — в журнале бэктеста и в Telegram видно, каким
вариантом открыта сделка.

### Выходы

`manage()` и `DesiredStop()` не меняются: стоп/трейл → цель → RSI-выход по `RSIPeriod`/`RSIUpper`
pullback. Позиция не хранит вариант входа — выходам он не нужен, и live-стейт не меняется.

### Lookback и Explain

`Lookback()` учитывает `ZoneEMAPeriod` и `ZoneRSIPeriod` только при `UseZoneEntry == 1`: у
выключенного zone окно свечей остаётся прежним (критерий 1, и лишний прогрев не нужен).

`Explain()` дописывает строки zone-гейтов: выключен / крест RSI / close vs EMA.

## 2. Удаление rsi_zone и откат раннера

### Живой раннер — откат к 7d20d99

Поддержку нескольких стратегий на счёте (адаптер `livecore/adapter`, владелец позиции в стейте,
приоритет входа, «бумажная стратегия при боевой соседке», гости вариадиком `NewService`,
вынос `livecore/rebuild`) добавили коммиты a9b20f4…f369b65, и больше ничего в этих файлах они не
меняли. Файлы возвращаются к состоянию 7d20d99 — версии, которая работает в проде — **новым
коммитом**, main не переписывается:

- `internal/service/trading_strategy/rsi_pullback/live/` — `live.go`, `pass.go`,
  `reconstruct/reconstruct.go`; удаляются `adapter.go`, `shared_test.go`;
- `internal/service/trading_strategy/livecore/statestore/` — без поля `Strategy`;
  удаляются `livecore/adapter/`, `livecore/rebuild/`;
- `cmd/pullparity/` — без флага `-strategy`;
- `internal/service_provider/`, `internal/app/init_config.go`, `internal/config/config.go`,
  `internal/config/telegram_client.go`; удаляются `internal/config/rsi_zone.go` и тест;
- `env/prod.env`, `env/prod.env.example`, `env/local.env.example` — без блока `RSI_ZONE_*` и
  `TELEGRAM_TOPIC_RSI_ZONE`;
- `.golangci.yml` — без исключения misspell `strat` (оно было для поля слота раннера);
- `docs/rsi_pullback/live.md` — без абзацев про общий счёт.

Поле `strategy` в прод-стейт не попадало: origin/main на 0f08771, код общего счёта не выкатывался.
Если бы попало — `encoding/json` неизвестное поле игнорирует.

Если сборка после отката требует правок из-за изменений вне этих файлов (после 7d20d99 их быть не
должно), правится минимально и отмечается в коммите.

### Удаляется целиком

- `internal/service/trading_strategy/rsi_zone/` — core, тикеры afks/baza/dias/domrf/lent, live;
- `internal/service/backtest/`: `rsi_zone_registry.go` и тест, `zone_screen.go`,
  `zone_screen_report.go`, `zone_screen_test.go`, `rsi_zone_grid_test.go` и
  `rsi_zone_*_grid_test.go`;
- регистрация `-strategy rsi_zone` в `cmd/backtest/main.go`;
- `cmd/zonescreen/`;
- `data/params/rsi_zone/`, `docs/rsi_zone/`, `.claude/commands/rsi-zone-calibrate.md`.

Спеки и планы rsi_zone в `docs/superpowers/` остаются как история.

### Остаётся

- `internal/service/backtest/screenrun` — им пользуется `cmd/pullscreen`;
- маппинг `MinPriceIncrement` в конвертере (фикс ревью скринера) — общий, от него зависят
  live-стопы.

### Документация

- `docs/rsi_pullback/strategy.md` — раздел «Второй вход (zone)»: условия, поля, отсутствие гейтов
  дня и объёма, общие выходы, приоритет pullback, нулевые дефолты. Только механика (правила
  CLAUDE.md): без замеров и тикеров.
- `CLAUDE.md` — абзац rsi_zone в Layout убирается; в строке rsi_pullback — упоминание второго
  входа.

## 3. Тесты

Ядро — TDD, в `rsi_pullback/strategy/core/core_test.go`:

- `UseZoneEntry=0` при заполненных zone-полях — сигнал совпадает с сигналом без zone-полей
  (сторож критерия 1);
- zone-вход срабатывает, когда EMA fast ≤ slow (pullback молчит), а close > EMA(ZoneEMAPeriod);
- гейт дня и объёмный гейт, закрытые для pullback, zone-вход не блокируют;
- стоп и цель zone-входа — от `StopDailyATR`/`TPDailyATR` и дневного ATR;
- совпадение на одном баре — причина pullback, не `zone:`;
- нулевой `ZoneRSIPeriod`/`ZoneEMAPeriod`/`ZoneRSILower` при `UseZoneEntry=1` — входа нет;
- close ≤ EMA или непрогретая EMA — входа нет; выходной день — входа нет; ATR = 0 — входа нет;
- `Lookback()` растёт только при `UseZoneEntry=1`;
- позиция, открытая zone-входом, закрывается по правилам pullback (SL/TP/RSI).

Каждый тест проверяется мутацией: отключённое условие в коде должно ронять свой тест.

Сторожевой тест реестра — нулевые zone-поля при `UseZoneEntry=1`.

Паритет: `go run ./cmd/pullparity` по боевой вселенной — ноль расхождений (живой раннер зовёт
тот же `core.Decide`, отдельного live-пути у zone нет). Движковый тест бэктеста с
`UseZoneEntry=1` на синтетических свечах: zone-сделка открывается и закрывается через `domain.Run`.

Итог — `./bin/mage ci`: lint, `go test -race ./...`, дрейф моков (моки адаптера исчезают вместе
с ним).

## Вне объёма

- Процедура калибровки zone-входа на тикере rsi_pullback — замена `/rsi-zone-calibrate`
  (следующая спека).
- Включение zone на любом тикере.
- Раздельные настройки выходов для позиций, открытых zone-входом.
