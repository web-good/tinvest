# Dividend Screener Gate Calibration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Починить чрезмерно строгий gate дивидендного скринера, из-за которого 80% вселенной (включая SBER, LKOH, SNGSP, MTSS, PHOR) отсеивается, и подтвердить исправление на живых данных.

**Architecture:** Правки только в чистом ядре ранжирования `internal/service/screener/dividend/rank/` (без I/O). Единственным жёстким основанием для отсева по данным остаётся «не платит дивиденд» (`yield <= 0`); отсутствие EBITDA/payout больше не исключает компанию, а нейтрально оценивается в пиллярах. Перцентильные пилляры делаются устойчивыми к «мёртвым» полям (весь пул — одно значение). Финал — повторный прогон по Invest API одноразовым диагностиком (без Telegram).

**Tech Stack:** Go 1.25, чистые функции + table-driven тесты, `godotenv` (уже в зависимостях), gRPC Tinkoff `GetAssetFundamentals`, `./bin/mage ci`.

## Global Constraints

- Go 1.25; идиоматичный Go, MixedCaps, ошибки — обёрнутый `%w`.
- Поле в модели и в `ScoredCompany` называется **`AssetUID`** (не `AssetUid`) — revive var-naming; не переименовывать обратно.
- `detector.Detect` и `classifier.Classify` остаются чистыми — эти правки их не касаются, всё в пакете `rank`.
- Единицы полей `GetAssetFundamentals` **ПОДТВЕРЖДЕНЫ живой валидацией 2026-07-21: yield и payout — проценты** (SBER payout 88, MTSS payout 338, ttmYield 15–41). Пороги в `rank.DefaultConfig()` на верной шкале — деления на 100 НЕ требуется.
- Поля fundamentals — `proto3 float64` с `omitempty`: «значение отсутствует» и «0.0» неразличимы. Отсюда правило: `0` в fundamental-поле трактовать как «нет данных», а не как «реальный ноль».
- `go build ./internal/... ./pkg/... ./cmd/...` (НЕ `./...` — падает на `magefiles`).
- Гейт: `./bin/mage ci` = lint + `go test -race ./...` + mock-drift. Мок-интерфейсы здесь не меняются — `./bin/mage mocks` не нужен.
- Диагностик из Task 3 (`cmd/divcheck`) — **временный, в git НЕ коммитить**, удалить в конце.

---

## File Structure

**Модифицируются:**
- `internal/service/screener/dividend/rank/rank.go` — убрать missing-data gate + const; нейтральный payout в Sustainability; устойчивые к вырожденному пулу перцентильные пилляры.
- `internal/service/screener/dividend/rank/rank_test.go` — инвертировать устаревший `TestRank_MissingData`; добавить тесты на выживание «банковских» имён, нейтральный payout, нейтральный вырожденный пул.
- `internal/service/screener/dividend/rank/config.go` — **только если** Task 3 выявит перекос порогов (по умолчанию не трогаем).

**Создаётся временно (не коммитить, удаляется в Task 3):**
- `cmd/divcheck/main.go` — одноразовый диагностик живого прогона.

---

## Task 1: Ослабить gate и нейтрально оценивать отсутствующий payout

**Files:**
- Modify: `internal/service/screener/dividend/rank/rank.go`
- Test: `internal/service/screener/dividend/rank/rank_test.go`

**Interfaces:**
- Consumes: `model.Fundamentals`, `rank.Config`, `rank.DefaultConfig()`.
- Produces: `Rank(universe []*model.Fundamentals, cfg Config) []ScoredCompany` — семантика меняется: компания с `yield > 0`, но `EbitdaTtm == 0` и/или `DividendPayoutRatioFy == 0`, БОЛЬШЕ НЕ отсеивается (`GateReason == ""`); её `Sustainability` считается с нейтральным (0.5) payout-компонентом. Константа `reasonMissingData` удаляется.

- [ ] **Step 1: Написать падающие тесты**

В `internal/service/screener/dividend/rank/rank_test.go` добавить два новых теста и заменить устаревший `TestRank_MissingData`.

