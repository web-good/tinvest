# Дивидендный фундаментальный скринер + интеграция в Golden X Score

**Дата:** 2026-07-20
**Статус:** утверждён к реализации

## Проблема и цель

Golden X — технический тайминг (перепроданность по недельному RSI), он отвечает на вопрос «КОГДА покупать», но ничего не знает о качестве бизнеса — «ЧТО стоит держать». У пользователя нет инструмента, который бы находил среди всей Московской биржи самые перспективные компании с точки зрения **долгосрочного дивидендного инвестора** и ранжировал их по фундаментальному качеству.

Tinkoff Invest API отдаёт богатую фундаменталку одним RPC (`GetAssetFundamentals`, батчи ≤100 asset-uid): дивидендная доходность и её история, payout ratio, NetDebt/EBITDA, ROIC/ROE, EV/EBITDA, рост дивиденда/выручки, FCF, маржа и т.д.

**Цель:** построить дискавери-скринер по всей бирже, который ранжирует компании по композитному дивидендному качеству, доступен как бот-команда, и чей рейтинг **питает `Score` в Golden X** — чем выше компания в рейтинге, тем больше очков её сигналу на покупку.

## Философия ранжирования

Главный принцип — **НЕ ранжировать по дивидендной доходности**. Высокая доходность чаще всего yield trap: цена рухнула, потому что рынок закладывает срезание дивиденда. Долгосрочному инвестору нужен *устойчивый растущий доход от финансово прочного бизнеса, купленного по разумной цене*. Рейтинг — взвешенный композит шести столпов, активно **штрафующий** красные флаги.

### Шесть столпов (в порядке веса)

1. **Устойчивость выплаты (наибольший вес)** — «смогут ли платить дальше?». `DividendPayoutRatioFy`: сладкая зона ~30–60%; >80% хрупко; >100% — красный флаг. Плюс покрытие дивиденда свободным денежным потоком (`FreeCashFlowTtm` vs `DividendRateTtm`) — для сырьевых РФ надёжнее бухгалтерской прибыли.
2. **Долговая безопасность (высокий вес).** `NetDebtToEbitda` — ключевая для РФ: <1 отлично, 1–2 норм, 2–3 осторожно, >3 опасно, отрицательный (чистый кэш) — лучший. Вторично: `TotalDebtToEquityMrq`, `FixedChargeCoverageRatioFy`, `CurrentRatioMrq`.
3. **Рост дивиденда (средний вес).** `FiveYearAnnualDividendGrowthRate`; сравнение текущей форвард-доходности с `FiveYearsAverageDividendYield`.
4. **Качество бизнеса (средний вес).** `Roic`/`Roe`, `NetMarginMrq`, EBITDA-маржа (`EbitdaTtm`/`RevenueTtm`).
5. **Оценка (меньший вес, страж).** `EvToEbitdaMrq` (главный межсекторный мультипликатор), `PeRatioTtm`, `PriceToBookTtm`, `PriceToFreeCashFlowTtm`. Оценка **никогда не перевешивает качество** (защита от value trap).
6. **Текущая доходность (скромный плюс, НЕ драйвер).** `ForwardAnnualDividendYield`/`DividendYieldDailyTtm` награждаем умеренно и **с потолком** (сверх ~14% доп. баллов нет). Если доходность экстремально высокая И (payout>100% ИЛИ NetDebt/EBITDA высокий ИЛИ FCF<0) → **yield-trap-флаг**, штраф/дисквалификация, а не награда.

### Метод комбинирования: гибрид «ворота + относительный ранг»

Повторяет философию существующего bonds-скринера (reliability-policy гейт + ранг по YTM):

1. **Ворота (gate).** Выкинуть заведомо негодных: нет дивиденда (`DivYieldFlag`=false или yield=0); `NetDebt/EBITDA > 4`; payout > 120%; yield-trap-флаг; отсутствуют ключевые поля (NetDebt/EBITDA, payout, yield). У каждой отсеянной компании фиксируется `GateReason`.
2. **Ранг выживших.** Для каждой метрики — **перцентиль по вселенной** (робастно к выбросам), ориентируем «больше-лучше», взвешиваем по столпам, суммируем в `Composite` 0–100.
3. **Мост в Golden X.** `Composite` → перцентильный ранг внутри вселенной → **ограниченный бонус 0..3**.

Почему гибрид: ворота убирают болезнь относительного ранжирования («лучший из мусора»), перцентильный ранг выживших даёт устойчивое к выбросам, объяснимое упорядочивание.

