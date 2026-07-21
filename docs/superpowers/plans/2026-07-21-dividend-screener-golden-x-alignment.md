# Dividend Screener Golden X Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Сделать дивидендный ранг полезным для Golden X — отфильтровать вселенную по капитализации (ликвидные торгуемые имена) и превратить фундаментальный бонус из относительного дециля в абсолютные полосы композита, чтобы бонус реально различал имена, которые Golden X торгует.

**Architecture:** Все изменения ранжирования — в чистом ядре `internal/service/screener/dividend/rank/` (без I/O): новое поле капитализации в `model.Fundamentals`, фильтр ликвидности в `gate()`, пороги бонуса в `rank.Config`. Механику бонуса меняем в оркестраторе `service.go` (`bonusFromScore` вместо `bonusFromRank`). Пороги (`MinMarketCap`, `BonusScoreT1/T2/T3`) — точки калибровки, финально подбираются на живых данных одноразовым диагностиком (не коммитим).

**Tech Stack:** Go 1.25, чистые функции + table-driven тесты, gRPC Tinkoff `GetAssetFundamentals` (поле `MarketCapitalization` уже в ответе), `godotenv`, `./bin/mage ci`.

## Global Constraints

- Go 1.25; идиоматичный Go, MixedCaps, ошибки — обёрнутый `%w`.
- Поле называется **`AssetUID`** (не `AssetUid`) — revive var-naming; не переименовывать.
- Поля `Fundamentals` — `proto3 float64` с `omitempty`: «нет данных» и «0.0» неразличимы. Правило: **`0` в fundamental-поле = «нет данных»**, не «реальный ноль». Для капитализации это значит: `MarketCapitalization == 0` трактуется как «неликвид/нет данных» и отсеивается.
- Единицы yield и payout **ПОДТВЕРЖДЕНЫ живой валидацией — проценты**. Единицы `MarketCapitalization` **НЕ подтверждены** (₽ vs млн ₽) — подтверждаются в Task 4 перед фиксацией `MinMarketCap`.
- Все изменения ранжирования — в пакете `rank`; механика бонуса — в `service.go`. `detector`/`classifier` не трогаем.
- `go build ./internal/... ./pkg/... ./cmd/...` (НЕ `./...` — падает на `magefiles`).
- Гейт: `./bin/mage ci` = lint + `go test -race ./...` + mock-drift. Добавление поля в `model.Fundamentals` НЕ меняет мок-интерфейсы — `./bin/mage mocks` не требуется (подтвердить mock-drift проверкой в Task 4).
- Диагностик из Task 4 (`cmd/divcheck`) — **временный, в git НЕ коммитить**, удалить в конце.

---

## File Structure

**Модифицируются:**
- `internal/model/fundamentals.go` — добавить поле `MarketCapitalization float64`.
- `internal/converter/fundamentals.go` — замапить `MarketCapitalization` из pb.
- `internal/converter/fundamentals_test.go` — покрыть новое поле.
- `internal/service/screener/dividend/rank/config.go` — `MinMarketCap`, `BonusScoreT1/T2/T3` в `Config`+`DefaultConfig`.
- `internal/service/screener/dividend/rank/rank.go` — `reasonIlliquid` + ветка в `gate()`.
- `internal/service/screener/dividend/rank/rank_test.go` — `aboveCapFloor`, ретрофит существующих фондов, новый `TestRank_GateIlliquid`.
- `internal/service/screener/dividend/service.go` — `bonusFromScore` вместо `bonusFromRank`.
- `internal/service/screener/dividend/service_test.go` — ретрофит фикстур капой; `TestBonusFromScore`; рефрейм `TestRankBonus_TopGetsMorePoints`.

**Создаётся временно (не коммитить, удаляется в Task 4):**
- `cmd/divcheck/main.go` — одноразовый диагностик живого прогона + калибровки.

---

## Task 1: Поле капитализации в модели и конвертере

**Files:**
- Modify: `internal/model/fundamentals.go`
- Modify: `internal/converter/fundamentals.go`
- Test: `internal/converter/fundamentals_test.go`