Заменить целиком функцию `TestRank_MissingData` (строки с `func TestRank_MissingData`) на:

```go
func TestRank_MissingFundamentalsNoLongerGated(t *testing.T) {
	// Платит дивиденд (yield 8), но EBITDA и payout не пришли от API (0).
	// Раньше отсеивалось как "нет ключевых данных" — теперь должно выживать.
	u := []*model.Fundamentals{
		{AssetUID: "nodata", ForwardAnnualDividendYield: 8, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nodata"].GateReason != "" {
		t.Fatalf("dividend payer must survive, gated: %q", got["nodata"].GateReason)
	}
}

func TestRank_KeepsBankLikeDividendPayer(t *testing.T) {
	// SBER-подобный: банк платит дивиденд (yield через TTM 27),
	// но EBITDA у банка отсутствует (0) и payout пришёл 0.
	u := []*model.Fundamentals{
		{AssetUID: "bank", DividendYieldDailyTtm: 27, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0, Roe: 0.22, FreeCashFlowTtm: 100, EvToEbitdaMrq: 0},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["bank"].GateReason != "" {
		t.Fatalf("bank dividend payer must survive, gated: %q", got["bank"].GateReason)
	}
}

func TestRank_NeutralSustainabilityWhenPayoutMissing(t *testing.T) {
	// payout отсутствует (0) → нейтральный payoutFit 0.5, а не 0;
	// FCF > 0 добавляет 0.3. Sustainability = 0.7*0.5 + 0.3 = 0.65.
	u := []*model.Fundamentals{
		{AssetUID: "nopayout", DividendYieldDailyTtm: 12, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0.5, EbitdaTtm: 0, Roe: 0.2, FreeCashFlowTtm: 100, EvToEbitdaMrq: 5},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nopayout"].GateReason != "" {
		t.Fatalf("should survive, gated: %q", got["nopayout"].GateReason)
	}
	if diff := got["nopayout"].Sustainability - 0.65; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Sustainability = %v, want 0.65 (neutral payout)", got["nopayout"].Sustainability)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает / не компилируется**

Run: `go test ./internal/service/screener/dividend/rank/ -run 'TestRank_MissingFundamentalsNoLongerGated|TestRank_KeepsBankLikeDividendPayer|TestRank_NeutralSustainabilityWhenPayoutMissing'`
Expected: FAIL — `nodata`/`bank` отсеиваются как «нет ключевых данных»; `Sustainability` у `nopayout` = `0.7*0 + 0.3 = 0.3`, не 0.65.

- [ ] **Step 3: Убрать missing-data gate и его константу**

В `internal/service/screener/dividend/rank/rank.go`:

1. В блоке констант удалить строку `reasonMissingData = "нет ключевых данных"`:

```go
const (
	reasonNoDividend    = "нет дивиденда"
	reasonHighLeverage  = "долг > порога"
	reasonUnsustainable = "payout > порога"
	reasonYieldTrap     = "yield trap"
)
```

2. В функции `gate` удалить блок missing-data (эти три строки):

```go
	if f.EbitdaTtm <= 0 || f.DividendPayoutRatioFy <= 0 {
		return reasonMissingData, false
	}
```

Итоговая `gate`:

```go
// gate возвращает (reason, isTrap). Пустой reason => компания проходит.
// Единственное жёсткое основание отсева по данным — отсутствие дивиденда
// (yield <= 0). Отсутствие EBITDA/payout (0 из-за proto3 omitempty) НЕ
// исключает компанию — оно нейтрально учитывается в пиллярах.
func gate(f *model.Fundamentals, cfg Config) (string, bool) {
	y := yieldOf(f)
	if y <= 0 {
		return reasonNoDividend, false
	}
	trap := y >= cfg.YieldTrapMinYield &&
		(f.DividendPayoutRatioFy > 100 || f.NetDebtToEbitda > 3 || f.FreeCashFlowTtm < 0)
	if trap {
		return reasonYieldTrap, true
	}
	if f.NetDebtToEbitda > cfg.MaxNetDebtToEbitda {
		return reasonHighLeverage, false
	}
	if f.DividendPayoutRatioFy > cfg.MaxPayoutPct {
		return reasonUnsustainable, false
	}
	return "", false
}
```

- [ ] **Step 4: Нейтральный payout в Sustainability**

В `rank.go`, в функции `Rank`, заменить строку расчёта `sc.Sustainability`:

```go
		sc.Sustainability = 0.7*payoutFit(f.DividendPayoutRatioFy, cfg) + 0.3*boolScore(f.FreeCashFlowTtm > 0)