## Архитектура

```
                    ┌─────────────────────────────────────────┐
                    │  screener/dividend/rank  (ЧИСТОЕ ЯДРО)   │
                    │  gate → перцентили → 6 столпов →         │
                    │  composite 0–100 + rank + trap-флаг      │
                    └───────────────▲─────────────────────────┘
                                    │ []Fundamentals
        ┌───────────────────────────┴───────────────┐
        │  screener/dividend  (оркестрация)          │
        │  Shares()→rub+DivYieldFlag→AssetUid→        │
        │  GetAssetFundamentals(батчи)→rank.Rank()   │
        └───────────────▲────────────────────────────┘
                        │ refresh (TTL 24ч)
        ┌───────────────┴───────────────┐
        │  RankProvider  (кэш, RWMutex)  │
        │  • Top(n)          → команда   │
        │  • RankBonus(id)→0..3 → GoldenX│
        └──────▲──────────────────▲──────┘
               │                  │
   ┌───────────┴──────┐   ┌───────┴──────────────────────┐
   │ /dividend_screener│   │ golden_x trade.go:            │
   │ (telegram_commands)│  │  bonus=provider.RankBonus(id) │
   │ → notification    │   │  → Classify(...,bonus)        │
   └───────────────────┘   │  → signalScore += bonus       │
                           └───────────────────────────────┘
```

### Компоненты

**1. Данные-фундамент.**
- `model.Share` +`AssetUid string` +`DivYieldFlag bool`; `converter.ConvertShareFromPb` мапит оба поля (аддитивно, не ломает существующее).
- Новый `model.Fundamentals` — подмножество полей `GetAssetFundamentalsResponse_StatisticResponse`, что реально используются: AssetUid, ForwardAnnualDividendYield, DividendYieldDailyTtm, DividendPayoutRatioFy, FiveYearsAverageDividendYield, FiveYearAnnualDividendGrowthRate, DividendRateTtm, NetDebtToEbitda, TotalDebtToEquityMrq, FixedChargeCoverageRatioFy, CurrentRatioMrq, Roic, Roe, NetMarginMrq, EbitdaTtm, RevenueTtm, FreeCashFlowTtm, EvToEbitdaMrq, PeRatioTtm, PriceToBookTtm, PriceToFreeCashFlowTtm.
- Обёртка `GetAssetFundamentals(ctx, assetUIDs []string) ([]*model.Fundamentals, error)` на instruments-grpc-клиенте: разбивает вход на батчи по 100, склеивает ответы, конвертит в `model.Fundamentals`. Regen мока (`./bin/mage mocks`).

**2. Чистое ядро `internal/service/screener/dividend/rank`.**
- Вход: `[]model.Fundamentals` (вселенная).
- Выход: `[]ScoredCompany{AssetUid, Composite float64, PillarScores, YieldTrap bool, GateReason string}` (отсеянные — с `GateReason`, ранжированные — отсортированы по `Composite` убыв.).
- Никакого I/O, никакой зависимости от времени. Веса столпов и пороги ворот — именованные константы/конфиг-структура (аналог `golden_x/model.Settings`), чтобы задача 6 могла их калибровать.
- Перцентильный хелпер: переиспользовать R-7 логику. `golden_x/percentile.r7` не экспортирована — извлечь общий перцентиль в `pkg/indicators` (или локальная копия), чтобы не плодить дубли; решение — в задаче 2.
- Тесты table-driven: каждое правило ворот, yield-trap, порядок ранга, поведение при отсутствующих полях.

**3. Оркестрация `internal/service/screener/dividend`.**
- Держит instruments-grpc-клиент.
- `Shares()` → фильтр `Currency=="rub"` уже в конвертере; доп. фильтр `DivYieldFlag` → собрать `AssetUid` → `GetAssetFundamentals` батчами → `rank.Rank()`.
- Считает и **возвращает видимую статистику отсева**: сколько компаний во вселенной, сколько отсеяно и по каким `GateReason` (урок bonds-скринера: отсев должен быть виден, не молчком — [[project-bonds-screener-quality]]).

**4. `RankProvider` — общий потокобезопасный кэш (шов между двумя потребителями).**
- Держит последний результат ранжирования + timestamp, `sync.RWMutex`.
- Ленивое обновление при доступе, если старше TTL (24ч); фундаменталка меняется раз в квартал, тики Golden X (раз в несколько часов) кэш не дёргают.
- `Top(n int) []ScoredCompany` — для команды.
- `RankBonus(instrumentID string) int` — для Golden X: маппинг перцентильного ранга компании → **0..3** (топ-дециль→+3, топ-квартиль→+2, топ-половина→+1, иначе/неизвестно/отсеяна/кэш пуст→0). Ключ — instrument uid (`model.Share.ID`); провайдер держит map instrumentID→ScoredCompany.