**Interfaces:**
- Produces: `model.Fundamentals.MarketCapitalization float64` (рыночная капитализация, валюта инструмента; `0` = нет данных). Читают Task 2 (`gate`) и Task 4 (диагностик).

- [ ] **Step 1: Обновить тест конвертера (RED)**

В `internal/converter/fundamentals_test.go` добавить `MarketCapitalization` в pb-вход, в `want` и в проверки.

В литерал `in[0]` (`investapi.GetAssetFundamentalsResponse_StatisticResponse{...}`) добавить строку сразу после `PriceToFreeCashFlowTtm: 20.20,`:

```go
			MarketCapitalization:             21.21,
```

В литерал `want` (`&model.Fundamentals{...}`) добавить сразу после `PriceToFreeCashFlowTtm: 20.20,`:

```go
		MarketCapitalization:             21.21,
```

И добавить проверку рядом с остальными (после блока `if got[0].PriceToFreeCashFlowTtm != want.PriceToFreeCashFlowTtm {...}`):

```go
	if got[0].MarketCapitalization != want.MarketCapitalization {
		t.Errorf("MarketCapitalization = %v, want %v", got[0].MarketCapitalization, want.MarketCapitalization)
	}
```

- [ ] **Step 2: Запустить — не компилируется (RED)**

Run: `go test ./internal/converter/ -run TestConvertFundamentalsFromPb`
Expected: FAIL — `model.Fundamentals` не имеет поля `MarketCapitalization` (ошибка компиляции `unknown field`).

- [ ] **Step 3: Добавить поле в модель**

В `internal/model/fundamentals.go`, в конце struct `Fundamentals`, после строки `PriceToFreeCashFlowTtm float64`, добавить:

```go

	MarketCapitalization float64 // рыночная капитализация, валюта инструмента
```

- [ ] **Step 4: Замапить в конвертере**

В `internal/converter/fundamentals.go`, в литерале `&model.Fundamentals{...}`, после строки `PriceToFreeCashFlowTtm:           f.PriceToFreeCashFlowTtm,` добавить:

```go
			MarketCapitalization:             f.MarketCapitalization,
```

- [ ] **Step 5: Запустить — зелёный (GREEN)**

Run: `go test ./internal/converter/ -run TestConvertFundamentalsFromPb -v`
Expected: PASS.

- [ ] **Step 6: Собрать пакеты**

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: успех.

- [ ] **Step 7: Коммит**

```bash
git add internal/model/fundamentals.go internal/converter/fundamentals.go internal/converter/fundamentals_test.go
git commit -m "feat(screener): map MarketCapitalization from GetAssetFundamentals"
```

---

## Task 2: Фильтр ликвидности в чистом ядре

**Files:**
- Modify: `internal/service/screener/dividend/rank/config.go`
- Modify: `internal/service/screener/dividend/rank/rank.go`
- Test: `internal/service/screener/dividend/rank/rank_test.go`
- Test (ретрофит, чтобы весь пакет остался зелёным): `internal/service/screener/dividend/service_test.go`

**Interfaces:**
- Consumes: `model.Fundamentals.MarketCapitalization` (Task 1).
- Produces: `rank.Config.MinMarketCap float64`; константа `reasonIlliquid = "низкая ликвидность"`; `gate()` отсеивает дивидендного плательщика с `MarketCapitalization < cfg.MinMarketCap` (в т.ч. 0). Семантика: пул выживших сжимается до ликвидных → перцентильные пилляры считаются среди ликвидных пиров.

**Обоснование:** живой прогон показал, что относительный ранг по всей вселенной (191 имя, включая неликвид small-cap) топит голубые фишки Golden X ниже медианы. Фильтр по капитализации оставляет только торгуемые имена; поскольку `Rank` строит перцентильные пулы по выжившим, композиты голубых фишек становятся сопоставимыми.

- [ ] **Step 1: Написать падающий тест + ретрофит существующих фондов**

В `internal/service/screener/dividend/rank/rank_test.go`:

(1) В начало файла, сразу после строки `package rank`, добавить (перед первой функцией) объявление helper-константы:

```go
// aboveCapFloor — капитализация заведомо выше DefaultConfig().MinMarketCap,
// чтобы фильтр ликвидности не мешал тестам, проверяющим другое поведение.
// Читает актуальный сид, поэтому устойчива к калибровке порога.
var aboveCapFloor = DefaultConfig().MinMarketCap + 1
```

(2) Добавить `MarketCapitalization: aboveCapFloor` **во все фонды, у которых `yield > 0` и которые должны либо выжить, либо быть отсеяны НЕ по ликвидности** (все, кроме `nodiv`). Конкретно — в каждый из этих литералов дописать поле:

- фонд `lev` (`TestRank_GateHighLeverage`)
- фонд `trap` (`TestRank_YieldTrap`)
- фонд `nodata` (`TestRank_MissingFundamentalsNoLongerGated`)
- фонд `bank` (`TestRank_KeepsBankLikeDividendPayer`)
- фонд `nopayout` (`TestRank_NeutralSustainabilityWhenPayoutMissing`)
- фонды `strong` и `weak` (`TestRank_OrdersSurvivorsByComposite`)
- фонды `a` и `b` (`TestRank_DegenerateGrowthPoolIsNeutral`)

Пример (фонд `lev`) — было:

```go
		{AssetUID: "lev", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 50, NetDebtToEbitda: 5, EbitdaTtm: 100, Roic: 0.1},
```

стало:

```go
		{AssetUID: "lev", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 50, NetDebtToEbitda: 5, EbitdaTtm: 100, Roic: 0.1, MarketCapitalization: aboveCapFloor},
```

Аналогично дописать `, MarketCapitalization: aboveCapFloor` перед закрывающей `}` в остальных перечисленных фондах. Фонд `nodiv` НЕ трогать (он отсеивается по «нет дивиденда» раньше проверки ликвидности).

(3) Добавить новый тест (в конец файла):

```go
func TestRank_GateIlliquid(t *testing.T) {
	cfg := DefaultConfig()
	u := []*model.Fundamentals{
		// Платит дивиденд, но капа ниже порога → отсев по ликвидности.
		{AssetUID: "small", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: cfg.MinMarketCap - 1},
		// Платит дивиденд, но капа не пришла (0) → отсев по ликвидности.
		{AssetUID: "nocap", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: 0},
		// Платит дивиденд, капа выше порога → проходит.
		{AssetUID: "big", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: cfg.MinMarketCap + 1},
	}
	got := byUID(Rank(u, cfg))
	if got["small"].GateReason != reasonIlliquid {
		t.Fatalf("small: GateReason = %q, want %q", got["small"].GateReason, reasonIlliquid)
	}
	if got["nocap"].GateReason != reasonIlliquid {
		t.Fatalf("nocap: GateReason = %q, want %q", got["nocap"].GateReason, reasonIlliquid)
	}
	if got["big"].GateReason != "" {
		t.Fatalf("big must survive, gated: %q", got["big"].GateReason)
	}
}
```

- [ ] **Step 2: Запустить — не компилируется (RED)**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestRank_GateIlliquid`
Expected: FAIL — `reasonIlliquid` не объявлена и `Config` не имеет `MinMarketCap` (ошибки компиляции).

- [ ] **Step 3: Добавить порог в Config**

В `internal/service/screener/dividend/rank/config.go`:

(1) В struct `Config`, после строки `YieldCapPct        float64 // потолок для yield-подсчёта`, добавить:

```go
	MinMarketCap       float64 // ниже (в т.ч. 0 = нет данных) — отсев как неликвид
```

(2) В `DefaultConfig()`, после строки `YieldCapPct:        14.0,`, добавить сид (единицы подтверждаются в Task 4):

```go
		MinMarketCap:       50_000_000_000, // ₽50 млрд, сид — калибруется на живых данных
```

- [ ] **Step 4: Добавить константу и ветку в gate**

В `internal/service/screener/dividend/rank/rank.go`:

(1) В блок констант добавить `reasonIlliquid` (после `reasonNoDividend`):