```

на:

```go
		sc.Sustainability = 0.7*sustainabilityPayout(f.DividendPayoutRatioFy, cfg) + 0.3*boolScore(f.FreeCashFlowTtm > 0)
```

И добавить рядом с `payoutFit` новую функцию:

```go
// sustainabilityPayout: как payoutFit, но при отсутствии данных о payout
// (0 из-за proto3 omitempty) возвращает нейтральные 0.5 — «неизвестно»
// не должно ни вознаграждать, ни штрафовать компанию, которая платит дивиденд.
func sustainabilityPayout(payout float64, cfg Config) float64 {
	if payout <= 0 {
		return 0.5
	}
	return payoutFit(payout, cfg)
}
```

- [ ] **Step 5: Запустить новые тесты — зелёные**

Run: `go test ./internal/service/screener/dividend/rank/ -run 'TestRank_MissingFundamentalsNoLongerGated|TestRank_KeepsBankLikeDividendPayer|TestRank_NeutralSustainabilityWhenPayoutMissing' -v`
Expected: PASS (все три).

- [ ] **Step 6: Запустить весь пакет — существующие тесты не сломаны**

Run: `go test ./internal/service/screener/dividend/... -v`
Expected: PASS. (`TestRank_YieldTrap`, `TestRank_GateHighLeverage`, `TestRank_GateNoDividend`, `TestRank_OrdersSurvivorsByComposite` и тесты `service`/`notification` — все зелёные; их входные данные имеют `payout > 0`, поэтому нейтральная ветка не задействуется.)

- [ ] **Step 7: Коммит**

```bash
git add internal/service/screener/dividend/rank/rank.go internal/service/screener/dividend/rank/rank_test.go
git commit -m "fix(screener): stop gating dividend payers on missing EBITDA/payout"
```

---

## Task 2: Устойчивые перцентильные пилляры для вырожденного пула

**Files:**
- Modify: `internal/service/screener/dividend/rank/rank.go`
- Test: `internal/service/screener/dividend/rank/rank_test.go`

**Interfaces:**
- Consumes: `indicators.PercentileRank`.
- Produces: приватные `percentileOrNeutral(pool []float64, x float64) float64` и `hasSpread(pool []float64) bool`; пилляры `DivGrowth`, `Quality`, `Valuation` считаются через `percentileOrNeutral`. Когда весь пул — одно значение (например, `FiveYearAnnualDividendGrowthRate == 0` по всей вселенной), пилляр = 0.5 для всех, а не 0.

**Обоснование:** живой прогон показал, что `FiveYearAnnualDividendGrowthRate` приходит нулём по всей вселенной. `PercentileRank` на пуле из одинаковых значений возвращает 0 всем (строгое `<`), из-за чего 15% веса composite превращаются в мёртвую константу 0 и занижают все оценки. Нейтрализация до 0.5 убирает искажение, не создавая ложной дифференциации.

- [ ] **Step 1: Написать падающий тест**

В `rank_test.go` добавить:

```go
func TestRank_DegenerateGrowthPoolIsNeutral(t *testing.T) {
	// Весь пул выживших имеет FiveYearAnnualDividendGrowthRate == 0 (поля нет у API).
	// DivGrowth должен быть нейтральным 0.5, а не 0.
	u := []*model.Fundamentals{
		{AssetUID: "a", DividendYieldDailyTtm: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, EbitdaTtm: 100, Roic: 0.2, EvToEbitdaMrq: 4, FreeCashFlowTtm: 100, FiveYearAnnualDividendGrowthRate: 0},
		{AssetUID: "b", DividendYieldDailyTtm: 9, DividendPayoutRatioFy: 50, NetDebtToEbitda: 1.0, EbitdaTtm: 100, Roic: 0.1, EvToEbitdaMrq: 6, FreeCashFlowTtm: 100, FiveYearAnnualDividendGrowthRate: 0},
	}
	got := byUID(Rank(u, DefaultConfig()))
	for _, id := range []string{"a", "b"} {
		if diff := got[id].DivGrowth - 0.5; diff > 1e-9 || diff < -1e-9 {
			t.Fatalf("%s DivGrowth = %v, want 0.5 (neutral degenerate pool)", id, got[id].DivGrowth)
		}
	}
}
```

- [ ] **Step 2: Запустить — падает**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestRank_DegenerateGrowthPoolIsNeutral`
Expected: FAIL — `DivGrowth = 0`, want 0.5.

