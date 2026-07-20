# Dividend Fundamental Screener — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Построить дискавери-скринер дивидендных акций по всей Мосбирже с композитным ранжированием качества, доступный как бот-команда `/dividend_screener`, чей рейтинг питает `Score` в Golden X.

**Architecture:** Чистое ядро ранжирования (`rank`) без I/O, поверх него оркестрация со shared-кэшем (`RankProvider`), два потребителя — бот-команда и Golden X. Фундаменталка тянется одним набором батч-RPC и кэшируется (TTL 24ч), чистые `Detect`/`Classify` Golden X остаются без I/O — бонус прокидывается как данные.

**Tech Stack:** Go 1.25, gRPC (Tinkoff Invest `GetAssetFundamentals`), Telegram Bot API, mockery v2, mage (`./bin/mage ci`).

## Global Constraints

- Go 1.25; идиоматичный Go, MixedCaps, ошибки — обёрнутый `%w`.
- `go build ./internal/... ./pkg/... ./cmd/...` (НЕ `./...` — падает на `magefiles`).
- Гейт: `./bin/mage ci` = lint + `go test -race ./...` + mock-drift. После правки мок-интерфейса — `./bin/mage mocks`.
- Пакеты именуются по смыслу; никаких `utils`/`helpers`.
- Валюта вселенной — только `rub` (уже фильтруется в `converter.ConvertSharesFromPb`).
- `detector.Detect` и `classifier.Classify` ОБЯЗАНЫ остаться чистыми (без I/O, без времени). Фунд-бонус входит как данные.
- **Единицы фундаментальных полей не подтверждены** и валидируются в Task 8: план исходит из допущения «yield и payout в процентах» (8.0 = 8%, 60.0 = 60%); все пороги — именованные константы в `rank.DefaultConfig()` для лёгкой калибровки.

---

## File Structure

**Создаются:**
- `internal/model/fundamentals.go` — `model.Fundamentals` (подмножество полей).
- `pkg/indicators/percentile.go` (+ `_test.go`) — общий R-7 перцентиль.
- `internal/service/screener/dividend/rank/rank.go` (+ `_test.go`) — чистое ядро: gate + столпы + composite.
- `internal/service/screener/dividend/rank/config.go` — веса/пороги (`Config`, `DefaultConfig`).
- `internal/service/screener/dividend/types.go` — `Screener` интерфейс, `instrumentsClient` интерфейс, `RankProvider` интерфейс.
- `internal/service/screener/dividend/service.go` (+ `_test.go`) — оркестрация + кэш/TTL, `Send`, `Top`, `RankBonus`.
- `internal/service/screener/dividend/notification/telegram.go` (+ `_test.go`) — рендер Top-N.
- `internal/service/screener/dividend/mocks/` — сгенерированный мок `instrumentsClient`.

**Модифицируются:**
- `internal/model/share.go` — +`AssetUid`, +`DivYieldFlag`.
- `internal/converter/share.go` — мапит два новых поля.
- `pkg/client/grpc/instruments_service_client.go` — +метод `GetAssetFundamentals`, +в интерфейс.
- `internal/converter/` — новый `ConvertFundamentalsFromPb`.
- `.mockery.yaml` — +пакет dividend.
- `internal/service/trading_strategy/golden_x/model/trade_result.go` — `ShareResult` +`FundamentalBonus int`.
- `internal/service/trading_strategy/golden_x/model/pipeline_result.go` — `DetectResult` +`FundamentalBonus int`.
- `internal/service/trading_strategy/golden_x/classifier/classifier.go` — `signalScore` учитывает бонус.
- `internal/service/trading_strategy/golden_x/detector/detect_all.go` — прокинуть бонус в `DetectResult`.
- `internal/service/trading_strategy/golden_x/types.go` — `WithRankProvider` Option + поле + no-op дефолт.
- `internal/service/trading_strategy/golden_x/trade.go` — опросить провайдера, прокинуть per-share бонус.
- `internal/service/telegram_commands/commands.go` — case `/dividend_screener` + help.
- `internal/service_provider/service.go` — построить и провязать dividend-сервис.
- `docs/golden_x/strategy.md`, `docs/golden_x/settings.md`, легенда нотификации.

---

## Task 1: Расширить `model.Share` полями actives-uid и div-flag

**Files:**
- Modify: `internal/model/share.go`
- Modify: `internal/converter/share.go`
- Test: `internal/converter/share_test.go` (создать, если нет)

**Interfaces:**
- Produces: `model.Share.AssetUid string`, `model.Share.DivYieldFlag bool`; `converter.ConvertShareFromPb` мапит оба.

- [ ] **Step 1: Написать падающий тест**

Создать `internal/converter/share_test.go`:

```go
package converter

import (
	"testing"

	investapi "tinvest/internal/pb/v1"
)

func TestConvertShareFromPb_MapsAssetUidAndDivFlag(t *testing.T) {
	in := &investapi.Share{
		Uid:          "instr-uid-1",
		AssetUid:     "asset-uid-1",
		Ticker:       "LKOH",
		Currency:     "rub",
		DivYieldFlag: true,
	}

	got := ConvertShareFromPb(in)

	if got.AssetUid != "asset-uid-1" {
		t.Fatalf("AssetUid = %q, want %q", got.AssetUid, "asset-uid-1")
	}
	if !got.DivYieldFlag {
		t.Fatalf("DivYieldFlag = false, want true")
	}
	if got.ID != "instr-uid-1" {
		t.Fatalf("ID = %q, want %q", got.ID, "instr-uid-1")
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что не компилируется/падает**

Run: `go test ./internal/converter/ -run TestConvertShareFromPb_MapsAssetUidAndDivFlag`
Expected: FAIL — `got.AssetUid undefined` (поля ещё нет).

- [ ] **Step 3: Добавить поля в модель**

В `internal/model/share.go` в конец структуры `Share`:

```go
	AssetUid                string
	DivYieldFlag            bool
```

- [ ] **Step 4: Замапить в конвертере**

В `internal/converter/share.go`, в `ConvertShareFromPb`, добавить в возвращаемый литерал:

```go
		AssetUid:     share.AssetUid,
		DivYieldFlag: share.DivYieldFlag,
```

- [ ] **Step 5: Запустить тест — зелёный**

Run: `go test ./internal/converter/ -run TestConvertShareFromPb_MapsAssetUidAndDivFlag -v`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add internal/model/share.go internal/converter/share.go internal/converter/share_test.go
git commit -m "feat(model): carry AssetUid and DivYieldFlag on Share"
```

---

## Task 2: `model.Fundamentals` + grpc-обёртка `GetAssetFundamentals` (батчи)

**Files:**
- Create: `internal/model/fundamentals.go`
- Create: `internal/converter/fundamentals.go`
- Create: `internal/converter/fundamentals_test.go`
- Modify: `pkg/client/grpc/instruments_service_client.go`

