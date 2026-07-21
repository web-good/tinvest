# Секторные поправки скоринга + секторный вид отчёта — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Для финансового сектора считать место с учётом неприменимых метрик (нейтрализовать FCF в Устойчивости, Safety=0.5, Оценку по P/B), не меняя поведение небанков, и группировать топ-N отчёта бота по секторам.

**Architecture:** Сектор приходит из `Shares` (`Share.Sector`), маппится в `model.Share.Sector`. Пакет `rank` получает `sectorByAsset map[string]SectorKind` в `Rank`, применяет секторные ветки в трёх пиллярах. `service.refresh` строит карту секторов; `render` группирует по `Share.Sector`. Диагностический `cmd/divscreen` расширяется для калибровки на живом API.

**Tech Stack:** Go 1.25, существующие пакеты `rank`, `dividend`, конвертеры, `cmd/divscreen`.

## Global Constraints

- **v1 трогает только `SectorFinancial`.** Все не-финансовые имена считаются байт-в-байт как сейчас — регрессионные тесты небанков не должны измениться.
- **Сектор — метаданные инструмента, не фундаментал.** В `model.Fundamentals` поле сектора НЕ добавляется; сектор передаётся отдельной картой `sectorByAsset` (ключ — `AssetUID`); отсутствующий ключ → `SectorOther`.
- **proto3 omitempty:** 0 в фунд-полях = «нет данных». Для банков `PriceToBookTtm <= 0` → нейтральные 0.5; Safety банка = 0.5 всегда.
- **Единицы:** yield/payout — проценты; P/B — отношение (1.0 = цена = балансу); MarketCapitalization — рубли.
- **Экспорт из `rank`:** `SectorKind`, `SectorOther`, `SectorFinancial`, `ClassifySector`. Сигнатура `Rank(universe []*model.Fundamentals, sectorByAsset map[string]SectorKind, cfg Config) []ScoredCompany`.
- **Классификатор и ярлыки секторов, полоса P/B — точки калибровки:** финальные строки-коды и пороги фиксируются на живом прогоне `divscreen` (Task 6). До калибровки используются документированные ориентиры (`"financial"`; P/B 1.0/2.5).
- Проверка качества: `./bin/mage ci` (lint + `go test -race ./...` + mock-drift).

---

### Task 1: Поле `Sector` в `model.Share` + конвертер

**Files:**
- Modify: `internal/model/share.go`
- Modify: `internal/converter/share.go:32-45` (`ConvertShareFromPb`)
- Test: `internal/converter/share_test.go` (создать, если нет)

**Interfaces:**
- Produces: `model.Share.Sector string` — код сектора инструмента из Invest API.

- [ ] **Step 1: Добавить поле в модель**

В `internal/model/share.go` в структуру `Share` добавить поле после `DivYieldFlag`:

```go
	DivYieldFlag            bool
	Sector                  string
```

- [ ] **Step 2: Замапить в конвертере**

В `internal/converter/share.go` в `ConvertShareFromPb` добавить строку:

```go
func ConvertShareFromPb(share *investapi.Share) *model.Share {
	return &model.Share{
		Figi:         share.Figi,
		Ticker:       share.Ticker,
		Isin:         share.Isin,
		Lot:          share.Lot,
		Currency:     share.Currency,
		Name:         share.Name,
		ID:           share.Uid,
		Trading:      share.ApiTradeAvailableFlag,
		AssetUID:     share.AssetUid,
		DivYieldFlag: share.DivYieldFlag,
		Sector:       share.Sector,
	}
}
```

- [ ] **Step 3: Тест конвертера**

Создать `internal/converter/share_test.go` (если файла нет — иначе добавить кейс):

```go
package converter

import (
	"testing"

	investapi "tinvest/internal/pb/v1"
)

func TestConvertShareFromPb_MapsSector(t *testing.T) {
	got := ConvertShareFromPb(&investapi.Share{
		Ticker:   "SBER",
		Currency: "rub",
		Sector:   "financial",
	})
	if got.Sector != "financial" {
		t.Fatalf("Sector = %q, want %q", got.Sector, "financial")
	}
}
```

- [ ] **Step 4: Прогнать тест**