```go
const (
	reasonNoDividend    = "нет дивиденда"
	reasonIlliquid      = "низкая ликвидность"
	reasonHighLeverage  = "долг > порога"
	reasonUnsustainable = "payout > порога"
	reasonYieldTrap     = "yield trap"
)
```

(2) В `gate()`, сразу после блока `if y <= 0 { return reasonNoDividend, false }`, добавить проверку ликвидности (до yield-trap):

```go
	if f.MarketCapitalization < cfg.MinMarketCap {
		return reasonIlliquid, false
	}
```

Обновить doc-comment над `gate` (заменить абзац с «Единственное жёсткое основание…»):

```go
// gate возвращает (reason, isTrap). Пустой reason => компания проходит.
// Жёсткие основания отсева по данным: нет дивиденда (yield <= 0) и низкая
// ликвидность (MarketCapitalization < MinMarketCap, в т.ч. 0 из-за proto3
// omitempty). Отсутствие EBITDA/payout (0) НЕ исключает компанию — оно
// нейтрально учитывается в пиллярах.
```

- [ ] **Step 5: Ретрофит фикстур сервиса (сохранить пакет зелёным)**

После добавления фильтра фонды в `service_test.go` с капой 0 отсеются как неликвид и сломают тесты сервиса. В `internal/service/screener/dividend/service_test.go`:

(1) В начало файла, после блока `import (...)`, добавить:

```go
// aboveCapFloor — капа выше сида MinMarketCap, чтобы фикстуры не резались
// фильтром ликвидности (тестируем ранжирование/бонус, а не ликвидность).
var aboveCapFloor = rank.DefaultConfig().MinMarketCap + 1
```

(2) В функции `fundUniverse()` дописать `, MarketCapitalization: aboveCapFloor` перед закрывающей `}` в каждый из 4 фондов (`a-strong`, `a-mid`, `a-weak`, `a-gated`).

(3) В `TestRankBonus_SharedAssetCoversAllInstruments` дописать `, MarketCapitalization: aboveCapFloor` в оба фонда (`a-shared`, `a-weak`).

- [ ] **Step 6: Запустить rank + service (GREEN, -race)**

Run: `go test ./internal/service/screener/dividend/... -race`
Expected: PASS. (`TestRank_GateIlliquid` зелёный; существующие rank-тесты зелёные — их выжившие/иначе-гейтнутые фонды теперь ликвидны; тесты сервиса зелёные — фикстуры ликвидны, `a-gated` по-прежнему гейтится по долгу/payout.)

- [ ] **Step 7: Собрать пакеты**

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: успех.

- [ ] **Step 8: Коммит**

```bash
git add internal/service/screener/dividend/rank/config.go internal/service/screener/dividend/rank/rank.go internal/service/screener/dividend/rank/rank_test.go internal/service/screener/dividend/service_test.go
git commit -m "feat(screener): gate illiquid dividend payers by market-cap floor"
```

---

## Task 3: Абсолютные полосы бонуса

**Files:**
- Modify: `internal/service/screener/dividend/rank/config.go`
- Modify: `internal/service/screener/dividend/service.go`
- Test: `internal/service/screener/dividend/service_test.go`

**Interfaces:**
- Consumes: `rank.Config`, `ScoredCompany.Composite`.
- Produces: `bonusFromScore(composite float64, cfg rank.Config) int` (полосы `≥T3→3, ≥T2→2, ≥T1→1, иначе 0`); `rank.Config.BonusScoreT1/T2/T3 float64`. `RankBonus(instrumentID)` возвращает `bonusFromScore` по композиту инструмента; `bonusFromRank` удаляется.

- [ ] **Step 1: Написать тесты (RED)**

В `internal/service/screener/dividend/service_test.go`:

(1) Добавить прямой юнит-тест полос (пороги локальные — не зависят от калибровки `DefaultConfig`):