**Interfaces:**
- Produces:
  - `model.Fundamentals` (см. Step 1).
  - `converter.ConvertFundamentalsFromPb(in []*investapi.GetAssetFundamentalsResponse_StatisticResponse) []*model.Fundamentals`.
  - Метод интерфейса `InstrumentsServiceClient`: `GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error)` — батчит по 100.

- [ ] **Step 1: Определить модель**

Создать `internal/model/fundamentals.go`:

```go
package model

// Fundamentals — подмножество фундаментальных показателей актива из
// Tinkoff GetAssetFundamentals, что использует дивидендный скринер.
type Fundamentals struct {
	AssetUid string

	ForwardAnnualDividendYield      float64 // форвардная дивдоходность, %
	DividendYieldDailyTtm           float64 // текущая дивдоходность TTM, %
	DividendPayoutRatioFy           float64 // payout ratio, %
	FiveYearsAverageDividendYield   float64 // средняя дивдоходность за 5 лет, %
	FiveYearAnnualDividendGrowthRate float64 // среднегодовой рост дивиденда за 5 лет
	DividendRateTtm                 float64 // дивиденд на акцию TTM

	NetDebtToEbitda            float64
	TotalDebtToEquityMrq       float64
	FixedChargeCoverageRatioFy float64
	CurrentRatioMrq            float64

	Roic         float64
	Roe          float64
	NetMarginMrq float64
	EbitdaTtm    float64
	RevenueTtm   float64
	FreeCashFlowTtm float64

	EvToEbitdaMrq          float64
	PeRatioTtm             float64
	PriceToBookTtm         float64
	PriceToFreeCashFlowTtm float64
}
```

- [ ] **Step 2: Написать падающий тест конвертера**

Создать `internal/converter/fundamentals_test.go`:

```go
package converter

import (
	"testing"

	investapi "tinvest/internal/pb/v1"
)

func TestConvertFundamentalsFromPb(t *testing.T) {
	in := []*investapi.GetAssetFundamentalsResponse_StatisticResponse{
		{
			AssetUid:                   "asset-1",
			ForwardAnnualDividendYield: 8.5,
			DividendPayoutRatioFy:      55,
			NetDebtToEbitda:            1.2,
			Roic:                       0.19,
			EvToEbitdaMrq:              3.4,
			FreeCashFlowTtm:            1000,
		},
	}

	got := ConvertFundamentalsFromPb(in)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].AssetUid != "asset-1" || got[0].ForwardAnnualDividendYield != 8.5 || got[0].NetDebtToEbitda != 1.2 {
		t.Fatalf("mismatch: %+v", got[0])
	}
}
```

- [ ] **Step 3: Запустить — падает**

Run: `go test ./internal/converter/ -run TestConvertFundamentalsFromPb`
Expected: FAIL — `ConvertFundamentalsFromPb undefined`.

- [ ] **Step 4: Реализовать конвертер**

Создать `internal/converter/fundamentals.go` (мапит поля один-в-один из `GetAssetFundamentalsResponse_StatisticResponse` в `model.Fundamentals`; имена полей pb см. в `internal/pb/v1/instruments.pb.go`):

```go
package converter

import (
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

func ConvertFundamentalsFromPb(in []*investapi.GetAssetFundamentalsResponse_StatisticResponse) []*model.Fundamentals {
	res := make([]*model.Fundamentals, 0, len(in))
	for _, f := range in {
		if f == nil {
			continue
		}
		res = append(res, &model.Fundamentals{
			AssetUid:                         f.AssetUid,
			ForwardAnnualDividendYield:       f.ForwardAnnualDividendYield,
			DividendYieldDailyTtm:            f.DividendYieldDailyTtm,
			DividendPayoutRatioFy:            f.DividendPayoutRatioFy,
			FiveYearsAverageDividendYield:    f.FiveYearsAverageDividendYield,
			FiveYearAnnualDividendGrowthRate: f.FiveYearAnnualDividendGrowthRate,
			DividendRateTtm:                  f.DividendRateTtm,
			NetDebtToEbitda:                  f.NetDebtToEbitda,
			TotalDebtToEquityMrq:             f.TotalDebtToEquityMrq,
			FixedChargeCoverageRatioFy:       f.FixedChargeCoverageRatioFy,
			CurrentRatioMrq:                  f.CurrentRatioMrq,
			Roic:                             f.Roic,
			Roe:                              f.Roe,
			NetMarginMrq:                     f.NetMarginMrq,
			EbitdaTtm:                        f.EbitdaTtm,
			RevenueTtm:                       f.RevenueTtm,
			FreeCashFlowTtm:                  f.FreeCashFlowTtm,
			EvToEbitdaMrq:                    f.EvToEbitdaMrq,
			PeRatioTtm:                       f.PeRatioTtm,
			PriceToBookTtm:                   f.PriceToBookTtm,
			PriceToFreeCashFlowTtm:           f.PriceToFreeCashFlowTtm,
		})
	}
	return res
}
```

- [ ] **Step 5: Тест конвертера зелёный**

Run: `go test ./internal/converter/ -run TestConvertFundamentalsFromPb -v`
Expected: PASS

- [ ] **Step 6: Добавить grpc-обёртку с батчингом**

В `pkg/client/grpc/instruments_service_client.go`:

1. В интерфейс `InstrumentsServiceClient` добавить строку:
```go
	GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error)
```
2. Реализация (в конец файла); батч 100 — лимит API:
```go
func (c *instrumentsServiceClient) GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error) {
	const batchSize = 100
	res := make([]*model.Fundamentals, 0, len(assetUIDs))
	for start := 0; start < len(assetUIDs); start += batchSize {
		end := start + batchSize
		if end > len(assetUIDs) {
			end = len(assetUIDs)
		}
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		resp, err := c.instrumentsAPI.GetAssetFundamentals(
			reqCtx,
			&investapi.GetAssetFundamentalsRequest{Assets: assetUIDs[start:end]},
			NewRPCCredential(c.auth),
		)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("failed to request fundamentals: %w", err)
		}
		res = append(res, converter.ConvertFundamentalsFromPb(resp.Fundamentals)...)
	}
	return res, nil
}
```

- [ ] **Step 7: Проверить сборку**

Run: `go build ./internal/... ./pkg/...`
Expected: успех.

- [ ] **Step 8: Коммит**

```bash
git add internal/model/fundamentals.go internal/converter/fundamentals.go internal/converter/fundamentals_test.go pkg/client/grpc/instruments_service_client.go
git commit -m "feat(grpc): batched GetAssetFundamentals + Fundamentals model"
```

---

## Task 3: Общий R-7 перцентиль в `pkg/indicators`

**Files:**
- Create: `pkg/indicators/percentile.go`
- Create: `pkg/indicators/percentile_test.go`

**Interfaces:**
- Produces: `indicators.Percentile(sortedAsc []float64, p float64) float64` — R-7 (NumPy/Excel default), `p ∈ [0,100]`; пустой вход → 0.
- Produces: `indicators.PercentileRank(values []float64, x float64) float64` — доля значений `< x` в `values`, результат `∈ [0,1]`; пустой вход → 0.

- [ ] **Step 1: Тесты**