- [ ] **Step 3: Добавить helpers и применить к пиллярам**

В `rank.go` добавить функции (рядом с `qualityMetric`):

```go
// percentileOrNeutral возвращает 0.5, когда в пуле нет разброса (все значения
// равны — например, fundamental-поле отсутствует по всей вселенной), чтобы
// мёртвый сигнал не занижал composite. Иначе — обычный PercentileRank.
func percentileOrNeutral(pool []float64, x float64) float64 {
	if !hasSpread(pool) {
		return 0.5
	}
	return indicators.PercentileRank(pool, x)
}

// hasSpread сообщает, есть ли в пуле хотя бы два различных значения.
func hasSpread(pool []float64) bool {
	if len(pool) < 2 {
		return false
	}
	first := pool[0]
	for _, v := range pool[1:] {
		if v != first {
			return true
		}
	}
	return false
}
```

В функции `Rank` заменить три строки расчёта перцентильных пилляров:

```go
		sc.DivGrowth = indicators.PercentileRank(divGrowth, f.FiveYearAnnualDividendGrowthRate)
		sc.Quality = indicators.PercentileRank(roic, qualityMetric(f))
		sc.Valuation = 1 - indicators.PercentileRank(evEbitda, f.EvToEbitdaMrq) // ниже EV/EBITDA — лучше
```

на:

```go
		sc.DivGrowth = percentileOrNeutral(divGrowth, f.FiveYearAnnualDividendGrowthRate)
		sc.Quality = percentileOrNeutral(roic, qualityMetric(f))
		sc.Valuation = 1 - percentileOrNeutral(evEbitda, f.EvToEbitdaMrq) // ниже EV/EBITDA — лучше
```