Run: `go test ./internal/converter/ -run TestConvertShareFromPb_MapsSector -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/share.go internal/converter/share.go internal/converter/share_test.go
git commit -m "feat(screener): map Share.Sector from Invest API"
```

---

### Task 2: Классификатор сектора в `rank`

**Files:**
- Create: `internal/service/screener/dividend/rank/sector.go`
- Test: `internal/service/screener/dividend/rank/sector_test.go`

**Interfaces:**
- Produces:
  - `type SectorKind int` со значениями `SectorOther = iota`, `SectorFinancial`.
  - `func ClassifySector(sector string) SectorKind` — регистронезависимо относит финансовые коды к `SectorFinancial`, остальное (в т.ч. пустая строка) — `SectorOther`.

- [ ] **Step 1: Написать падающий тест**

Создать `internal/service/screener/dividend/rank/sector_test.go`:

```go
package rank

import "testing"

func TestClassifySector(t *testing.T) {
	cases := []struct {
		in   string
		want SectorKind
	}{
		{"financial", SectorFinancial},
		{"Financial", SectorFinancial},
		{"FINANCIAL", SectorFinancial},
		{"energy", SectorOther},
		{"", SectorOther},
		{"unknown_code", SectorOther},
	}
	for _, c := range cases {
		if got := ClassifySector(c.in); got != c.want {
			t.Errorf("ClassifySector(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Прогнать — упадёт компиляцией**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestClassifySector`
Expected: FAIL (undefined: SectorKind/ClassifySector)

- [ ] **Step 3: Реализовать классификатор**

Создать `internal/service/screener/dividend/rank/sector.go`:

```go
package rank

import "strings"

// SectorKind — грубая классификация сектора инструмента для секторных поправок
// скоринга. v1 различает только финансовый сектор (банки/финансы), где часть
// фунд-метрик неприменима (нет EBITDA/FCF), от всех остальных.
type SectorKind int

const (
	SectorOther SectorKind = iota
	SectorFinancial
)

// financialSectorCodes — коды сектора Invest API, относимые к финансам.
// Набор откалиброван на живом API (см. cmd/divscreen). Сравнение
// регистронезависимое.
var financialSectorCodes = map[string]struct{}{
	"financial": {},
}

// ClassifySector относит строковый код сектора Invest API к SectorKind.
// Неизвестный или пустой код → SectorOther.
func ClassifySector(sector string) SectorKind {
	if _, ok := financialSectorCodes[strings.ToLower(strings.TrimSpace(sector))]; ok {
		return SectorFinancial
	}
	return SectorOther
}
```

- [ ] **Step 4: Прогнать тест**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestClassifySector -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/screener/dividend/rank/sector.go internal/service/screener/dividend/rank/sector_test.go
git commit -m "feat(screener): sector classifier (financial vs other) in rank"
```

---

### Task 3: Полоса P/B для банков + поля Config

**Files:**
- Modify: `internal/service/screener/dividend/rank/config.go`
- Modify: `internal/service/screener/dividend/rank/rank.go` (добавить `bankValuation`)
- Test: `internal/service/screener/dividend/rank/rank_test.go`

**Interfaces:**
- Consumes: `Config`, `clamp01` (уже есть в `rank.go`).
- Produces:
  - `Config.BankPBIdealHigh float64`, `Config.BankPBZero float64`.
  - `func bankValuation(pb float64, cfg Config) float64` — абсолютная полоса оценки банка по P/B.

- [ ] **Step 1: Добавить поля Config**

В `internal/service/screener/dividend/rank/config.go` в структуру `Config` после `PayoutIdealHigh`:

```go
	PayoutIdealLow  float64 // нижняя граница идеальной зоны payout
	PayoutIdealHigh float64 // верхняя граница идеальной зоны payout

	BankPBIdealHigh float64 // P/B, до которого оценка банка максимальна (1.0)
	BankPBZero      float64 // P/B, при котором оценка банка падает до 0
```

В `DefaultConfig()` после `PayoutIdealHigh: 60.0,`:

```go
		PayoutIdealLow:  30.0,
		PayoutIdealHigh: 60.0,

		BankPBIdealHigh: 1.0, // ориентир; live-калибровка в Task 6
		BankPBZero:      2.5, // ориентир; live-калибровка в Task 6