Создать `pkg/indicators/percentile_test.go`:

```go
package indicators

import (
	"math"
	"testing"
)

func TestPercentile_R7(t *testing.T) {
	s := []float64{1, 2, 3, 4}
	if got := Percentile(s, 50); math.Abs(got-2.5) > 1e-9 {
		t.Fatalf("p50 = %v, want 2.5", got)
	}
	if got := Percentile(s, 0); got != 1 {
		t.Fatalf("p0 = %v, want 1", got)
	}
	if got := Percentile(s, 100); got != 4 {
		t.Fatalf("p100 = %v, want 4", got)
	}
	if got := Percentile(nil, 50); got != 0 {
		t.Fatalf("empty = %v, want 0", got)
	}
}

func TestPercentileRank(t *testing.T) {
	vals := []float64{10, 20, 30, 40}
	if got := PercentileRank(vals, 25); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("rank(25) = %v, want 0.5", got)
	}
	if got := PercentileRank(vals, 5); got != 0 {
		t.Fatalf("rank(5) = %v, want 0", got)
	}
	if got := PercentileRank(vals, 100); math.Abs(got-1.0) > 1e-9 {
		t.Fatalf("rank(100) = %v, want 1", got)
	}
}
```

- [ ] **Step 2: Запустить — падает**

Run: `go test ./pkg/indicators/ -run 'TestPercentile'`
Expected: FAIL — `Percentile undefined`.

- [ ] **Step 3: Реализация**

Создать `pkg/indicators/percentile.go`:

```go
package indicators

import (
	"math"
	"sort"
)

// Percentile returns the R-7 (linear-interpolation) percentile of an
// already-ascending-sorted slice. p is in [0, 100]. Empty input returns 0.
func Percentile(sortedAsc []float64, p float64) float64 {
	n := len(sortedAsc)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sortedAsc[0]
	}
	rank := (p / 100.0) * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sortedAsc[lo]
	}
	weight := rank - float64(lo)
	return sortedAsc[lo]*(1-weight) + sortedAsc[hi]*weight
}

// PercentileRank returns the fraction of values strictly less than x, in [0,1].
// Empty input returns 0. Input is not mutated.
func PercentileRank(values []float64, x float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	count := 0
	for _, v := range sorted {
		if v < x {
			count++
		}
	}
	return float64(count) / float64(len(sorted))
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test ./pkg/indicators/ -run 'TestPercentile' -v`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add pkg/indicators/percentile.go pkg/indicators/percentile_test.go
git commit -m "feat(indicators): shared R-7 Percentile and PercentileRank"
```

---

## Task 4: Чистое ядро ранжирования `rank`

**Files:**
- Create: `internal/service/screener/dividend/rank/config.go`
- Create: `internal/service/screener/dividend/rank/rank.go`
- Create: `internal/service/screener/dividend/rank/rank_test.go`

**Interfaces:**
- Consumes: `model.Fundamentals`, `indicators.Percentile`/`PercentileRank`.
- Produces:
  - `rank.Config` + `rank.DefaultConfig() Config` (веса/пороги).
  - `rank.ScoredCompany{AssetUid string; Composite float64; Sustainability, Safety, DivGrowth, Quality, Valuation, YieldScore float64; YieldTrap bool; GateReason string}`.
  - `rank.Rank(universe []*model.Fundamentals, cfg Config) []rank.ScoredCompany` — отсеянные получают `GateReason != ""` и `Composite == 0`; выжившие отсортированы по `Composite` убыв.; стабильная сортировка (тай-брейк по `AssetUid`).

**Алгоритм (2 прохода):** (1) прогнать gate, собрать выживших; (2) по выжившим посчитать перцентильные пулы (DivGrowth, Quality=Roic, Valuation=EvToEbitda) и composite; отсеянные идут в хвост с `Composite=0`.

- [ ] **Step 1: Config с именованными порогами**

Создать `internal/service/screener/dividend/rank/config.go`:

```go
package rank

// Config держит веса столпов (сумма = 1.0) и пороги ворот. Все значения —
// точки калибровки (Task 8 сверяет с живыми данными). Единицы: yield и
// payout в процентах (8.0 = 8%, 60 = 60%).
type Config struct {
	WeightSustainability float64
	WeightSafety         float64
	WeightDivGrowth      float64
	WeightQuality        float64
	WeightValuation      float64
	WeightYield          float64

	MaxNetDebtToEbitda float64 // выше — gate highLeverage
	MaxPayoutPct       float64 // выше — gate unsustainablePayout
	YieldTrapMinYield  float64 // ниже этого yield trap не рассматривается
	YieldCapPct        float64 // потолок для yield-подсчёта

	PayoutIdealLow  float64 // нижняя граница идеальной зоны payout
	PayoutIdealHigh float64 // верхняя граница идеальной зоны payout
}

func DefaultConfig() Config {
	return Config{
		WeightSustainability: 0.30,
		WeightSafety:         0.25,
		WeightDivGrowth:      0.15,
		WeightQuality:        0.15,
		WeightValuation:      0.10,
		WeightYield:          0.05,

		MaxNetDebtToEbitda: 4.0,
		MaxPayoutPct:       120.0,
		YieldTrapMinYield:  20.0,
		YieldCapPct:        14.0,

		PayoutIdealLow:  30.0,
		PayoutIdealHigh: 60.0,
	}
}
```

- [ ] **Step 2: Тесты ядра (gate + trap + порядок + missing)**

Создать `internal/service/screener/dividend/rank/rank_test.go`:

```go
package rank

import (
	"testing"

	"tinvest/internal/model"
)

func byUID(scored []ScoredCompany) map[string]ScoredCompany {
	m := make(map[string]ScoredCompany, len(scored))
	for _, s := range scored {
		m[s.AssetUid] = s
	}
	return m
}

func TestRank_GateHighLeverage(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUid: "lev", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 50, NetDebtToEbitda: 5, EbitdaTtm: 100, Roic: 0.1},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["lev"].GateReason == "" {
		t.Fatalf("expected gate for high leverage, got none")
	}
	if got["lev"].Composite != 0 {
		t.Fatalf("gated composite = %v, want 0", got["lev"].Composite)
	}
}

func TestRank_GateNoDividend(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUid: "nodiv", ForwardAnnualDividendYield: 0, DividendYieldDailyTtm: 0, DividendPayoutRatioFy: 50, NetDebtToEbitda: 1, EbitdaTtm: 100},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nodiv"].GateReason == "" {
		t.Fatalf("expected gate for no dividend")
	}
}

func TestRank_YieldTrap(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUid: "trap", ForwardAnnualDividendYield: 25, DividendPayoutRatioFy: 110, NetDebtToEbitda: 3.5, EbitdaTtm: 100, FreeCashFlowTtm: -50, Roic: 0.05},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if !got["trap"].YieldTrap || got["trap"].GateReason == "" {
		t.Fatalf("expected yield-trap gate, got %+v", got["trap"])
	}
}