- [ ] **Step 4: Запустить тест — зелёный**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestRank_DegenerateGrowthPoolIsNeutral -v`
Expected: PASS.

- [ ] **Step 5: Весь пакет + race**

Run: `go test ./internal/service/screener/dividend/... -race`
Expected: PASS. (`TestRank_OrdersSurvivorsByComposite` не меняется: у его двух выживших пилляр-пулы имеют разброс, поэтому `percentileOrNeutral == PercentileRank`.)

- [ ] **Step 6: Коммит**

```bash
git add internal/service/screener/dividend/rank/rank.go internal/service/screener/dividend/rank/rank_test.go
git commit -m "fix(screener): neutralize percentile pillars on degenerate (all-equal) pools"
```

---

## Task 3: Повторная валидация на живых данных + подтверждение порогов

**Files:**
- Create (временный, не коммитить): `cmd/divcheck/main.go`
- Modify (только если выявлен перекос): `internal/service/screener/dividend/rank/config.go`

**Цель:** прогнать исправленное ядро по Invest API и подтвердить: (1) SBER, LKOH, SNGSP, MTSS, PHOR теперь `RANKED`; (2) доля отсева упала с 80% до вменяемой (остаток `нет дивиденда` — легитимные не платящие: GAZP и т.п.); (3) топ содержит узнаваемые дивидендные имена. Диагностик бьёт по gRPC-API (токен `T_BANK` из `env/token.env`), Telegram не трогается — прод-бот не конфликтует.

- [ ] **Step 1: Создать временный диагностик**

Создать `cmd/divcheck/main.go`:

```go
// Command divcheck is a one-shot diagnostic for the dividend screener gate
// calibration: it hits the live Invest API directly (no Telegram) to confirm
// blue-chip dividend payers survive the gate. Temporary — delete after use.
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

	dividendShares := make([]*model.Share, 0, len(shares))
	uids := make([]string, 0, len(shares))
	shareByAsset := make(map[string]*model.Share, len(shares))
	for _, sh := range shares {
		if !sh.DivYieldFlag || sh.AssetUID == "" {
			continue
		}
		dividendShares = append(dividendShares, sh)
		uids = append(uids, sh.AssetUID)
		shareByAsset[sh.AssetUID] = sh
	}
	fmt.Printf("shares total=%d  dividend-flagged with asset=%d\n", len(shares), len(dividendShares))

	funds, err := ic.GetAssetFundamentals(ctx, uids)
	if err != nil {
		return fmt.Errorf("GetAssetFundamentals: %w", err)
	}
	fundByAsset := make(map[string]*model.Fundamentals, len(funds))
	for _, f := range funds {
		fundByAsset[f.AssetUID] = f
	}

	scored := rank.Rank(funds, rank.DefaultConfig())
	scoredByAsset := make(map[string]rank.ScoredCompany, len(scored))
	gated := 0
	byReason := map[string]int{}
	survivors := make([]rank.ScoredCompany, 0, len(scored))
	for _, sc := range scored {
		scoredByAsset[sc.AssetUID] = sc
		if sc.GateReason != "" {
			gated++
			byReason[sc.GateReason]++
			continue
		}
		survivors = append(survivors, sc)
	}

	fmt.Printf("\n=== rank.Rank stats ===\nuniverse(with fundamentals)=%d  ranked=%d  gated=%d\n", len(funds), len(survivors), gated)
	reasons := make([]string, 0, len(byReason))
	for r := range byReason {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	for _, r := range reasons {
		fmt.Printf("  gated: %-24s %d\n", r, byReason[r])
	}

	probe := []string{"LKOH", "SBER", "GAZP", "MTSS", "PHOR", "TATN", "SNGSP", "MGNT", "ROSN", "CHMF"}
	fmt.Println("\n=== gate outcome for probe blue-chips (expect SBER/LKOH/SNGSP/MTSS/PHOR now RANKED) ===")
	for _, sh := range dividendShares {
		if !contains(probe, sh.Ticker) {
			continue
		}
		sc := scoredByAsset[sh.AssetUID]
		outcome := "RANKED"
		if sc.GateReason != "" {
			outcome = "GATED: " + sc.GateReason
		}
		fmt.Printf("  %-6s %s\n", sh.Ticker, outcome)
	}

	fmt.Println("\n=== TOP 15 ===")
	n := 15
	if len(survivors) < n {
		n = len(survivors)
	}
	for i := 0; i < n; i++ {
		sc := survivors[i]
		sh := shareByAsset[sc.AssetUID]
		ticker, name := "?", "?"
		if sh != nil {
			ticker, name = sh.Ticker, sh.Name
		}
		f := fundByAsset[sc.AssetUID]
		yield := 0.0
		if f != nil {
			yield = f.ForwardAnnualDividendYield
			if yield == 0 {
				yield = f.DividendYieldDailyTtm
			}
		}
		fmt.Printf("%2d. %-6s comp=%5.1f yield=%.2f sust=%.2f safe=%.2f grow=%.2f qual=%.2f val=%.2f %s\n",
			i+1, ticker, sc.Composite, yield, sc.Sustainability, sc.Safety, sc.DivGrowth, sc.Quality, sc.Valuation, name)
	}
	return nil
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
Expected: успех. (Удалить получившийся бинарь `divcheck` из корня позже — Step 6.)

- [ ] **Step 3: Прогнать по живому API**

Run: `go run ./cmd/divcheck`
Expected: печатается статистика отсева, gate-исход по probe-именам и TOP-15.

- [ ] **Step 4: Проверить критерии приёмки**

Сверить вывод:
- **SBER, LKOH, SNGSP, MTSS, PHOR → `RANKED`** (до фикса были `GATED`). Это главный критерий.
- `gated` заметно меньше прежних 151/189; в разбивке доминирует `нет дивиденда` (легитимные не платящие: GAZP и т.п.), а `нет ключевых данных` исчезла как причина.
- TOP-15 содержит узнаваемые дивидендные имена (SBER/LKOH/TATN/SNGSP и т.п.), а не только small-cap Россети.

Если критерии выполнены — переходить к Step 5. Если что-то выглядит перекошенным (например, теперь всплыл yield-trap на легитимном имени или порог `MaxPayoutPct`/`YieldTrapMinYield` режет очевидно хорошую компанию), внести точечную правку констант в `internal/service/screener/dividend/rank/config.go` (`DefaultConfig`), пересобрать и повторить Step 3–4. Логику (`rank.go`) на этом шаге не менять.

- [ ] **Step 5: Зафиксировать калибровку (если правились пороги)**

Если `config.go` менялся:

```bash
git add internal/service/screener/dividend/rank/config.go
git commit -m "chore(screener): calibrate dividend ranking thresholds on live data"
```

Если правок нет — коммита нет, факт подтверждения единиц/гейта фиксируется в описании PR и памяти.

- [ ] **Step 6: Удалить временный диагностик**

```bash
rm -rf cmd/divcheck
rm -f divcheck
git status --short
```
Expected: `cmd/divcheck` и бинарь `divcheck` отсутствуют; рабочее дерево без временных файлов.

- [ ] **Step 7: Финальный гейт**

Run: `./bin/mage ci`
Expected: PASS (lint + race-тесты + mock-drift).

---

## Known limitations (сознательно НЕ адресуются в этом плане)

- **Missing net debt → оптимистичный Safety.** `leverageScore(0) == 0.9`: у компании с отсутствующим (0) `NetDebtToEbitda` пилляр Safety получает почти максимум, как будто долг близок к нулю. Для банков это не имеет значения (метрика к ним неприменима), а yield-trap gate всё ещё ловит `NetDebtToEbitda > 3` у высокодоходных имён с ПРИСУТСТВУЮЩИМ долгом. Риск скрытого рычага у имени с невыгруженным долгом остаётся — осознанно оставлен, т.к. отличить «нет данных» от «реального нуля» на proto3-поле нельзя, а нейтрализация `0 → 0.5` штрафовала бы легитимные компании с реально низким долгом. Фиксируется как явное ограничение, не как YAGNI ([[feedback_dont_defer_safety_risks]]).
- **`indicators.Percentile` (R-7)** по-прежнему не используется ядром (Minor из финального ревью ветки) — вне scope.

---

## Self-Review

- **Покрытие проблемы:** главный дефект (gate `EbitdaTtm>0 && payout>0` режет 80% вселенной, включая SBER/LKOH) закрыт Task 1; мёртвый пилляр DivGrowth — Task 2; подтверждение на живых данных с явными критериями приёмки — Task 3. Единицы полей уже подтверждены (в Global Constraints), отдельная задача не нужна.
- **Плейсхолдеры:** нет — весь код и все команды приведены дословно; единственная условная ветка (правка порогов) обусловлена наблюдаемым критерием в Step 4, а не «доработать позже».
- **Согласованность типов:** `AssetUID` (не `AssetUid`) везде; `sustainabilityPayout(float64, Config) float64`, `percentileOrNeutral([]float64, float64) float64`, `hasSpread([]float64) bool`, `gate(*model.Fundamentals, Config) (string, bool)`, `Rank([]*model.Fundamentals, Config) []ScoredCompany` — имена и сигнатуры совпадают между задачами и с существующим кодом.
- **Ломающиеся тесты учтены:** устаревший `TestRank_MissingData` инвертирован в Task 1 Step 1 (иначе он бы падал после снятия гейта); прочие существующие тесты используют `payout > 0` и не задеты нейтральными ветками.