```go
func TestBonusFromScore(t *testing.T) {
	cfg := rank.Config{BonusScoreT1: 50, BonusScoreT2: 70, BonusScoreT3: 90}
	cases := []struct {
		composite float64
		want      int
	}{
		{95, 3}, {90, 3}, // >= T3
		{89, 2}, {70, 2}, // >= T2
		{69, 1}, {50, 1}, // >= T1
		{49, 0}, {0, 0},  // < T1
	}
	for _, c := range cases {
		if got := bonusFromScore(c.composite, cfg); got != c.want {
			t.Errorf("bonusFromScore(%v) = %d, want %d", c.composite, got, c.want)
		}
	}
}
```

(2) Заменить целиком тело `TestRankBonus_TopGetsMorePoints` на рефрейм под абсолютные полосы (монотонность + гейт/неизвестный, без хрупкой привязки к диапазону `[1,3]`, т.к. полосы калибруются позже):

```go
func TestRankBonus_TopGetsMorePoints(t *testing.T) {
	svc := newMockedService(t)

	strong := svc.RankBonus("i-strong")
	weak := svc.RankBonus("i-weak")
	gated := svc.RankBonus("i-gated")
	unknown := svc.RankBonus("i-does-not-exist")

	// bonusFromScore монотонна по композиту, а композит strong >= weak,
	// поэтому бонус strong не меньше weak при любых порогах.
	if strong < weak {
		t.Fatalf("strong bonus %d should be >= weak %d", strong, weak)
	}
	if gated != 0 {
		t.Fatalf("gated bonus = %d, want 0", gated)
	}
	if unknown != 0 {
		t.Fatalf("unknown bonus = %d, want 0", unknown)
	}
}
```

- [ ] **Step 2: Запустить — не компилируется (RED)**

Run: `go test ./internal/service/screener/dividend/ -run 'TestBonusFromScore|TestRankBonus_TopGetsMorePoints'`
Expected: FAIL — `bonusFromScore` не объявлена и `rank.Config` не имеет `BonusScoreT1/T2/T3` (ошибки компиляции).

- [ ] **Step 3: Добавить пороги бонуса в Config**

В `internal/service/screener/dividend/rank/config.go`:

(1) В struct `Config`, после `MinMarketCap float64 ...`, добавить:

```go

	BonusScoreT1 float64 // композит >= T1 → бонус +1
	BonusScoreT2 float64 // композит >= T2 → бонус +2
	BonusScoreT3 float64 // композит >= T3 → бонус +3
```

(2) В `DefaultConfig()`, после `MinMarketCap: ...`, добавить сиды (калибруются в Task 4):

```go

		BonusScoreT1: 55, // сиды — подбираются по распределению композитов ликвидной вселенной
		BonusScoreT2: 65,
		BonusScoreT3: 75,
```

- [ ] **Step 4: Заменить bonusFromRank на bonusFromScore**

В `internal/service/screener/dividend/service.go`:

(1) Удалить функцию `bonusFromRank` целиком (вместе с её doc-comment) и заменить на:

```go
// bonusFromScore отображает композит (0..100) в фундаментальный бонус Golden X.
// Пороги — точки калибровки (см. live-шаг); полосы абсолютны, чтобы бонус не
// зависел от состава вселенной.
func bonusFromScore(composite float64, cfg rank.Config) int {
	switch {
	case composite >= cfg.BonusScoreT3:
		return 3
	case composite >= cfg.BonusScoreT2:
		return 2
	case composite >= cfg.BonusScoreT1:
		return 1
	default:
		return 0
	}
}
```

(2) В функции `refresh`, в цикле по `survivors`, заменить строку:

```go
		bonus := bonusFromRank(i, total)
```

на:

```go
		bonus := bonusFromScore(sc.Composite, s.cfg)
```

(3) Удалить ставшую неиспользуемой переменную `total := len(survivors)` (строка перед циклом). Индекс `i` в `for i, sc := range survivors` больше не нужен для бонуса, но может остаться, если используется дальше; если после правки `i` не используется — заменить на `for _, sc := range survivors`.

- [ ] **Step 5: Запустить тесты (GREEN, -race)**

Run: `go test ./internal/service/screener/dividend/... -race -v -run 'TestBonusFromScore|TestRankBonus|TestTop'`
Expected: PASS (все). Затем весь пакет:

Run: `go test ./internal/service/screener/dividend/... -race`
Expected: PASS.