**5. Бот-команда `/dividend_screener`.**
- В `internal/service/telegram_commands/commands.go` новый case + строка в help-тексте.
- Читает `provider.Top(N)`, рендерит через новый `screener/dividend/notification`: топ-N c `Composite`, yield, NetDebt/EBITDA, payout, EV/EBITDA, ROIC + бейджи (⚠️ trap-флаг), и хвост со статистикой отсева. Стиль — зеркало bonds-скринера.
- Проводка в `internal/service_provider`.
- Тесты: golden-string рендера.

**6. Интеграция в Golden X.**
- Инъекция `RankProvider` в `golden_x.service` (`types.go`, конструктор).
- В `trade.go` для каждой акции: `bonus := provider.RankBonus(share.ID)`; прокинуть в `detector.DetectAll`/`classifier.Classify` как **данные** (per-share map или поле в `DetectResult`), чтобы `Classify` осталась чистой (без вызова провайдера внутри).
- `signalScore` += `bonus` (диапазон Score станет 1..11).
- `ShareResult` +`FundamentalBonus int`; нотификация: строка/легенда объясняет фунд-бонус.
- Обновить `docs/golden_x/{strategy,settings}.md` и легенду.

## Обработка качества данных

Многие акции РФ имеют пропущенные/нулевые фундаментальные поля. Стратегия:
- Ворота отсеивают компании без ключевых полей (NetDebt/EBITDA, payout, yield) с явным `GateReason`.
- Скринер показывает, сколько компаний отсеяно и почему — отсев виден в выводе команды.
- `RankBonus` при отсутствии компании в рейтинге возвращает 0 (без бонуса, а не паника/ошибка).

## Кэширование (обоснование)

Забор фундаменталки по всей бирже (~200+ акций, батчи по 100 = 2–3 RPC + список акций) делается **один раз и кэшируется**. Тики Golden X (раз в несколько часов) НЕ должны его дёргать — читают кэш. Команда читает кэш, обновляя при устаревании (TTL 24ч). Так Golden X остаётся быстрым и дешёвым по API.

## Тестирование

- **Ядро `rank`:** table-driven — каждое правило ворот, yield-trap, порядок ранга по composite, маппинг ранг→бонус, missing-data.
- **Provider:** TTL/staleness, `RankBonus` маппинг, потокобезопасность (race-тест).
- **Notification:** golden-string (как bonds).
- **Classifier:** расширить существующие тесты на поле `FundamentalBonus` в `signalScore`.
- Гейт CI: `./bin/mage ci` (lint + `go test -race ./...` + mock-drift).

## План реализации (6 задач)

Каждая задача — свежее окно + двухэтапное ревью (`superpowers:subagent-driven-development`).

1. **Данные-фундамент:** `model.Share`+2 поля, `model.Fundamentals`, grpc-обёртка `GetAssetFundamentals` (батчи), regen мока. Тесты: конвертер, батчинг.
2. **Чистое ядро `rank`:** ворота + 6 столпов + перцентили + composite. Тесты: каждое правило.
3. **Оркестрация + `RankProvider`:** вселенная→фундаменталка→rank, кэш/TTL, `Top`/`RankBonus`. Тесты: staleness, ранг→бонус, race.
4. **Бот-команда + нотификация:** `/dividend_screener`, рендер top-N, проводка, help. Тесты: golden-string.
5. **Интеграция в Golden X:** бонус в `Classify`/`signalScore`, инъекция провайдера, легенда, доки.
6. **Валидация/полировка:** прогон на живых данных — сколько отсеяно, вменяемость топа; правка весов/порогов ворот при необходимости.

Порядок: 1→2→3 строят фундамент снизу вверх; 4 и 5 — независимые потребители провайдера (можно параллелить); 6 замыкает.

## Явно вне охвата (YAGNI)

- Историческая фундаменталка / тренды метрик во времени (берём последний срез TTM/MRQ).
- Периодический push-дайджест топа (пока только бот-команда по запросу; можно добавить позже).
- Секторные лимиты/веса в самом рейтинге (bonds-скринер их имеет; здесь — потом, если топ окажется перекошен по сектору).
- Бэктест дивидендной доходности рейтинга (нет исторической фундаменталки в API).