```

- [ ] **Step 2: Написать падающий тест**

В `internal/service/screener/dividend/rank/rank_test.go` добавить:

```go
func TestBankValuation(t *testing.T) {
	cfg := DefaultConfig() // IdealHigh=1.0, Zero=2.5
	cases := []struct {
		name string
		pb   float64
		want float64
	}{
		{"no data → neutral", 0, 0.5},
		{"negative → neutral", -1, 0.5},
		{"cheap at ideal high → max", 1.0, 1.0},
		{"below ideal high → max", 0.6, 1.0},
		{"at zero point → 0", 2.5, 0.0},
		{"above zero point → 0", 3.0, 0.0},
		{"midpoint → 0.5", 1.75, 0.5}, // (2.5-1.75)/(2.5-1.0)=0.5
	}
	for _, c := range cases {
		if got := bankValuation(c.pb, cfg); got != c.want {
			t.Errorf("%s: bankValuation(%v) = %v, want %v", c.name, c.pb, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Прогнать — упадёт**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestBankValuation`
Expected: FAIL (undefined: bankValuation)

- [ ] **Step 4: Реализовать `bankValuation`**

В `internal/service/screener/dividend/rank/rank.go` после `leverageScore` добавить:

```go
// bankValuation: оценка банка по P/B (у банков нет EBITDA, EV/EBITDA неприменим).
// P/B <= BankPBIdealHigh → 1.0 (дёшево); линейно к 0 при P/B >= BankPBZero;
// P/B <= 0 (нет данных, proto3 omitempty) → нейтральные 0.5.
func bankValuation(pb float64, cfg Config) float64 {
	if pb <= 0 {
		return 0.5
	}
	if pb <= cfg.BankPBIdealHigh {
		return 1.0
	}
	span := cfg.BankPBZero - cfg.BankPBIdealHigh
	if span <= 0 {
		return 0
	}
	return clamp01((cfg.BankPBZero - pb) / span)
}
```

- [ ] **Step 5: Прогнать тест**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestBankValuation -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/screener/dividend/rank/config.go internal/service/screener/dividend/rank/rank.go internal/service/screener/dividend/rank/rank_test.go
git commit -m "feat(screener): absolute P/B valuation band for banks"
```

---

### Task 4: Секторные ветки в `Rank` (три поправки)

**Files:**
- Modify: `internal/service/screener/dividend/rank/rank.go` (сигнатура `Rank`, ветки пилляров)
- Modify: `internal/service/screener/dividend/rank/rank_test.go`
- Modify: все вызовы `rank.Rank` (обновляются в Task 5/7; здесь — только ядро и его тесты)

**Interfaces:**
- Consumes: `ClassifySector`/`SectorKind` (Task 2), `bankValuation` (Task 3), `sustainabilityPayout`, `leverageScore`, `percentileOrNeutral` (есть).
- Produces: `func Rank(universe []*model.Fundamentals, sectorByAsset map[string]SectorKind, cfg Config) []ScoredCompany` — банки получают Sustainability без FCF, Safety=0.5, Valuation по P/B; небанки — без изменений.

- [ ] **Step 1: Написать падающие тесты (секторные ветки)**

В `internal/service/screener/dividend/rank/rank_test.go` добавить. Хелпер существующих тестов использует фикстуры фундаменталов — свериться с текущими (payout/FCF/ND/PB/yield/marketcap/freefloat выше порогов ворот). Значения ниже подобраны, чтобы проходить ворота (`DefaultConfig`): MarketCap ≥ 50e9, FreeFloat ≥ 0.07, yield 0<..<20.

```go
func TestRank_FinancialSustainabilityIgnoresFCF(t *testing.T) {
	cfg := DefaultConfig()
	base := func(fcf float64) *model.Fundamentals {
		return &model.Fundamentals{
			AssetUID:                   "bank",
			ForwardAnnualDividendYield: 10,
			DividendPayoutRatioFy:      50, // идеальная зона → payoutFit 1.0
			FreeCashFlowTtm:            fcf,
			MarketCapitalization:       100e9,
			FreeFloat:                  0.3,
			PriceToBookTtm:             1.0,
		}
	}
	sec := map[string]SectorKind{"bank": SectorFinancial}

	withZeroFCF := Rank([]*model.Fundamentals{base(0)}, sec, cfg)
	withPosFCF := Rank([]*model.Fundamentals{base(100)}, sec, cfg)

	if withZeroFCF[0].Sustainability != withPosFCF[0].Sustainability {
		t.Fatalf("bank Sustainability must ignore FCF: fcf0=%v fcfPos=%v",
			withZeroFCF[0].Sustainability, withPosFCF[0].Sustainability)
	}
	// payout 50 в идеальной зоне → sustainabilityPayout = 1.0
	if withZeroFCF[0].Sustainability != 1.0 {
		t.Fatalf("bank Sustainability = %v, want 1.0 (payoutFit only)", withZeroFCF[0].Sustainability)
	}
}

func TestRank_FinancialSafetyNeutral(t *testing.T) {
	cfg := DefaultConfig()
	f := &model.Fundamentals{
		AssetUID:                   "bank",
		ForwardAnnualDividendYield: 10,
		DividendPayoutRatioFy:      50,
		NetDebtToEbitda:            5, // был бы leverageScore 0.15... но для банка игнор
		MarketCapitalization:       100e9,
		FreeFloat:                  0.3,
		PriceToBookTtm:             1.0,
	}
	got := Rank([]*model.Fundamentals{f}, map[string]SectorKind{"bank": SectorFinancial}, cfg)
	if got[0].Safety != 0.5 {
		t.Fatalf("bank Safety = %v, want 0.5", got[0].Safety)
	}
}

func TestRank_FinancialValuationByPB(t *testing.T) {
	cfg := DefaultConfig()
	f := &model.Fundamentals{
		AssetUID:                   "bank",
		ForwardAnnualDividendYield: 10,
		DividendPayoutRatioFy:      50,
		EvToEbitdaMrq:              0,   // неприменимо; не должно давать 0.98
		PriceToBookTtm:             1.0, // ≤ IdealHigh → 1.0
		MarketCapitalization:       100e9,
		FreeFloat:                  0.3,
	}
	got := Rank([]*model.Fundamentals{f}, map[string]SectorKind{"bank": SectorFinancial}, cfg)
	if got[0].Valuation != 1.0 {
		t.Fatalf("bank Valuation = %v, want 1.0 (P/B band)", got[0].Valuation)
	}
}

func TestRank_NonFinancialUnchanged(t *testing.T) {
	cfg := DefaultConfig()
	f := &model.Fundamentals{
		AssetUID:                   "oil",
		ForwardAnnualDividendYield: 10,
		DividendPayoutRatioFy:      50,
		FreeCashFlowTtm:            100, // FCF>0 → +0.3
		NetDebtToEbitda:            -0.1, // чистый кэш → leverageScore 1.0
		MarketCapitalization:       100e9,
		FreeFloat:                  0.3,
	}
	got := Rank([]*model.Fundamentals{f}, map[string]SectorKind{"oil": SectorOther}, cfg)
	// Sustainability = 0.7*1.0 + 0.3*1.0 = 1.0 (старая формула)
	if got[0].Sustainability != 1.0 {
		t.Fatalf("non-bank Sustainability = %v, want 1.0 (0.7*payout + 0.3*FCF)", got[0].Sustainability)
	}
	// Safety = leverageScore(-0.1) = 1.0
	if got[0].Safety != 1.0 {
		t.Fatalf("non-bank Safety = %v, want 1.0", got[0].Safety)
	}
}
```

- [ ] **Step 2: Прогнать — упадёт (сигнатура Rank без sectorByAsset)**

Run: `go test ./internal/service/screener/dividend/rank/ -run TestRank_`
Expected: FAIL (компиляция: Rank принимает 2 аргумента)

- [ ] **Step 3: Изменить сигнатуру и ветки `Rank`**

В `internal/service/screener/dividend/rank/rank.go` заменить сигнатуру и тело цикла подсчёта пилляров. Новая сигнатура:

```go
func Rank(universe []*model.Fundamentals, sectorByAsset map[string]SectorKind, cfg Config) []ScoredCompany {
```

В цикле по `survivors` (сейчас строки 157-172) заменить вычисление трёх пилляров на секторные ветки:

```go
	scored := make([]ScoredCompany, 0, len(survivors))
	for _, f := range survivors {
		sc := ScoredCompany{AssetUID: f.AssetUID}
		kind := sectorByAsset[f.AssetUID] // отсутствующий ключ → SectorOther (нулевое значение)

		switch kind {
		case SectorFinancial:
			// У банков FCF/EBITDA/leverage неприменимы (дыры данных / бессмысленны).
			sc.Sustainability = sustainabilityPayout(f.DividendPayoutRatioFy, cfg)
			sc.Safety = 0.5
			sc.Valuation = bankValuation(f.PriceToBookTtm, cfg)
		default:
			sc.Sustainability = 0.7*sustainabilityPayout(f.DividendPayoutRatioFy, cfg) + 0.3*boolScore(f.FreeCashFlowTtm > 0)
			sc.Safety = leverageScore(f.NetDebtToEbitda)
			sc.Valuation = 1 - percentileOrNeutral(evEbitda, f.EvToEbitdaMrq) // ниже EV/EBITDA — лучше
		}

		sc.DivGrowth = percentileOrNeutral(divGrowth, f.FiveYearAnnualDividendGrowthRate)
		sc.Quality = percentileOrNeutral(roic, qualityMetric(f))
		sc.YieldScore = clamp01(minf(yieldOf(f), cfg.YieldCapPct) / cfg.YieldCapPct)

		sc.Composite = 100 * (cfg.WeightSustainability*sc.Sustainability +
			cfg.WeightSafety*sc.Safety +
			cfg.WeightDivGrowth*sc.DivGrowth +
			cfg.WeightQuality*sc.Quality +
			cfg.WeightValuation*sc.Valuation +
			cfg.WeightYield*sc.YieldScore)
		scored = append(scored, sc)
	}
```

Примечание: пул `evEbitda` по-прежнему строится по всем выжившим (Task не меняет его построение); банки в него попадают, но свою Valuation читают из `bankValuation`. Небанки ранжируются перцентилем EV/EBITDA как раньше.

- [ ] **Step 4: Прогнать новые + существующие тесты пакета**

Run: `go test ./internal/service/screener/dividend/rank/ -v`
Expected: FAIL на существующих кейсах, вызывающих `Rank(funds, cfg)` со старой сигнатурой (2 аргумента). Обновить их вызовы на `Rank(funds, nil, cfg)` (nil-карта → все `SectorOther`, поведение небанков не меняется) — существующие ожидания должны сохраниться, т.к. небанковская ветка идентична прежней формуле. Прогнать снова:

Run: `go test ./internal/service/screener/dividend/rank/ -v`
Expected: PASS (все, включая новые секторные)

- [ ] **Step 5: Commit**

```bash
git add internal/service/screener/dividend/rank/rank.go internal/service/screener/dividend/rank/rank_test.go
git commit -m "feat(screener): sector-aware pillars in Rank (financial branch)"
```

---

### Task 5: Проброс сектора в `service.refresh`

**Files:**
- Modify: `internal/service/screener/dividend/service.go:48-108` (`refresh`)
- Test: `internal/service/screener/dividend/service_test.go`

**Interfaces:**
- Consumes: `rank.ClassifySector`, `rank.SectorKind`, `model.Share.Sector`, новая сигнатура `rank.Rank`.
- Produces: `refresh` строит `sectorByAsset` и передаёт в `rank.Rank`.

- [ ] **Step 1: Написать падающий тест (сектор влияет на композит через сервис)**

В `internal/service/screener/dividend/service_test.go` добавить кейс: два актива с ОДИНАКОВЫМИ фундаменталами, но один помечен сектором `financial` (через `Share.Sector`), другой — нет; композиты различаются (у банка Safety=0.5 вместо leverageScore). Использовать существующий мок `MockinstrumentsClient` и его паттерн из соседних тестов. Пример (адаптировать под существующие хелперы файла):

```go
func TestRefresh_SectorAffectsComposite(t *testing.T) {
	bankUID, oilUID := "bank-asset", "oil-asset"
	shares := []*model.Share{
		{ID: "SBER", Ticker: "SBER", Name: "Bank", AssetUID: bankUID, DivYieldFlag: true, Sector: "financial"},
		{ID: "LKOH", Ticker: "LKOH", Name: "Oil", AssetUID: oilUID, DivYieldFlag: true, Sector: "energy"},
	}
	fund := func(uid string) *model.Fundamentals {
		return &model.Fundamentals{
			AssetUID:                   uid,
			ForwardAnnualDividendYield: 10,
			DividendPayoutRatioFy:      50,
			NetDebtToEbitda:            -0.1, // небанк: leverageScore 1.0; банк игнор → 0.5
			FreeCashFlowTtm:            100,
			PriceToBookTtm:             1.0,
			MarketCapitalization:       100e9,
			FreeFloat:                  0.3,
		}
	}
	funds := []*model.Fundamentals{fund(bankUID), fund(oilUID)}

	client := NewMockinstrumentsClient(t)
	client.EXPECT().Shares(mock.Anything).Return(shares, nil)
	client.EXPECT().GetAssetFundamentals(mock.Anything, mock.Anything).Return(funds, nil)

	s := NewService(client)
	top, _, err := s.Top(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	byTicker := map[string]float64{}
	for _, rs := range top {
		byTicker[rs.Share.Ticker] = rs.Scored.Composite
	}
	if byTicker["SBER"] == byTicker["LKOH"] {
		t.Fatalf("bank and non-bank with equal fundamentals must differ: SBER=%v LKOH=%v",
			byTicker["SBER"], byTicker["LKOH"])
	}
}
```

- [ ] **Step 2: Прогнать — упадёт**

Run: `go test ./internal/service/screener/dividend/ -run TestRefresh_SectorAffectsComposite`
Expected: FAIL (сигнатура `rank.Rank` / сектор не прокинут)

- [ ] **Step 3: Прокинуть сектор в `refresh`**

В `internal/service/screener/dividend/service.go` в `refresh`, после построения `sharesByAsset` и перед `rank.Rank`, собрать карту секторов и передать её:

```go
	sectorByAsset := make(map[string]rank.SectorKind, len(sharesByAsset))
	for assetUID, instruments := range sharesByAsset {
		if len(instruments) > 0 {
			sectorByAsset[assetUID] = rank.ClassifySector(instruments[0].Sector)
		}
	}

	funds, err := s.client.GetAssetFundamentals(ctx, uids)
	if err != nil {
		return fmt.Errorf("dividend screener: fetch fundamentals: %w", err)
	}

	scored := rank.Rank(funds, sectorByAsset, s.cfg)
```

- [ ] **Step 4: Прогнать тесты пакета**

Run: `go test ./internal/service/screener/dividend/ -v`
Expected: PASS (обновить любые прочие вызовы `rank.Rank` в тестах пакета на новую сигнатуру, передавая `nil` или карту)

- [ ] **Step 5: Commit**

```bash
git add internal/service/screener/dividend/service.go internal/service/screener/dividend/service_test.go
git commit -m "feat(screener): thread instrument sector into Rank via refresh"
```

---

### Task 6: Диагностический CLI — колонки сектора/P/B и калибровка на живом API

**Files:**
- Modify: `cmd/divscreen/main.go`

**Interfaces:**
- Consumes: `rank.ClassifySector`, `rank.SectorKind`, новая сигнатура `rank.Rank`, `model.Share.Sector`.

- [ ] **Step 1: Прокинуть сектор в вызов Rank и probe**

В `cmd/divscreen/main.go`:
- В `run`, при построении `byUID`, сохранять сектор шары: добавить поле `sector string` в структуру `asset` и заполнять `sector: sh.Sector` при первом инструменте актива.
- Построить `sectorByAsset := map[string]rank.SectorKind{}` из `byUID` (`rank.ClassifySector(a.sector)`), передать в `rank.Rank(funds, sectorByAsset, cfg)`.
- В `printRanked`: добавить в шапку и строки колонки `Sector` (`a.sector`) и `P/B` (`f.PriceToBookTtm`).
- В `printProbe`: печатать класс сектора (`financial`/`other`) и P/B, и (для банка) отметку, что применены секторные поправки.

Полный diff оставлен исполнителю (файл целиком в контексте); ключевое — сигнатура `rank.Rank` теперь трёхаргументная, класс сектора виден в выводе.

- [ ] **Step 2: Собрать CLI**

Run: `go build ./cmd/divscreen`
Expected: успешная сборка

- [ ] **Step 3: Живой прогон — перечислить коды секторов**

Run: `go run ./cmd/divscreen -top 0`
Ожидание: в колонке `Sector` видны реальные строки-коды. Зафиксировать множество кодов; определить, какие относятся к банкам/финансам (Сбер, ВТБ, Банк СПб, Совкомбанк, МОЕX, TCS и т.п.).

- [ ] **Step 4: Обновить классификатор и ярлыки по фактам**

Если реальный код финансов не `"financial"` (или их несколько — напр. отдельно биржи/страхование), обновить `financialSectorCodes` в `internal/service/screener/dividend/rank/sector.go`. Зафиксировать полный набор кодов для карты ярлыков (Task 7).

- [ ] **Step 5: Живой probe банков — калибровка P/B**

Run: `go run ./cmd/divscreen -probe SBER,BSPB,SVCB,VTBR,MOEX`
Ожидание: для банков видно P/B, класс `financial`, Safety=0.5, Valuation по P/B. По распределению живых P/B подтвердить/подправить `BankPBIdealHigh`/`BankPBZero` в `DefaultConfig()` (ориентиры 1.0/2.5). Задокументировать финал в комментарии рядом с полями.

- [ ] **Step 6: Регрессия curated-11 и SBER/BSPB/TATN**

Run: `go run ./cmd/divscreen -top 0`
Ожидание: 8 небанков curated — композит не изменился; SBER сместился вниз (снятие ложных Safety/Valuation), BSPB выше SBER по payout, TATN не задет (не банк). Зафиксировать наблюдения для докстроки Task 8.

- [ ] **Step 7: Commit**

```bash
git add cmd/divscreen/main.go internal/service/screener/dividend/rank/config.go internal/service/screener/dividend/rank/sector.go
git commit -m "chore(screener): divscreen sector/PB columns; calibrate financial codes and P/B band on live data"
```

---

### Task 7: Секторная группировка в `render`

**Files:**
- Modify: `internal/service/screener/dividend/render.go`
- Test: `internal/service/screener/dividend/render_test.go`

**Interfaces:**
- Consumes: `RankedShare.Share.Sector`.
- Produces: `Render` группирует топ-N по секторам с подзаголовком-агрегатом; `sectorLabel(code string) string` — эмодзи+RU-ярлык, фолбэк «Прочее».

- [ ] **Step 1: Написать падающий тест рендера**

В `internal/service/screener/dividend/render_test.go` добавить:

```go
func TestRender_GroupsBySector(t *testing.T) {
	ranked := []RankedShare{
		{Share: &model.Share{Ticker: "MSRS", Name: "Rosseti", Sector: "utilities"},
			Scored: rank.ScoredCompany{Composite: 74}},
		{Share: &model.Share{Ticker: "TATN", Name: "Tatneft", Sector: "energy"},
			Scored: rank.ScoredCompany{Composite: 73}},
		{Share: &model.Share{Ticker: "BSPB", Name: "BSPB", Sector: "financial"},
			Scored: rank.ScoredCompany{Composite: 72}},
		{Share: &model.Share{Ticker: "SBER", Name: "Sber", Sector: "financial"},
			Scored: rank.ScoredCompany{Composite: 64}},
	}
	out := Render(ranked, Stats{Universe: 100, Ranked: 4, Gated: 96})

	// Финансовый сектор представлен подзаголовком с числом имён и средним.
	if !strings.Contains(out, "2 имени") && !strings.Contains(out, "2 имён") {
		t.Errorf("ожидался агрегат по числу имён финансов, got:\n%s", out)
	}
	// Сектор с #1 (utilities, MSRS 74) идёт раньше финансов (лучший 72).
	if strings.Index(out, "MSRS") > strings.Index(out, "SBER") {
		t.Errorf("сектор с лучшим композитом должен идти первым:\n%s", out)
	}
	// Неизвестный код не появляется как «Прочее», раз все коды известны.
	// (проверка ярлыка отдельно ниже)
}

func TestSectorLabel_Fallback(t *testing.T) {
	if got := sectorLabel("no_such_code_xyz"); !strings.Contains(got, "Прочее") {
		t.Errorf("неизвестный код → «Прочее», got %q", got)
	}
}
```

(Импортировать `rank` и `model` в тест-файл, если ещё не импортированы.)

- [ ] **Step 2: Прогнать — упадёт**

Run: `go test ./internal/service/screener/dividend/ -run 'TestRender_GroupsBySector|TestSectorLabel'`
Expected: FAIL

- [ ] **Step 3: Реализовать группировку и ярлыки**

В `internal/service/screener/dividend/render.go` переписать тело `Render` так, чтобы:
1. Разбить `ranked` по `rs.Share.Sector` в бакеты, сохраняя порядок (composite desc уже отсортирован).
2. Упорядочить секторы по максимальному композиту в бакете (сектор с #1 первым).
3. Для каждого сектора вывести подзаголовок `<label> · N имён · ср. XX` (склонение «имя/имени/имён» — простой хелпер по числу), затем карточки компаний прежнего формата.
4. Добавить `sectorLabel(code string) string` с картой известных кодов (наполняется по факту Task 6 — как минимум `financial`, `energy`, `utilities`, `materials`, `telecom`, `consumer`, `it`, `health_care`, `industrials`, `real_estate`) и фолбэком `"📊 Прочее"`.

Реализация подзаголовка (эскиз; исполнитель дополняет карту кодов):

```go
func sectorLabel(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "financial":
		return "🏦 Финансы"
	case "energy":
		return "🛢 Нефтегаз"
	case "utilities":
		return "⚡ Энергетика"
	case "materials":
		return "⛏ Материалы"
	case "telecom":
		return "📡 Телеком"
	case "consumer":
		return "🛒 Потребительский"
	case "it":
		return "💻 IT"
	case "health_care":
		return "💊 Здравоохранение"
	case "industrials":
		return "🏭 Промышленность"
	case "real_estate":
		return "🏢 Недвижимость"
	default:
		return "📊 Прочее"
	}
}

func plural(n int, one, few, many string) string {
	nn := n % 100
	if nn >= 11 && nn <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}
```

Карточка компании — сохранить прежний формат (номер общего места, название/тикер, композит, строка пилляров). Номер места — глобальный (позиция в топ-N), чтобы порядок был читаем и после бакетинга.

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/screener/dividend/ -run 'TestRender|TestSectorLabel' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/screener/dividend/render.go internal/service/screener/dividend/render_test.go
git commit -m "feat(screener): group dividend report by sector with aggregate subheaders"
```

---

### Task 8: Документация

**Files:**
- Modify: `docs/dividend/screener.md`

- [ ] **Step 1: Описать секторные поправки**

В `docs/dividend/screener.md` добавить раздел «Секторные поправки (v1 — финансы)»:
- Таблица: для `financial` Sustainability без FCF, Safety=0.5, Valuation по P/B; небанки — без изменений.
- Правило `PriceToBookTtm <= 0 → нейтральные 0.5`; Safety банка = 0.5 (обоснование: нет capital adequacy).
- Полоса P/B (`BankPBIdealHigh`/`BankPBZero`) с финальными калиброванными значениями.
- Классификатор сектора: какие коды Invest API относятся к финансам (по факту Task 6).

- [ ] **Step 2: Описать секторный вид отчёта**

Добавить, что топ-N бота сгруппирован по секторам с подзаголовком `<label> · N имён · ср. XX`, порядок секторов — по лучшему композиту.

- [ ] **Step 3: Запись в историю калибровки**

Добавить запись 2026-07-21 (sector-aware): финальные коды финансов, полоса P/B, наблюдения по SBER/BSPB/TATN и целостности curated-11.

- [ ] **Step 4: Commit**

```bash
git add docs/dividend/screener.md
git commit -m "docs(screener): document sector-aware pillars and sectored report view"
```

---

### Task 9: Полный прогон CI

- [ ] **Step 1: Регенерация моков при необходимости**

Если менялись мокаемые интерфейсы (не ожидается — `instrumentsClient` не тронут), выполнить `./bin/mage mocks`. Иначе пропустить.

- [ ] **Step 2: Полный CI**

Run: `./bin/mage ci`
Expected: exit 0 (lint + `go test -race ./...` + mock-drift check зелёные)

- [ ] **Step 3: Финальная фиксация (если остались правки моков/линта)**

```bash
git add -A
git commit -m "chore(screener): mage ci green for sector-aware scoring"
```

(Если правок нет — шаг пропустить.)