- [ ] **Step 6: Собрать пакеты**

Run: `go build ./internal/... ./pkg/... ./cmd/...`
Expected: успех.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/screener/dividend/rank/config.go internal/service/screener/dividend/service.go internal/service/screener/dividend/service_test.go
git commit -m "feat(screener): fundamental bonus by absolute composite bands"
```

---

## Task 4: Live-калибровка + финальный гейт

**Files:**
- Create (временный, не коммитить): `cmd/divcheck/main.go`
- Modify (калибровка): `internal/service/screener/dividend/rank/config.go`

**Цель:** прогнать исправленное ядро по Invest API и: (1) подтвердить единицы `MarketCapitalization`; (2) подобрать `MinMarketCap` так, чтобы все 11 curated-имён Golden X проходили, а micro-cap (Мордовская энергосбытовая, ТНС энерго и т.п.) отсеивались как «низкая ликвидность»; (3) подобрать `BonusScoreT1/T2/T3` по распределению композитов ликвидной вселенной. Диагностик бьёт по gRPC (токен `T_BANK` из `env/token.env`), Telegram не трогает.

**Curated-11 тикеры Golden X (все должны выживать):** SNGSP, TATNP, ROSN, LKOH, SBERP, CHMF, NLMK, MAGN, PHOR, TRNFP, BSPB.

- [ ] **Step 1: Создать временный диагностик**

Создать `cmd/divcheck/main.go`:

```go
// Command divcheck is a one-shot diagnostic for the dividend screener
// Golden X alignment: it hits the live Invest API directly (no Telegram) to
// calibrate the market-cap floor and the composite bonus bands. Temporary —
// delete after use.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/joho/godotenv"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
	grpcclient "tinvest/pkg/client/grpc"
)