func TestRank_MissingData(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUid: "nodata", ForwardAnnualDividendYield: 8, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nodata"].GateReason == "" {
		t.Fatalf("expected gate for missing data")
	}
}

func TestRank_OrdersSurvivorsByComposite(t *testing.T) {
	u := []*model.Fundamentals{
		// сильная: низкий долг, идеальный payout, высокий ROIC, дешёвая, рост дивиденда
		{AssetUid: "strong", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, EbitdaTtm: 100, Roic: 0.25, EvToEbitdaMrq: 3, FreeCashFlowTtm: 500, FiveYearAnnualDividendGrowthRate: 0.15},
		// слабая: высокий долг (но <4), высокий payout, низкий ROIC, дорогая
		{AssetUid: "weak", ForwardAnnualDividendYield: 7, DividendPayoutRatioFy: 95, NetDebtToEbitda: 3.5, EbitdaTtm: 100, Roic: 0.03, EvToEbitdaMrq: 12, FreeCashFlowTtm: 10, FiveYearAnnualDividendGrowthRate: -0.05},
	}
	scored := Rank(u, DefaultConfig())
	if scored[0].AssetUid != "strong" {
		t.Fatalf("order = %v, want strong first", []string{scored[0].AssetUid, scored[1].AssetUid})
	}
	if scored[0].Composite <= scored[1].Composite {
		t.Fatalf("composite not ordered: %v <= %v", scored[0].Composite, scored[1].Composite)
	}
}
```

- [ ] **Step 3: Запустить — падает**

Run: `go test ./internal/service/screener/dividend/rank/`
Expected: FAIL — `Rank undefined`.

- [ ] **Step 4: Реализовать ядро**

Создать `internal/service/screener/dividend/rank/rank.go`:

```go
package rank

import (
	"sort"

	"tinvest/internal/model"
	"tinvest/pkg/indicators"
)

// ScoredCompany — результат ранжирования одной компании. Отсеянные воротами
// имеют GateReason != "" и Composite == 0.
type ScoredCompany struct {
	AssetUid string
	Composite float64

	Sustainability float64
	Safety         float64
	DivGrowth      float64
	Quality        float64
	Valuation      float64
	YieldScore     float64

	YieldTrap  bool
	GateReason string
}

// Причины отсева (стабильные строки для вывода и тестов).
const (
	reasonNoDividend      = "нет дивиденда"
	reasonMissingData     = "нет ключевых данных"
	reasonHighLeverage    = "долг > порога"
	reasonUnsustainable   = "payout > порога"
	reasonYieldTrap       = "yield trap"
)

func yieldOf(f *model.Fundamentals) float64 {
	if f.ForwardAnnualDividendYield > 0 {
		return f.ForwardAnnualDividendYield
	}
	return f.DividendYieldDailyTtm
}