// curated — 11 дивидендных тикеров Golden X, все обязаны выживать.
var curated = []string{"SNGSP", "TATNP", "ROSN", "LKOH", "SBERP", "CHMF", "NLMK", "MAGN", "PHOR", "TRNFP", "BSPB"}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "divcheck failed:", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load("./env/local.env")
	if err := godotenv.Load("./env/token.env"); err != nil {
		return fmt.Errorf("load token.env: %w", err)
	}
	token := os.Getenv("T_BANK")
	if token == "" {
		return fmt.Errorf("T_BANK not set")
	}

	client, err := grpcclient.NewClientGrpc("invest-public-api.tinkoff.ru:443", token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	ic := client.InstrumentsServiceClient()
	ctx := context.Background()

	shares, err := ic.Shares(ctx)
	if err != nil {
		return fmt.Errorf("Shares: %w", err)
	}

	uids := make([]string, 0, len(shares))
	shareByAsset := make(map[string]*model.Share, len(shares))
	for _, sh := range shares {
		if !sh.DivYieldFlag || sh.AssetUID == "" {
			continue
		}
		uids = append(uids, sh.AssetUID)
		shareByAsset[sh.AssetUID] = sh
	}

	funds, err := ic.GetAssetFundamentals(ctx, uids)
	if err != nil {
		return fmt.Errorf("GetAssetFundamentals: %w", err)
	}
	fundByAsset := make(map[string]*model.Fundamentals, len(funds))
	for _, f := range funds {
		fundByAsset[f.AssetUID] = f
	}

	cfg := rank.DefaultConfig()
	scored := rank.Rank(funds, cfg)
	scoredByAsset := make(map[string]rank.ScoredCompany, len(scored))
	byReason := map[string]int{}
	survivors := make([]rank.ScoredCompany, 0, len(scored))
	for _, sc := range scored {
		scoredByAsset[sc.AssetUID] = sc
		if sc.GateReason != "" {
			byReason[sc.GateReason]++
			continue
		}
		survivors = append(survivors, sc)
	}

	// 1. Единицы капитализации: сырые значения по curated-именам.
	fmt.Printf("MinMarketCap seed = %.0f\n", cfg.MinMarketCap)
	fmt.Println("\n=== UNIT CHECK: raw MarketCapitalization for curated Golden X names ===")
	for _, sh := range shareByAsset {
		if !contains(curated, sh.Ticker) {
			continue
		}
		f := fundByAsset[sh.AssetUID]
		mc := 0.0
		if f != nil {
			mc = f.MarketCapitalization
		}
		sc := scoredByAsset[sh.AssetUID]
		outcome := "RANKED"
		if sc.GateReason != "" {
			outcome = "GATED: " + sc.GateReason
		}
		fmt.Printf("  %-7s mktcap=%.3e comp=%.1f  %s\n", sh.Ticker, mc, sc.Composite, outcome)
	}

	// 2. Разбивка отсева.
	fmt.Printf("\n=== gate stats ===\nuniverse=%d ranked=%d gated=%d\n", len(funds), len(survivors), len(funds)-len(survivors))
	reasons := make([]string, 0, len(byReason))
	for r := range byReason {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	for _, r := range reasons {
		fmt.Printf("  gated: %-24s %d\n", r, byReason[r])
	}

	// 3. Распределение композитов ликвидной вселенной (для полос бонуса).
	comps := make([]float64, len(survivors))
	for i, sc := range survivors {
		comps[i] = sc.Composite
	}
	sort.Float64s(comps)
	if len(comps) > 0 {
		fmt.Println("\n=== liquid composite distribution (for T1/T2/T3) ===")
		for _, p := range []float64{0.10, 0.25, 0.50, 0.75, 0.90} {
			idx := int(p * float64(len(comps)-1))
			fmt.Printf("  p%02.0f = %.1f\n", p*100, comps[idx])
		}
		fmt.Printf("  min=%.1f max=%.1f\n", comps[0], comps[len(comps)-1])
	}

	// 4. TOP-15 ликвидной вселенной.
	sort.SliceStable(survivors, func(i, j int) bool { return survivors[i].Composite > survivors[j].Composite })
	fmt.Println("\n=== TOP 15 (liquid) ===")
	n := 15
	if len(survivors) < n {
		n = len(survivors)
	}
	for i := 0; i < n; i++ {
		sc := survivors[i]
		sh := shareByAsset[sc.AssetUID]
		ticker := "?"
		if sh != nil {
			ticker = sh.Ticker
		}
		fmt.Printf("%2d. %-7s comp=%5.1f bonus=%d\n", i+1, ticker, sc.Composite, bonus(sc.Composite, cfg))
	}
	return nil
}

func bonus(composite float64, cfg rank.Config) int {
	switch {
	case composite >= cfg.BonusScoreT3:
		return 3
	case composite >= cfg.BonusScoreT2:
		return 2
	case composite >= cfg.BonusScoreT1:
		return 1
	default:
		return 0
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Собрать диагностик**

Run: `go build ./cmd/divcheck`
Expected: успех. Если API-сигнатуры (`grpcclient.NewClientGrpc`, `InstrumentsServiceClient`, `Shares`, `GetAssetFundamentals`) отличаются — адаптировать вызовы под реальный код (цель — рабочий диагностик, не дословная копия). Удалить бинарь `divcheck` из корня в Step 6.

- [ ] **Step 3: Прогнать по живому API**

Run: `go run ./cmd/divcheck`
Expected: печатается unit-check по curated-именам, разбивка отсева, распределение композитов и TOP-15.

- [ ] **Step 4: Подтвердить единицы и подобрать пороги**

Проанализировать вывод и внести правки в `internal/service/screener/dividend/rank/config.go` (`DefaultConfig`), пересобирая и повторяя Step 3, пока не выполнены критерии:

1. **Единицы `MarketCapitalization` подтверждены.** По сырым `mktcap` curated-имён понять шкалу (₽ vs млн ₽). Если шкала не ₽ (напр. значения ~10^2–10^4 вместо 10^11–10^13) — подставить `MinMarketCap` на верной шкале. Зафиксировать вывод в отчёте.
2. **Все 11 curated-имён → `RANKED`.** Ни одно не должно отсеиваться по «низкая ликвидность». Подобрать `MinMarketCap` так, чтобы наименьшая капа среди curated проходила, а micro-cap (Мордовская энергосбытовая ≈ единицы млрд ₽ и т.п.) — нет. Проверить: в разбивке появилась причина «низкая ликвидность» с ненулевым счётчиком.
3. **`BonusScoreT1/T2/T3` осмысленно делят распределение.** По напечатанным перцентилям композитов ликвидной вселенной задать пороги так, чтобы бонус различал сильные/средние/слабые имена (ориентир: T1≈p25, T2≈p50, T3≈p75 ликвидного распределения; итоговые значения — на усмотрение по данным). Проверить в TOP-15, что curated-фишки получают ненулевой, различающийся бонус.

Логику (`rank.go`, `service.go`) на этом шаге НЕ менять — только константы в `config.go`.

- [ ] **Step 5: Зафиксировать калибровку**

Если `config.go` менялся (ожидаемо — как минимум `MinMarketCap` и/или пороги):

```bash
git add internal/service/screener/dividend/rank/config.go
git commit -m "chore(screener): calibrate market-cap floor and bonus bands on live data"
```

- [ ] **Step 6: Удалить временный диагностик**

```bash
rm -rf cmd/divcheck
rm -f divcheck
git status --short
```

Expected: `cmd/divcheck` и бинарь `divcheck` отсутствуют; рабочее дерево без временных файлов.

- [ ] **Step 7: Финальный гейт**

Run: `./bin/mage ci`
Expected: PASS (lint + `go test -race ./...` + mock-drift без дрейфа — мок-интерфейсы не менялись).

---

## Known limitations (сознательно НЕ адресуются)

- **Кривой снимок API** (напр. LKOH: ROE −20%, payout 0): имя остаётся в ликвидной вселенной (большая капа), но получает низкий композит → низкий/нулевой бонус. Это корректно — бонус отражает текущий фундамент; скринер не «чинит» данные API. Вне scope.
- **Капитализация ≠ оборот.** Крупная, но неактивно торгуемая фишка пройдёт порог. На голубых фишках Мосбиржи не наблюдается; ADV-фильтр отложен до появления доказательств необходимости.
- **Missing net debt → оптимистичный Safety** и **неиспользуемый `indicators.Percentile` (R-7)** — перенесено из ограничений предыдущего плана, вне scope.

---

## Self-Review

- **Покрытие спеки:** поле капы (Task 1) ↔ спека §1; фильтр ликвидности в чистой `gate()` (Task 2) ↔ §2; абсолютные полосы бонуса (Task 3) ↔ §3; live-калибровка единиц/порогов (Task 4) ↔ «Live-validation step». Telegram (§4) кода не требует — новая причина «низкая ликвидность» проходит через generic `Stats.ByReason` без правок `render.go`/`service.Top`.
- **Плейсхолдеры:** нет — весь код и команды приведены дословно; сиды порогов (`MinMarketCap` 50e9, T 55/65/75) — не плейсхолдеры, а стартовые значения с явным критерием калибровки в Task 4 Step 4.
- **Согласованность типов:** `MarketCapitalization float64` (модель↔конвертер↔тесты↔gate↔диагностик); `MinMarketCap`, `BonusScoreT1/T2/T3 float64` в `rank.Config`; `bonusFromScore(float64, rank.Config) int`; `reasonIlliquid = "низкая ликвидность"`; `gate(*model.Fundamentals, Config) (string, bool)` — сигнатуры совпадают между задачами и с существующим кодом. `AssetUID` везде.
- **Ломающиеся тесты учтены:** добавление поля ломает `TestConvertFundamentalsFromPb` (обновлён в Task 1); фильтр ликвидности ломает существующие rank- и service-тесты с капой 0 (ретрофит `aboveCapFloor` в Task 2 Steps 1 и 5); смена бонуса ломает `TestRankBonus_TopGetsMorePoints` (рефрейм в Task 3 Step 1). Каждый коммит оставляет весь пакет зелёным.
- **Порядок ворот:** проверка ликвидности стоит сразу после «нет дивиденда» (спека, data-flow) — поэтому фонды с `yield>0`, тестирующие другие причины/выживание, требуют капы выше порога (учтено в Task 2 Step 1).