// gate возвращает (reason, isTrap). Пустой reason => компания проходит.
func gate(f *model.Fundamentals, cfg Config) (string, bool) {
	y := yieldOf(f)
	if y <= 0 {
		return reasonNoDividend, false
	}
	if f.EbitdaTtm <= 0 || f.DividendPayoutRatioFy <= 0 {
		return reasonMissingData, false
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

// payoutFit: 1.0 в идеальной зоне, линейно к 0 у краёв (0 и MaxPayoutPct).
func payoutFit(payout float64, cfg Config) float64 {
	switch {
	case payout >= cfg.PayoutIdealLow && payout <= cfg.PayoutIdealHigh:
		return 1.0
	case payout < cfg.PayoutIdealLow:
		return clamp01(payout / cfg.PayoutIdealLow)
	default: // > ideal high
		span := cfg.MaxPayoutPct - cfg.PayoutIdealHigh
		if span <= 0 {
			return 0
		}
		return clamp01((cfg.MaxPayoutPct - payout) / span)
	}
}

// leverageScore: чем меньше долг, тем лучше. Чистый кэш (<0) — максимум.
func leverageScore(nd float64) float64 {
	switch {
	case nd < 0:
		return 1.0
	case nd <= 1:
		return 0.9
	case nd <= 2:
		return 0.7
	case nd <= 3:
		return 0.4
	default:
		return 0.15
	}
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func Rank(universe []*model.Fundamentals, cfg Config) []ScoredCompany {
	survivors := make([]*model.Fundamentals, 0, len(universe))
	gated := make([]ScoredCompany, 0)

	for _, f := range universe {
		reason, trap := gate(f, cfg)
		if reason != "" {
			gated = append(gated, ScoredCompany{AssetUid: f.AssetUid, GateReason: reason, YieldTrap: trap})
			continue
		}
		survivors = append(survivors, f)
	}

	// Перцентильные пулы по выжившим.
	divGrowth := make([]float64, len(survivors))
	roic := make([]float64, len(survivors))
	evEbitda := make([]float64, len(survivors))
	for i, f := range survivors {
		divGrowth[i] = f.FiveYearAnnualDividendGrowthRate
		roic[i] = qualityMetric(f)
		evEbitda[i] = f.EvToEbitdaMrq
	}

	scored := make([]ScoredCompany, 0, len(survivors))
	for _, f := range survivors {
		sc := ScoredCompany{AssetUid: f.AssetUid}
		sc.Sustainability = 0.7*payoutFit(f.DividendPayoutRatioFy, cfg) + 0.3*boolScore(f.FreeCashFlowTtm > 0)
		sc.Safety = leverageScore(f.NetDebtToEbitda)
		sc.DivGrowth = indicators.PercentileRank(divGrowth, f.FiveYearAnnualDividendGrowthRate)
		sc.Quality = indicators.PercentileRank(roic, qualityMetric(f))
		sc.Valuation = 1 - indicators.PercentileRank(evEbitda, f.EvToEbitdaMrq) // ниже EV/EBITDA — лучше
		sc.YieldScore = clamp01(minf(yieldOf(f), cfg.YieldCapPct) / cfg.YieldCapPct)

		sc.Composite = 100 * (cfg.WeightSustainability*sc.Sustainability +
			cfg.WeightSafety*sc.Safety +
			cfg.WeightDivGrowth*sc.DivGrowth +
			cfg.WeightQuality*sc.Quality +
			cfg.WeightValuation*sc.Valuation +
			cfg.WeightYield*sc.YieldScore)
		scored = append(scored, sc)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Composite != scored[j].Composite {
			return scored[i].Composite > scored[j].Composite
		}
		return scored[i].AssetUid < scored[j].AssetUid
	})

	return append(scored, gated...)
}

// qualityMetric: ROIC, с фолбэком на ROE, если ROIC не задан.
func qualityMetric(f *model.Fundamentals) float64 {
	if f.Roic != 0 {
		return f.Roic
	}
	return f.Roe
}

func boolScore(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 5: Тесты зелёные**

Run: `go test ./internal/service/screener/dividend/rank/ -v`
Expected: PASS (все 5).

- [ ] **Step 6: Коммит**

```bash
git add internal/service/screener/dividend/rank/
git commit -m "feat(screener): pure dividend ranking core (gate + pillars + composite)"
```

---

## Task 5: Оркестрация + `RankProvider` (кэш/TTL, Top, RankBonus)

**Files:**
- Create: `internal/service/screener/dividend/types.go`
- Create: `internal/service/screener/dividend/service.go`
- Create: `internal/service/screener/dividend/service_test.go`
- Modify: `.mockery.yaml`
- Create (generated): `internal/service/screener/dividend/mocks/mock_instrumentsClient.go`

**Interfaces:**
- Consumes: `rank.Rank`, `rank.ScoredCompany`, `model.Share`, `model.Fundamentals`, `telegram.Client`.
- Produces:
  - интерфейс `Screener interface { Send(ctx context.Context, tg telegram.Client) error }`.
  - интерфейс `RankProvider interface { RankBonus(instrumentID string) int }`.
  - `NewService(client instrumentsClient, opts ...Option) *service` (реализует оба).
  - `(*service).Top(ctx context.Context, n int) ([]RankedShare, Stats, error)` для нотификации.
  - `RankedShare{Share *model.Share; Scored rank.ScoredCompany}`; `Stats{Universe, Ranked, Gated int; ByReason map[string]int}`.

**Кэш:** `refresh` строит `map[instrumentID]rank.ScoredCompany` + отсортированный `[]RankedShare` + `Stats` + `time.Time`. `RankBonus` и `Top` вызывают `ensureFresh` (рефреш при пустом кэше или старше TTL). `RankBonus` дополнительно never-panics: при отсутствии/ошибке → 0.

- [ ] **Step 1: types.go**

Создать `internal/service/screener/dividend/types.go`:

```go
package dividend

import (
	"context"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
	"tinvest/pkg/client/telegram"
)

const defaultTTL = 24 * time.Hour
const defaultTopN = 15

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*model.Share, error)
	GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error)
}

// Screener — потребитель бот-команды.
type Screener interface {
	Send(ctx context.Context, tg telegram.Client) error
}

// RankProvider — узкий потребитель Golden X.
type RankProvider interface {
	RankBonus(instrumentID string) int
}

type RankedShare struct {
	Share  *model.Share
	Scored rank.ScoredCompany
}

type Stats struct {
	Universe int
	Ranked   int
	Gated    int
	ByReason map[string]int
}

type Option func(*service)

func WithConfig(c rank.Config) Option { return func(s *service) { s.cfg = c } }
func WithTTL(d time.Duration) Option  { return func(s *service) { s.ttl = d } }
```

- [ ] **Step 2: Тест сервиса (маппинг ранг→бонус, staleness, отсев)**

Создать `internal/service/screener/dividend/service_test.go`:

```go
package dividend

import (
	"context"
	"testing"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/mocks"

	"github.com/stretchr/testify/mock"
)

func fundUniverse() ([]*model.Share, []*model.Fundamentals) {
	shares := []*model.Share{
		{ID: "i-strong", AssetUid: "a-strong", Name: "Strong", DivYieldFlag: true},
		{ID: "i-mid", AssetUid: "a-mid", Name: "Mid", DivYieldFlag: true},
		{ID: "i-weak", AssetUid: "a-weak", Name: "Weak", DivYieldFlag: true},
		{ID: "i-gated", AssetUid: "a-gated", Name: "Gated", DivYieldFlag: true},
	}
	funds := []*model.Fundamentals{
		{AssetUid: "a-strong", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.3, EbitdaTtm: 100, Roic: 0.25, EvToEbitdaMrq: 3, FreeCashFlowTtm: 500, FiveYearAnnualDividendGrowthRate: 0.2},
		{AssetUid: "a-mid", ForwardAnnualDividendYield: 8, DividendPayoutRatioFy: 55, NetDebtToEbitda: 1.5, EbitdaTtm: 100, Roic: 0.12, EvToEbitdaMrq: 6, FreeCashFlowTtm: 100, FiveYearAnnualDividendGrowthRate: 0.05},
		{AssetUid: "a-weak", ForwardAnnualDividendYield: 7, DividendPayoutRatioFy: 95, NetDebtToEbitda: 3.5, EbitdaTtm: 100, Roic: 0.03, EvToEbitdaMrq: 12, FreeCashFlowTtm: 10, FiveYearAnnualDividendGrowthRate: -0.05},
		{AssetUid: "a-gated", ForwardAnnualDividendYield: 30, DividendPayoutRatioFy: 130, NetDebtToEbitda: 5, EbitdaTtm: 100, FreeCashFlowTtm: -10},
	}
	return shares, funds
}

func newMockedService(t *testing.T) *service {
	shares, funds := fundUniverse()
	m := mocks.NewMockinstrumentsClient(t)
	m.On("Shares", mock.Anything).Return(shares, nil)
	m.On("GetAssetFundamentals", mock.Anything, mock.Anything).Return(funds, nil)
	return NewService(m)
}

func TestRankBonus_TopGetsMorePoints(t *testing.T) {
	svc := newMockedService(t)

	strong := svc.RankBonus("i-strong")
	weak := svc.RankBonus("i-weak")
	gated := svc.RankBonus("i-gated")
	unknown := svc.RankBonus("i-does-not-exist")

	if !(strong >= weak) {
		t.Fatalf("strong bonus %d should be >= weak %d", strong, weak)
	}
	if strong < 1 || strong > 3 {
		t.Fatalf("strong bonus %d out of [1,3]", strong)
	}
	if gated != 0 {
		t.Fatalf("gated bonus = %d, want 0", gated)
	}
	if unknown != 0 {
		t.Fatalf("unknown bonus = %d, want 0", unknown)
	}
}

func TestTop_ReportsGateStats(t *testing.T) {
	svc := newMockedService(t)
	ranked, stats, err := svc.Top(context.Background(), 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 3 {
		t.Fatalf("ranked = %d, want 3 (one gated)", len(ranked))
	}
	if stats.Gated != 1 || stats.Universe != 4 {
		t.Fatalf("stats = %+v, want Gated 1 / Universe 4", stats)
	}
	if ranked[0].Share.ID != "i-strong" {
		t.Fatalf("top = %s, want i-strong", ranked[0].Share.ID)
	}
}
```

- [ ] **Step 3: Зарегистрировать мок и сгенерировать**

В `.mockery.yaml` добавить в `packages:`:

```yaml
  tinvest/internal/service/screener/dividend:
    config:
      all: false
    interfaces:
      instrumentsClient:
```

Затем:

Run: `./bin/mage mocks`
Expected: создан `internal/service/screener/dividend/mocks/mock_instrumentsClient.go`.

- [ ] **Step 4: Запустить тест — падает**

Run: `go test ./internal/service/screener/dividend/`
Expected: FAIL — `NewService`/`service` undefined.

- [ ] **Step 5: Реализовать сервис**

Создать `internal/service/screener/dividend/service.go`:

```go
package dividend

import (
	"context"
	"sort"
	"sync"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
	"tinvest/pkg/client/telegram"
)

type service struct {
	client instrumentsClient
	cfg    rank.Config
	ttl    time.Duration

	mu       sync.RWMutex
	bonusByID map[string]int
	ranked    []RankedShare
	stats     Stats
	loadedAt  time.Time
}

func NewService(client instrumentsClient, opts ...Option) *service {
	s := &service{
		client: client,
		cfg:    rank.DefaultConfig(),
		ttl:    defaultTTL,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *service) ensureFresh(ctx context.Context) error {
	s.mu.RLock()
	fresh := !s.loadedAt.IsZero() && time.Since(s.loadedAt) < s.ttl
	s.mu.RUnlock()
	if fresh {
		return nil
	}
	return s.refresh(ctx)
}

func (s *service) refresh(ctx context.Context) error {
	shares, err := s.client.Shares(ctx)
	if err != nil {
		return err
	}

	dividendShares := make([]*model.Share, 0, len(shares))
	uids := make([]string, 0, len(shares))
	shareByAsset := make(map[string]*model.Share, len(shares))
	for _, sh := range shares {
		if !sh.DivYieldFlag || sh.AssetUid == "" {
			continue
		}
		dividendShares = append(dividendShares, sh)
		uids = append(uids, sh.AssetUid)
		shareByAsset[sh.AssetUid] = sh
	}

	funds, err := s.client.GetAssetFundamentals(ctx, uids)
	if err != nil {
		return err
	}

	scored := rank.Rank(funds, s.cfg)

	// Разделить на выживших (по порядку) и посчитать перцентильный ранг.
	survivors := make([]rank.ScoredCompany, 0, len(scored))
	stats := Stats{Universe: len(dividendShares), ByReason: map[string]int{}}
	for _, sc := range scored {
		if sc.GateReason != "" {
			stats.Gated++
			stats.ByReason[sc.GateReason]++
			continue
		}
		survivors = append(survivors, sc)
	}
	stats.Ranked = len(survivors)

	ranked := make([]RankedShare, 0, len(survivors))
	bonusByID := make(map[string]int, len(survivors))
	total := len(survivors)
	for i, sc := range survivors {
		sh := shareByAsset[sc.AssetUid]
		if sh == nil {
			continue
		}
		ranked = append(ranked, RankedShare{Share: sh, Scored: sc})
		bonusByID[sh.ID] = bonusFromRank(i, total)
	}

	s.mu.Lock()
	s.ranked = ranked
	s.bonusByID = bonusByID
	s.stats = stats
	s.loadedAt = time.Now()
	s.mu.Unlock()
	return nil
}

// bonusFromRank: топ-дециль →3, топ-квартиль →2, топ-половина →1, иначе 0.
// idx 0 — лучший из total.
func bonusFromRank(idx, total int) int {
	if total <= 0 {
		return 0
	}
	q := float64(idx) / float64(total)
	switch {
	case q < 0.10:
		return 3
	case q < 0.25:
		return 2
	case q < 0.50:
		return 1
	default:
		return 0
	}
}

func (s *service) RankBonus(instrumentID string) int {
	if err := s.ensureFresh(context.Background()); err != nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bonusByID[instrumentID]
}

func (s *service) Top(ctx context.Context, n int) ([]RankedShare, Stats, error) {
	if err := s.ensureFresh(ctx); err != nil {
		return nil, Stats{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	top := s.ranked
	if n > 0 && len(top) > n {
		top = top[:n]
	}
	out := make([]RankedShare, len(top))
	copy(out, top)
	return out, s.stats, nil
}

var _ = sort.Ints // сохранить импорт, если не используется напрямую
```

Примечание: если линтер ругнётся на неиспользуемый `sort`, убрать импорт и последнюю строку.

- [ ] **Step 6: Тесты + race**

Run: `go test ./internal/service/screener/dividend/ -race -v`
Expected: PASS

- [ ] **Step 7: Коммит**

```bash
git add internal/service/screener/dividend/ .mockery.yaml
git commit -m "feat(screener): dividend RankProvider with cached universe ranking"
```

---

## Task 6: Бот-команда `/dividend_screener` + нотификация

**Files:**
- Create: `internal/service/screener/dividend/notification/telegram.go`
- Create: `internal/service/screener/dividend/notification/telegram_test.go`
- Modify: `internal/service/screener/dividend/service.go` (метод `Send`)
- Modify: `internal/service/telegram_commands/commands.go`
- Modify: `internal/service_provider/service.go`

**Interfaces:**
- Consumes: `[]dividend.RankedShare`, `dividend.Stats`.
- Produces: `notification.Render(ranked []RankedShare, stats Stats) string`; `(*service).Send(ctx, tg) error`.

- [ ] **Step 1: Тест нотификации (golden-string)**

Создать `internal/service/screener/dividend/notification/telegram_test.go`:

```go
package notification

import (
	"strings"
	"testing"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
)

func TestRender_ContainsHeaderTopAndStats(t *testing.T) {
	ranked := []RankedShare{
		{Share: &model.Share{Name: "Лукойл", Ticker: "LKOH"}, Scored: rank.ScoredCompany{Composite: 82.5}},
	}
	stats := Stats{Universe: 40, Ranked: 25, Gated: 15, ByReason: map[string]int{"yield trap": 5}}

	out := Render(ranked, stats)

	for _, want := range []string{"Дивидендный скринер", "Лукойл", "LKOH", "82", "Отсеяно", "yield trap"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n%s", want, out)
		}
	}
}
```

Примечание: `RankedShare` и `Stats` в этом пакете — алиасы на `dividend.RankedShare`/`dividend.Stats`. Чтобы избежать цикла импорта (`dividend` → `notification` → `dividend`), нотификация принимает **свои** типы. Решение: определить `Render` над `dividend`-типами, а пакет `notification` импортировать из `service.go` (`dividend` → `notification`, без обратной связи). Значит `RankedShare`/`Stats` в тесте — из пакета `dividend`. Скорректировать импорт в тесте на `dividend.RankedShare`/`dividend.Stats` и вызвать `notification.Render`.

- [ ] **Step 2: Переписать тест с правильными типами**

Заменить тело теста на использование внешних типов:

```go
package notification

import (
	"strings"
	"testing"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend"
	"tinvest/internal/service/screener/dividend/rank"
)

func TestRender_ContainsHeaderTopAndStats(t *testing.T) {
	ranked := []dividend.RankedShare{
		{Share: &model.Share{Name: "Лукойл", Ticker: "LKOH"}, Scored: rank.ScoredCompany{Composite: 82.5}},
	}
	stats := dividend.Stats{Universe: 40, Ranked: 25, Gated: 15, ByReason: map[string]int{"yield trap": 5}}

	out := Render(ranked, stats)

	for _, want := range []string{"Дивидендный скринер", "Лукойл", "LKOH", "82", "Отсеяно", "yield trap"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n%s", want, out)
		}
	}
}
```

- [ ] **Step 3: Запустить — падает**

Run: `go test ./internal/service/screener/dividend/notification/`
Expected: FAIL — `Render undefined`.

- [ ] **Step 4: Реализовать Render**

Создать `internal/service/screener/dividend/notification/telegram.go`:

```go
package notification

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"tinvest/internal/service/screener/dividend"
)

func Render(ranked []dividend.RankedShare, stats dividend.Stats) string {
	var b strings.Builder
	b.WriteString("<b>🏆 Дивидендный скринер (Мосбиржа)</b>\n\n")
	fmt.Fprintf(&b, "<i>Вселенная: %d · в рейтинге: %d · отсеяно: %d</i>\n\n", stats.Universe, stats.Ranked, stats.Gated)

	for i, rs := range ranked {
		f := rs.Scored
		trap := ""
		if f.YieldTrap {
			trap = " ⚠️"
		}
		fmt.Fprintf(&b, "<b>%d. %s</b> (%s)%s\n", i+1, htmlEscape(rs.Share.Name), rs.Share.Ticker, trap)
		fmt.Fprintf(&b, "   Композит: <b>%.0f</b>/100\n", f.Composite)
		fmt.Fprintf(&b, "   Устойчивость %.2f · Долг %.2f · Рост %.2f · Качество %.2f · Оценка %.2f\n\n",
			f.Sustainability, f.Safety, f.DivGrowth, f.Quality, f.Valuation)
	}

	if len(stats.ByReason) > 0 {
		b.WriteString("<i>Отсеяно по причинам:</i>\n")
		reasons := make([]string, 0, len(stats.ByReason))
		for r := range stats.ByReason {
			reasons = append(reasons, r)
		}
		sort.Strings(reasons)
		for _, r := range reasons {
			fmt.Fprintf(&b, "· %s — %d\n", r, stats.ByReason[r])
		}
	}

	fmt.Fprintf(&b, "\n<i>Данные на %s</i>", time.Now().Format("02.01.2006 15:04"))
	return b.String()
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
```

- [ ] **Step 5: Добавить `Send` в сервис**

В `internal/service/screener/dividend/service.go` добавить импорт `"tinvest/internal/service/screener/dividend/notification"` и метод:

```go
func (s *service) Send(ctx context.Context, tg telegram.Client) error {
	ranked, stats, err := s.Top(ctx, defaultTopN)
	if err != nil {
		return err
	}
	return tg.SendMessage(notification.Render(ranked, stats))
}
```

- [ ] **Step 6: Провязать команду**

В `internal/service/telegram_commands/commands.go`:

1. Импорт: `"tinvest/internal/service/screener/dividend"`.
2. В `Commands` добавить поле `screener dividend.Screener`.
3. В `New(...)` добавить параметр `sc dividend.Screener` и присвоить `screener: sc`.
4. В `helpText` добавить строку: `/dividend_screener — скринер дивидендных акций`.
5. В `switch` добавить:
```go
	case "/dividend_screener":
		c.runExclusive(ctx, cmd, tg, func(ctx context.Context) error {
			return c.screener.Send(ctx, tg)
		})
```

- [ ] **Step 7: Провязать в service_provider**

В `internal/service_provider/service.go` добавить геттер (по образцу `GetBondsTradingService`):

```go
func (*ServiceProvider) GetDividendScreener() dividend.Screener {
	if serviceProvider.service.dividendScreener == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		serviceProvider.service.dividendScreener = dividend.NewService(grpcClient.InstrumentsServiceClient())
	}
	return serviceProvider.service.dividendScreener
}
```

Добавить поле `dividendScreener dividend.Screener` в структуру `service` (файл, где она объявлена — искать `type service struct` в `service_provider`), импорт `"tinvest/internal/service/screener/dividend"`, и передать `s.GetDividendScreener()` в вызов `telegram_commands.New(...)`.

Примечание: `dividend.NewService` возвращает `*service`, реализующий и `Screener`, и `RankProvider`. Здесь используем как `Screener`; тот же экземпляр понадобится в Task 7 как `RankProvider` — вынести создание в отдельный геттер `GetDividendService() *dividend...` нельзя (unexported). Решение: добавить экспортируемый геттер, возвращающий конкретный тип через интерфейс-объединение — см. Task 7 Step 5, где вводится общий синглтон.

- [ ] **Step 8: Сборка + тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/service/screener/... ./internal/service/telegram_commands/...`
Expected: PASS

- [ ] **Step 9: Коммит**

```bash
git add internal/service/screener/dividend/ internal/service/telegram_commands/commands.go internal/service_provider/service.go
git commit -m "feat(screener): /dividend_screener bot command + telegram render"
```

---

## Task 7: Интеграция фунд-бонуса в Golden X Score

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/model/trade_result.go`
- Modify: `internal/service/trading_strategy/golden_x/model/pipeline_result.go`
- Modify: `internal/service/trading_strategy/golden_x/classifier/classifier.go`
- Modify: `internal/service/trading_strategy/golden_x/classifier/cap_sectors_test.go` (или новый `classifier_test.go`)
- Modify: `internal/service/trading_strategy/golden_x/detector/detect_all.go`
- Modify: `internal/service/trading_strategy/golden_x/types.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go`
- Modify: `internal/service_provider/service.go`

**Interfaces:**
- Consumes: `dividend.RankProvider` (метод `RankBonus(instrumentID string) int`).
- Produces: `signalScore` включает `sr.FundamentalBonus`; `DetectResult.FundamentalBonus`; `golden_x.WithRankProvider(p RankProvider) Option`.

- [ ] **Step 1: Тест signalScore с бонусом**

Создать `internal/service/trading_strategy/golden_x/classifier/classifier_test.go`:

```go
package classifier

import (
	"testing"

	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
)

func TestSignalScore_AddsFundamentalBonus(t *testing.T) {
	base := gxmodel.ShareResult{BuyTier: gxmodel.TierGreen, TrendStatus: gxmodel.TrendWith}
	withBonus := base
	withBonus.FundamentalBonus = 3

	if signalScore(withBonus)-signalScore(base) != 3 {
		t.Fatalf("bonus delta = %d, want 3", signalScore(withBonus)-signalScore(base))
	}
}
```

- [ ] **Step 2: Запустить — падает**

Run: `go test ./internal/service/trading_strategy/golden_x/classifier/ -run TestSignalScore_AddsFundamentalBonus`
Expected: FAIL — `FundamentalBonus undefined`.

- [ ] **Step 3: Добавить поля и учесть в score**

1. В `model/trade_result.go` в `ShareResult` добавить: `FundamentalBonus int`.
2. В `model/pipeline_result.go` в `DetectResult` добавить: `FundamentalBonus int`.
3. В `classifier/classifier.go`:
   - в конце `signalScore` перед `return s` добавить: `s += sr.FundamentalBonus`.
   - в `Classify`, при сборке `sr`, добавить: `sr.FundamentalBonus = dr.FundamentalBonus` (перед `sr.Score = signalScore(sr)`).

- [ ] **Step 4: Тест зелёный**

Run: `go test ./internal/service/trading_strategy/golden_x/classifier/ -v`
Expected: PASS (включая существующие).

- [ ] **Step 5: Прокинуть бонус через detect_all и trade**

1. В `detector/detect_all.go` изменить сигнатуру:
```go
func DetectAll(ctx context.Context, fetched []gxmodel.FetchResult, in dto.Trade, settings gxmodel.Settings, bonusFor func(instrumentID string) int) []gxmodel.DetectResult {
```
и при добавлении результата:
```go
		results = append(results, gxmodel.DetectResult{Share: fr.Share, Signal: sig, FundamentalBonus: bonusFor(fr.Share.ID)})
```

2. В `golden_x/types.go`:
   - импорт `"tinvest/internal/service/screener/dividend"`.
   - в `service` добавить поле `rankProvider dividend.RankProvider`.
   - Option:
```go
func WithRankProvider(p dividend.RankProvider) Option {
	return func(svc *service) { svc.rankProvider = p }
}
```
   - в `NewService` после установки полей задать no-op дефолт, если не передан:
```go
	if svc.rankProvider == nil {
		svc.rankProvider = noopRankProvider{}
	}
```
   - добавить тип:
```go
type noopRankProvider struct{}

func (noopRankProvider) RankBonus(string) int { return 0 }
```

3. В `golden_x/trade.go`, в `Trade`, заменить вызов `DetectAll`:
```go
	signals := detector.DetectAll(ctx, fetched, in, s.settings, s.rankProvider.RankBonus)
```

- [ ] **Step 6: Провязать один синглтон dividend-сервиса на оба потребителя**

В `internal/service_provider/service.go`:

1. Хранить конкретный экземпляр: поле `dividendService *dividend...` нельзя (unexported тип). Решение: хранить как `dividend.Screener` **и** отдать его же как `RankProvider` через отдельный приватный синглтон-геттер, возвращающий значение, реализующее оба интерфейса. Ввести экспортируемый метод на конструкторе: в пакете `dividend` добавить, что `NewService` возвращает тип, присваиваемый обоим интерфейсам — в провайдере хранить `dividendScreener dividend.Screener`, и рядом хранить `dividendRank dividend.RankProvider`, инициализируя ОБА одним вызовом:

```go
func (s *ServiceProvider) dividendSvc() *dividendSingleton {
	if serviceProvider.service.dividendSingleton == nil {
		grpcClient, _ := serviceProvider.GetGrpcClient()
		svc := dividend.NewService(grpcClient.InstrumentsServiceClient())
		serviceProvider.service.dividendSingleton = &dividendSingleton{screener: svc, provider: svc}
	}
	return serviceProvider.service.dividendSingleton
}
```
где
```go
type dividendSingleton struct {
	screener dividend.Screener
	provider dividend.RankProvider
}
```
Обновить `GetDividendScreener` (Task 6 Step 7) на `return s.dividendSvc().screener`.

2. В `GetGoldenXTradingService` передать провайдера:
```go
		serviceProvider.service.goldenXTradingService = golden_x.NewService(
			grpcClient.MarketDataServiceClient(),
			tgClient,
			golden_x.WithRankProvider(serviceProvider.dividendSvc().provider),
		)
```

- [ ] **Step 7: Сборка + весь гейт**

Run: `go build ./internal/... ./pkg/... ./cmd/... && ./bin/mage ci`
Expected: PASS (lint + race-тесты + mock-drift).

- [ ] **Step 8: Обновить доки и легенду**

1. В `notification/notifications.go` `legendBlock` (Golden X) добавить строку про фунд-бонус, например: `"🏆 фунд-рейтинг (+1..+3 к Score)\n"`. В строке покупки, где печатается `Score`, при `sr.FundamentalBonus > 0` можно добавить суффикс `🏆`.
2. В `docs/golden_x/strategy.md` (Шаг «Score»/секция про score) и `docs/golden_x/settings.md` описать новый компонент: диапазон Score стал 1..11, +0..3 от перцентильного ранга дивидендного скринера, обновляется раз в 24ч, no-op если провайдер не подключён.

- [ ] **Step 9: Коммит**

```bash
git add internal/service/trading_strategy/golden_x/ internal/service_provider/service.go docs/golden_x/
git commit -m "feat(golden_x): fold dividend fundamental rank into buy Score"
```

---

## Task 8: Валидация на живых данных + калибровка

**Files:**
- Modify (по итогам): `internal/service/screener/dividend/rank/config.go` (константы), `rank/rank.go` (только если единицы полей иные).

**Цель:** прогнать `/dividend_screener` на реальном токене, сверить единицы полей и вменяемость топа.

- [ ] **Step 1: Запустить приложение в dev**

Run: `go run ./cmd/main` (с заполненным `env/local.env`), затем в Telegram вызвать `/dividend_screener`.
Expected: приходит сообщение с топом и статистикой отсева.

- [ ] **Step 2: Сверить единицы и отсев**

Проверить в выводе/логах:
- yield и payout — проценты (напр. 8.5 / 55), а не доли (0.085 / 0.55). Если доли — скорректировать `YieldTrapMinYield`, `YieldCapPct`, `PayoutIdealLow/High`, `MaxPayoutPct` (÷100) в `DefaultConfig()`.
- `Stats.Gated` не съедает почти всю вселенную (урок bonds-скринера: гейт не должен резать легитимные имена). Если отсев чрезмерен — ослабить пороги/уточнить `reasonMissingData`.
- Топ-10 выглядит осмысленно для дивидендного инвестора (нет очевидного мусора вверху, нет очевидной голубой фишки в отсеве без причины).

- [ ] **Step 3: Зафиксировать калибровку**

Если правил константы — коммит:
```bash
git add internal/service/screener/dividend/rank/config.go
git commit -m "chore(screener): calibrate dividend ranking thresholds on live data"
```
Если правок нет — задача закрыта записью в память/PR-описании, что единицы подтверждены.

- [ ] **Step 4: Финальный гейт**

Run: `./bin/mage ci`
Expected: PASS

---

## Self-Review (выполнено при написании)

- **Покрытие спеки:** данные-фундамент (T1–T2) · ядро/6 столпов/gate/trap (T4) · перцентили (T3) · оркестрация+кэш+TTL+Top+RankBonus (T5) · команда+нотификация+видимый отсев (T6) · интеграция в Score с сохранением чистоты Classify/Detect (T7) · валидация единиц и калибровка (T8). Все разделы спеки покрыты.
- **Плейсхолдеры:** нет TBD/«обработать ошибки» — код приведён. Единственная явная неопределённость (единицы полей) вынесена в именованные константы + Task 8, что является дизайн-решением, а не плейсхолдером.
- **Согласованность типов:** `RankBonus(instrumentID string) int`, `Top(ctx,n)→([]RankedShare,Stats,error)`, `rank.Rank([]*model.Fundamentals,Config)→[]ScoredCompany`, `Render([]dividend.RankedShare,dividend.Stats)→string`, `DetectAll(...,bonusFor func(string)int)` — имена и сигнатуры совпадают между задачами.
- **Риск service_provider (T6 Step 7 / T7 Step 6):** unexported `*service` не хранится напрямую; введён `dividendSingleton`, отдающий один экземпляр как `Screener` и `RankProvider`. Оба потребителя используют один кэш.
