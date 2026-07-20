# Bonds Screener Quality Upgrade — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Поднять качество облигационного скринера — явная тестируемая политика надёжности и корректная YTM (через `indicators.XIRR`) вместо линейной аннуализации, сохранив структуру лесенки.

**Architecture:** Pipeline остаётся `Bonds() → Finder → CalculateProfit → Sender`. Фильтр надёжности переезжает из спрятанного хардкода в конвертере в явную политику `Filter` внутри `Finder`. Расчёт доходности заменяет линейную формулу на XIRR по датированным денежным потокам. Границы ступеней лесенки выносятся в явный конфиг. Вывод в Telegram дополняется YTM/сектором.

**Tech Stack:** Go 1.25, gRPC (Tinkoff Invest API), `pkg/indicators` (XIRR), table-driven тесты.

## Global Constraints

- Go 1.25; сборка проверяется через `go build ./internal/... ./pkg/... ./cmd/...` (пакет `magefiles` без `main` — не билдить).
- Тесты — table-driven, по стилю репозитория; запуск `go test ./...` (в CI с `-race`).
- Округление денег/процентов — через `utils.RoundFloat`.
- Комментарии и текст сообщений — на русском, как в существующем коде.
- Не трогать поведение других стратегий; `ConvertBondsFromPb`/`Bonds()` используются только облигационным скринером (проверяется grep-ом в Task 2).

---

### Task 1: Расширить модель `Bond` и маппинг новых proto-полей

**Files:**
- Modify: `pkg/client/grpc/model/bond.go`
- Modify: `pkg/client/grpc/converter/bond.go` (только `ConvertBondModelFromBondPb`)
- Test: `pkg/client/grpc/converter/bond_test.go` (создать, если отсутствует)

**Interfaces:**
- Produces: поля `pkgmodel.Bond`: `LiquidityFlag bool`, `SubordinatedFlag bool`, `ForQualInvestorFlag bool`, `PerpetualFlag bool`, `Sector string`, `IssueSize int64`. Существующий `RiskLevel string` сохраняется.

- [ ] **Step 1: Написать падающий тест маппинга новых полей**

Создать/дополнить `pkg/client/grpc/converter/bond_test.go`:

```go
package converter

import (
	"testing"
	investapi "tinvest/internal/pb/v1"
)

func TestConvertBondModelFromBondPb_NewFields(t *testing.T) {
	pb := &investapi.Bond{
		Uid:                 "uid-1",
		Name:                "ОФЗ 26238",
		Nominal:             &investapi.MoneyValue{Units: 1000},
		AciValue:            &investapi.MoneyValue{Units: 15, Nano: 500000000},
		RiskLevel:           investapi.RiskLevel_RISK_LEVEL_LOW,
		LiquidityFlag:       true,
		SubordinatedFlag:    true,
		ForQualInvestorFlag: true,
		PerpetualFlag:       true,
		Sector:              "government",
		IssueSize:           5000000,
	}

	got := ConvertBondModelFromBondPb(pb)

	if !got.LiquidityFlag || !got.SubordinatedFlag || !got.ForQualInvestorFlag || !got.PerpetualFlag {
		t.Fatalf("флаги не замаплены: %+v", got)
	}
	if got.Sector != "government" {
		t.Fatalf("Sector = %q, ожидалось government", got.Sector)
	}
	if got.IssueSize != 5000000 {
		t.Fatalf("IssueSize = %d, ожидалось 5000000", got.IssueSize)
	}
}
```

- [ ] **Step 2: Запустить тест — убедиться, что не компилируется/падает**

Run: `go test ./pkg/client/grpc/converter/ -run TestConvertBondModelFromBondPb_NewFields -v`
Expected: FAIL — компиляция падает (нет полей в модели).

- [ ] **Step 3: Добавить поля в модель**

В `pkg/client/grpc/model/bond.go`, внутрь `type Bond struct`, дописать после `RiskLevel`:

```go
	LiquidityFlag       bool
	SubordinatedFlag    bool
	ForQualInvestorFlag bool
	PerpetualFlag       bool
	Sector              string
	IssueSize           int64
```

- [ ] **Step 4: Замапить поля в конвертере**

В `pkg/client/grpc/converter/bond.go`, в `ConvertBondModelFromBondPb`, дописать в возвращаемый `&model.Bond{...}`:

```go
		LiquidityFlag:       bond.LiquidityFlag,
		SubordinatedFlag:    bond.SubordinatedFlag,
		ForQualInvestorFlag: bond.ForQualInvestorFlag,
		PerpetualFlag:       bond.PerpetualFlag,
		Sector:              bond.Sector,
		IssueSize:           bond.IssueSize,
```

- [ ] **Step 5: Запустить тест — убедиться, что проходит**

Run: `go test ./pkg/client/grpc/converter/ -run TestConvertBondModelFromBondPb_NewFields -v`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add pkg/client/grpc/model/bond.go pkg/client/grpc/converter/bond.go pkg/client/grpc/converter/bond_test.go
git commit -m "feat(bonds): map liquidity/subordinated/qual/perpetual/sector/issue-size fields"
```

---

### Task 2: Явная политика надёжности в `Finder` + «тупой» конвертер

**Files:**
- Create: `internal/service/trading_strategy/bonds/pipeline/filter.go`
- Create: `internal/service/trading_strategy/bonds/pipeline/filter_test.go`
- Modify: `internal/service/trading_strategy/bonds/pipeline/finder.go`
- Modify: `pkg/client/grpc/converter/bond.go` (`ConvertBondsFromPb`)

**Interfaces:**
- Consumes: поля `pkgmodel.Bond` из Task 1.
- Produces: `pipeline.PassesReliability(bond *pkgmodel.Bond) bool` — единая политика отбора надёжных бумаг.

- [ ] **Step 1: Проверить blast radius конвертера**

Run: `grep -rn "ConvertBondsFromPb" internal/ pkg/`
Expected: единственный вызов — в `pkg/client/grpc/instruments_service_client.go` (метод `Bonds`). Если найдётся другой потребитель — остановиться и сообщить: снятие фильтра затронет и его.

- [ ] **Step 2: Написать падающий тест политики надёжности**

Создать `internal/service/trading_strategy/bonds/pipeline/filter_test.go`:

```go
package pipeline

import (
	"testing"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func goodBond() *pkgmodel.Bond {
	return &pkgmodel.Bond{
		RiskLevel:           "RISK_LEVEL_LOW",
		LiquidityFlag:       true,
		SubordinatedFlag:    false,
		ForQualInvestorFlag: false,
		PerpetualFlag:       false,
	}
}

func TestPassesReliability(t *testing.T) {
	tests := []struct {
		name string
		mut  func(b *pkgmodel.Bond)
		want bool
	}{
		{"надёжная проходит", func(b *pkgmodel.Bond) {}, true},
		{"не LOW risk — отсев", func(b *pkgmodel.Bond) { b.RiskLevel = "RISK_LEVEL_MODERATE" }, false},
		{"неликвид — отсев", func(b *pkgmodel.Bond) { b.LiquidityFlag = false }, false},
		{"суборд — отсев", func(b *pkgmodel.Bond) { b.SubordinatedFlag = true }, false},
		{"только для квалов — отсев", func(b *pkgmodel.Bond) { b.ForQualInvestorFlag = true }, false},
		{"бессрочная — отсев", func(b *pkgmodel.Bond) { b.PerpetualFlag = true }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := goodBond()
			tc.mut(b)
			if got := PassesReliability(b); got != tc.want {
				t.Fatalf("PassesReliability = %v, want %v", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 3: Запустить тест — убедиться, что не компилируется**

Run: `go test ./internal/service/trading_strategy/bonds/pipeline/ -run TestPassesReliability -v`
Expected: FAIL — `PassesReliability` не определён.

- [ ] **Step 4: Реализовать политику**

Создать `internal/service/trading_strategy/bonds/pipeline/filter.go`:

```go
package pipeline

import pkgmodel "tinvest/pkg/client/grpc/model"

// riskLevelLow — строковое представление RiskLevel_RISK_LEVEL_LOW из proto
// (модель хранит RiskLevel как .String()).
const riskLevelLow = "RISK_LEVEL_LOW"

// PassesReliability — явная политика отбора надёжных облигаций для стратегии
// «надёжные облигации». Раньше фильтр по риску был захардкожен и спрятан в
// конвертере; теперь он видимый, тестируемый и настраиваемый здесь.
func PassesReliability(b *pkgmodel.Bond) bool {
	if b.RiskLevel != riskLevelLow {
		return false
	}
	if !b.LiquidityFlag {
		return false
	}
	if b.SubordinatedFlag || b.ForQualInvestorFlag || b.PerpetualFlag {
		return false
	}
	return true
}
```

- [ ] **Step 5: Запустить тест — убедиться, что проходит**

Run: `go test ./internal/service/trading_strategy/bonds/pipeline/ -run TestPassesReliability -v`
Expected: PASS

- [ ] **Step 6: Подключить политику в `Finder`**

В `internal/service/trading_strategy/bonds/pipeline/finder.go`, внутри цикла `for _, bond := range bonds`, сразу после `time.Sleep(...)` добавить:

```go
			if !PassesReliability(bond) {
				continue
			}
```

Существующие проверки (окно погашения, `FloatingCouponFlag`, `AmortizationFlag`, `Nkd == 0`, OFZ-regex) оставить как есть — они дополняют политику надёжности критериями конкретной ступени.

- [ ] **Step 7: Снять фильтрацию из конвертера (сделать «тупым» маппером)**

В `pkg/client/grpc/converter/bond.go` заменить тело `ConvertBondsFromPb` на чистый маппинг без отсева:

```go
func ConvertBondsFromPb(in *investapi.BondsResponse) []*model.Bond {
	res := make([]*model.Bond, 0, len(in.Instruments))
	for _, bond := range in.Instruments {
		res = append(res, ConvertBondModelFromBondPb(bond))
	}
	return res
}
```

- [ ] **Step 8: Прогнать тесты пакета и сборку**

Run: `go test ./internal/service/trading_strategy/bonds/... ./pkg/client/grpc/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, сборка успешна.

- [ ] **Step 9: Коммит**

```bash
git add internal/service/trading_strategy/bonds/pipeline/filter.go internal/service/trading_strategy/bonds/pipeline/filter_test.go internal/service/trading_strategy/bonds/pipeline/finder.go pkg/client/grpc/converter/bond.go
git commit -m "feat(bonds): explicit reliability policy in Finder; convert becomes pure mapper"
```

---

### Task 3: Настоящая YTM через XIRR + guard на устаревшую цену и skip-семантика

**Files:**
- Modify: `internal/service/trading_strategy/bonds/computable/calculate_profit.go`
- Modify: `internal/service/trading_strategy/bonds/computable/calculate_profit_test.go`
- Modify: `internal/service/trading_strategy/bonds/pipeline/calculate_profit.go`

**Interfaces:**
- Consumes: `indicators.XIRR([]indicators.CashFlow) (float64, error)`, `indicators.CashFlow{Date, Amount}`.
- Produces: `calculateProfit(...) (domain.BondReport, error)` — теперь возвращает ошибку; `PercentByYear` в отчёте несёт YTM в процентах (годовых). Пайплайн `CalculateProfit` при ошибке **пропускает** бумагу (не роняет всю ступень).

Денежная модель потоков (её недисконтированная сумма равна прежнему `finalProfit`, поэтому расчёт остаётся консистентным):
- `now`: `-(bondPrice + Nkd)`
- каждый будущий купон `c`: `+c.Amount * (1 - 0.13)`
- к самому раннему будущему купону добавить налоговый щит НКД: `+Nkd * 0.13`
- в дату погашения: `+Nominal - max(Nominal - bondPrice, 0) * 0.13`

- [ ] **Step 1: Обновить существующий тест на YTM + добавить edge-кейсы**

В `internal/service/trading_strategy/bonds/computable/calculate_profit_test.go`:

1. Переименовать смысл поля `expectedYield` → это теперь YTM (годовая доходность в %). Значения-ожидания заменить на диапазонную проверку (XIRR даёт не ту же цифру, что линейная формула). Заменить блок ассерта результата на:

```go
			got, err := calculateProfit(tt.bond, tt.coupons, tt.candle)
			if err != nil {
				t.Fatalf("calculateProfit вернул ошибку: %v", err)
			}
			if got.PercentByYear < tt.ytmMin || got.PercentByYear > tt.ytmMax {
				t.Fatalf("YTM = %.2f, ожидался диапазон [%.2f, %.2f]", got.PercentByYear, tt.ytmMin, tt.ytmMax)
			}
```

2. В структуру кейса заменить поле `expectedYield float64` на `ytmMin, ytmMax float64`. Для кейса «ОФЗ с дисконтом» (цена 98.5%, до погашения 1 год, купоны 60₽): YTM должна быть заметно выше купонной (~6%) из-за дисконта — задать `ytmMin: 6.0, ytmMax: 12.0`. Для кейса «с премией» — `ytmMin: 1.0, ytmMax: 9.0`. (Значения широкие намеренно: тест проверяет корректный порядок величины и знак, а не точную арифметику.)

3. Добавить отдельный тест на устаревшую цену и на несходимость:

```go
func TestCalculateProfit_StaleCandleSkipped(t *testing.T) {
	// Прямой юнит-тест guard'а живёт в CalculateProfit (метод), где есть свеча с датой.
	// Здесь проверяем чистую функцию: поток без положительных значений → XIRR-ошибка → skip.
	bond := &pkgmodel.Bond{
		Name:         "BROKEN",
		Nominal:      1000,
		Nkd:          0,
		MaturityDate: time.Now().AddDate(1, 0, 0),
	}
	// Купонов нет, цена = номинал → нет прироста, поток одного знака → XIRR не решается.
	candle := &model.CandleItemTechAnalyse{Close: utils.CreateInternalQuotation(100, 0)}
	if _, err := calculateProfit(bond, nil, candle); err == nil {
		t.Fatal("ожидалась ошибка (skip), получен nil")
	}
}
```

- [ ] **Step 2: Запустить тест — убедиться, что не компилируется/падает**

Run: `go test ./internal/service/trading_strategy/bonds/computable/ -v`
Expected: FAIL — сигнатура `calculateProfit` ещё не возвращает `error`.

- [ ] **Step 3: Переписать `calculateProfit` на XIRR**

В `internal/service/trading_strategy/bonds/computable/calculate_profit.go` заменить хвост функции `calculateProfit` (начиная с вычисления `closePrice`) так, чтобы вернуть `(domain.BondReport, error)`. Полная новая версия функции:

```go
func calculateProfit(bond *pkgmodel.Bond, coupons []*pkgmodel.BondCoupon, candles *model.CandleItemTechAnalyse) (domain.BondReport, error) {
	const taxRate = 0.13

	now := time.Now()

	var currentYearCoupons float64
	for _, coupon := range coupons {
		if coupon.CouponDate.Year() == now.Year() {
			currentYearCoupons += utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)
		}
	}

	closePrice := utils.CombinePrice(candles.Close.Units, candles.Close.Nano)
	bondPrice := (closePrice * bond.Nominal) / 100
	totalInvestment := bondPrice + bond.Nkd

	// Текущая купонная доходность (второй показатель, оставляем как было).
	var annualCouponIncome float64
	if currentYearCoupons > 0 {
		annualCouponIncome = currentYearCoupons
	} else if len(coupons) > 0 {
		annualCouponIncome = utils.CombinePrice(coupons[0].PayOnBond.Units, coupons[0].PayOnBond.Nano) * float64(bond.CouponQuantityPerYear)
	}
	couponPercentByYear := 0.0
	if totalInvestment > 0 {
		couponPercentByYear = (annualCouponIncome * 100) / totalInvestment
	}

	// Денежные потоки для XIRR.
	flows := []indicators.CashFlow{{Date: now, Amount: -totalInvestment}}

	var firstCouponIdx = -1
	for _, coupon := range coupons {
		if !coupon.CouponDate.After(now) {
			continue
		}
		amount := utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano) * (1 - taxRate)
		flows = append(flows, indicators.CashFlow{Date: coupon.CouponDate, Amount: amount})
		if firstCouponIdx == -1 {
			firstCouponIdx = len(flows) - 1
		}
	}
	// Налоговый щит НКД — к самому раннему будущему купону.
	if firstCouponIdx != -1 {
		flows[firstCouponIdx].Amount += bond.Nkd * taxRate
	}

	// Возврат номинала за вычетом налога на прирост (цена ниже номинала).
	var priceTax float64
	if diff := bond.Nominal - bondPrice; diff > 0 {
		priceTax = diff * taxRate
	}
	flows = append(flows, indicators.CashFlow{Date: bond.MaturityDate, Amount: bond.Nominal - priceTax})

	ytm, err := indicators.XIRR(flows)
	if err != nil {
		return domain.BondReport{}, err
	}

	// Совокупная прибыль и линейная годовая прибыль в деньгах (второстепенные показатели).
	var totalCouponsNet float64
	for _, f := range flows[1:] {
		totalCouponsNet += f.Amount
	}
	finalProfit := totalCouponsNet - totalInvestment
	daysToMaturity := int(bond.MaturityDate.Sub(now).Hours() / 24)
	if daysToMaturity < 1 {
		daysToMaturity = 1
	}
	profitPerYear := (finalProfit * 365) / float64(daysToMaturity)

	return factory.CreateBondReport(bond, finalProfit, profitPerYear, ytm*100, couponPercentByYear), nil
}
```

Добавить импорт `"tinvest/pkg/indicators"` в файл. Удалить теперь неиспользуемые локальные переменные прежней реализации (`totalFutureCoupons`, `couponTax`, `nominalPriceTax`, `totalReturn`, `percentByYear`) — они заменены потоками выше.

- [ ] **Step 4: Обновить вызывающий метод `CalculateProfit` — guard устаревшей цены + проброс ошибки**

В том же файле, в методе `func (s *service) CalculateProfit(...)`, добавить константу порога и guard перед `return`, и пробросить ошибку от `calculateProfit`:

```go
const maxCandleAgeDays = 7 // страховка от неликвида: цена не должна быть слишком старой
```

Заменить финальный `return calculateProfit(bond, coupons, candles[len(candles)-1]), nil` на:

```go
	last := candles[len(candles)-1]
	if time.Since(last.Time) > maxCandleAgeDays*24*time.Hour {
		return domain.BondReport{}, errors.New("stale candle: price too old")
	}

	return calculateProfit(bond, coupons, last)
```

(`errors` уже импортирован в файле.)

- [ ] **Step 5: Пайплайн `CalculateProfit` — пропускать бумагу при ошибке, не ронять ступень**

В `internal/service/trading_strategy/bonds/pipeline/calculate_profit.go` заменить тело цикла так, чтобы ошибка приводила к `continue` (skip), а не к `close(doneCh)`:

```go
		for bond := range bondsCh {
			profit, err := computableSrc.CalculateProfit(ctx, bond)
			if err != nil {
				logger.ErrorContext(ctx, "error in CalculateProfit", slog.String("error_msg", err.Error()))
				continue
			}

			select {
			case <-doneCh:
				return
			case out <- profit:
			}
		}
```

(Это заодно устраняет латентный баг: прежний `close(doneCh)` при второй ошибке в ступени паниковал бы двойным закрытием канала.)

- [ ] **Step 6: Запустить тесты и сборку**

Run: `go test ./internal/service/trading_strategy/bonds/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, сборка успешна.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/bonds/computable/calculate_profit.go internal/service/trading_strategy/bonds/computable/calculate_profit_test.go internal/service/trading_strategy/bonds/pipeline/calculate_profit.go
git commit -m "feat(bonds): real YTM via XIRR, stale-price guard, skip-on-error pipeline"
```

---

### Task 4: Явная конфигурация лесенки

**Files:**
- Create: `internal/service/trading_strategy/bonds/ladder.go`
- Create: `internal/service/trading_strategy/bonds/ladder_test.go`
- Modify: `internal/service/trading_strategy/bonds/trade.go`

**Interfaces:**
- Produces: `type Rung struct { IsOfz bool; From, To time.Time }` и `DefaultLadder(now time.Time) []Rung` — ступени лесенки с теми же границами, что были захардкожены.

- [ ] **Step 1: Написать падающий тест конфигурации лесенки**

Создать `internal/service/trading_strategy/bonds/ladder_test.go`:

```go
package bonds

import (
	"testing"
	"time"
)

func TestDefaultLadder(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rungs := DefaultLadder(now)

	if len(rungs) != 4 {
		t.Fatalf("ожидалось 4 ступени, получено %d", len(rungs))
	}
	// Три ОФЗ-ступени + одна корпоративная.
	ofz := 0
	for _, r := range rungs {
		if r.IsOfz {
			ofz++
		}
		if !r.To.After(r.From) {
			t.Fatalf("ступень с некорректными границами: %+v", r)
		}
	}
	if ofz != 3 {
		t.Fatalf("ожидалось 3 ОФЗ-ступени, получено %d", ofz)
	}
	// Первая ОФЗ-ступень: 180д..2г.
	if !rungs[0].From.Equal(now.AddDate(0, 0, 180)) || !rungs[0].To.Equal(now.AddDate(2, 0, 0)) {
		t.Fatalf("границы первой ступени неверны: %+v", rungs[0])
	}
}
```

- [ ] **Step 2: Запустить тест — убедиться, что не компилируется**

Run: `go test ./internal/service/trading_strategy/bonds/ -run TestDefaultLadder -v`
Expected: FAIL — `DefaultLadder`/`Rung` не определены.

- [ ] **Step 3: Реализовать конфиг лесенки**

Создать `internal/service/trading_strategy/bonds/ladder.go`:

```go
package bonds

import "time"

// Rung — одна ступень облигационной лесенки: окно погашения и тип бумаг.
type Rung struct {
	IsOfz bool
	From  time.Time
	To    time.Time
}

// DefaultLadder возвращает ступени лесенки для стратегии «надёжные облигации».
// Границы те же, что раньше были захардкожены в trade.go; вынесены сюда, чтобы
// их было видно, тестировать и легко менять.
func DefaultLadder(now time.Time) []Rung {
	return []Rung{
		{IsOfz: true, From: now.AddDate(0, 0, 180), To: now.AddDate(2, 0, 0)},
		{IsOfz: true, From: now.AddDate(2, 0, 0), To: now.AddDate(6, 0, 0)},
		{IsOfz: true, From: now.AddDate(6, 0, 0), To: now.AddDate(16, 0, 0)},
		{IsOfz: false, From: now.AddDate(0, 0, 180), To: now.AddDate(3, 0, 0)},
	}
}
```

- [ ] **Step 4: Запустить тест — убедиться, что проходит**

Run: `go test ./internal/service/trading_strategy/bonds/ -run TestDefaultLadder -v`
Expected: PASS

- [ ] **Step 5: Переписать `trade.go` на цикл по лесенке**

Заменить тело `Trade` в `internal/service/trading_strategy/bonds/trade.go` (после получения `bonds`) на:

```go
	now := time.Now()
	for _, r := range DefaultLadder(now) {
		wg.Add(1)
		go func(r Rung) {
			doneCh := make(chan struct{})
			pipeline.Sender(
				ctx,
				pipeline.CalculateProfit(
					ctx,
					doneCh,
					pipeline.Finder(doneCh, bonds, r.IsOfz, r.From, r.To),
					computable.NewService(s.instrumentServiceGrpcClient, s.marketDataServiceGrpcClient),
				),
				tg,
				&wg,
				r.From,
				r.To,
			)
		}(r)
	}
	wg.Wait()

	return nil
```

- [ ] **Step 6: Прогнать тесты и сборку**

Run: `go test ./internal/service/trading_strategy/bonds/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, сборка успешна.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/bonds/ladder.go internal/service/trading_strategy/bonds/ladder_test.go internal/service/trading_strategy/bonds/trade.go
git commit -m "refactor(bonds): extract explicit ladder config from hardcoded trade.go"
```

---

### Task 5: Отчёт и вывод — YTM/сектор/надёжность + сортировка по YTM

**Files:**
- Modify: `internal/domain/bond_report.go`
- Modify: `internal/service/trading_strategy/bonds/factory/factory.go` (файл `CreateBondReport`)
- Modify: `internal/service/trading_strategy/bonds/notification/telegram.go`
- Create: `internal/service/trading_strategy/bonds/notification/telegram_test.go`
- Modify: `internal/service/trading_strategy/bonds/pipeline/sender.go`
- Create: `internal/service/trading_strategy/bonds/pipeline/sender_test.go`

**Interfaces:**
- Consumes: поля `pkgmodel.Bond` (Task 1); `BondReport.PercentByYear` несёт YTM (Task 3).
- Produces: `domain.BondReport` с полями `Sector string`, `RiskLevel string`, `Liquidity bool`; `pipeline.topByYTM(reports []domain.BondReport, n int) []domain.BondReport` — сортировка ступени по YTM.

- [ ] **Step 1: Написать падающий тест сортировки по YTM**

Создать `internal/service/trading_strategy/bonds/pipeline/sender_test.go`:

```go
package pipeline

import (
	"testing"
	"tinvest/internal/domain"
)

func TestTopByYTM(t *testing.T) {
	in := []domain.BondReport{
		{Name: "A", PercentByYear: 10},
		{Name: "B", PercentByYear: 14},
		{Name: "C", PercentByYear: 12},
	}
	got := topByYTM(in, 2)
	if len(got) != 2 {
		t.Fatalf("ожидалось 2, получено %d", len(got))
	}
	if got[0].Name != "B" || got[1].Name != "C" {
		t.Fatalf("порядок по YTM неверен: %+v", got)
	}
}
```

- [ ] **Step 2: Запустить тест — убедиться, что не компилируется**

Run: `go test ./internal/service/trading_strategy/bonds/pipeline/ -run TestTopByYTM -v`
Expected: FAIL — `topByYTM` не определён.

- [ ] **Step 3: Добавить поля в `domain.BondReport`**

В `internal/domain/bond_report.go`, в `type BondReport struct`, дописать:

```go
	Sector    string
	RiskLevel string
	Liquidity bool
```

- [ ] **Step 4: Заполнить новые поля в фабрике**

В `internal/service/trading_strategy/bonds/factory/factory.go`, в возвращаемый `domain.BondReport{...}`, дописать:

```go
		Sector:    bond.Sector,
		RiskLevel: bond.RiskLevel,
		Liquidity: bond.LiquidityFlag,
```

- [ ] **Step 5: Реализовать `topByYTM` и подключить в `Sender`**

В `internal/service/trading_strategy/bonds/pipeline/sender.go`:

1. Добавить функцию:

```go
func topByYTM(reports []domain.BondReport, n int) []domain.BondReport {
	c := collection.New[domain.BondReport]()
	for _, r := range reports {
		c.Add(r)
	}
	return c.GetTopByCriteria(func(i, j domain.BondReport) bool {
		return i.PercentByYear > j.PercentByYear
	}, n)
}
```

2. Заменить в `Sender` блок с `sortByCouponYield`/`GetTopByCriteria` на:

```go
	sortedResult := topByYTM(collectionBond.GetAll(), 10)
```

Удалить теперь неиспользуемую переменную `sortByCouponYield` и связанную логику переключения критерия.

- [ ] **Step 6: Запустить тест сортировки — убедиться, что проходит**

Run: `go test ./internal/service/trading_strategy/bonds/pipeline/ -run TestTopByYTM -v`
Expected: PASS

- [ ] **Step 7: Написать падающий тест вывода (YTM-метка + сектор + шапка политики)**

Создать `internal/service/trading_strategy/bonds/notification/telegram_test.go`:

```go
package notification

import (
	"strings"
	"testing"
	"time"
	"tinvest/internal/domain"
)

func TestSend_ContainsYTMAndSectorAndPolicy(t *testing.T) {
	bonds := []domain.BondReport{
		{Name: "ОФЗ 26238", PercentByYear: 14.2, CouponPercentByYear: 12.1, Sector: "government", ExecutionDate: time.Now()},
	}
	msg := Send(bonds, time.Now(), time.Now().AddDate(1, 0, 0))

	if !strings.Contains(msg, "YTM") {
		t.Fatal("в сообщении нет метки YTM")
	}
	if !strings.Contains(msg, "government") {
		t.Fatal("в сообщении нет сектора")
	}
	if !strings.Contains(msg, "LOW risk") {
		t.Fatal("в сообщении нет строки политики отбора")
	}
}
```

- [ ] **Step 8: Запустить тест вывода — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/bonds/notification/ -run TestSend_ContainsYTMAndSectorAndPolicy -v`
Expected: FAIL — нет метки YTM/сектора/политики.

- [ ] **Step 9: Обновить формат `Send`**

В `internal/service/trading_strategy/bonds/notification/telegram.go`:

1. После строки типа облигаций (после `fmt.Fprintf(..., "🏛️ <b>%s</b>\n\n", bondType)`) добавить строку политики отбора:

```go
	notifyMessageBuilder.WriteString("🛡️ <i>Отбор: только LOW risk, несубординированные, ликвидные</i>\n\n")
```

2. В блоке метрик заменить строку `PercentByYear` на явную метку YTM и добавить сектор. Заменить существующий `WriteString(...)` блок метрик на:

```go
		notifyMessageBuilder.WriteString(
			"💰 <b>Доходность к погашению (YTM):</b> " + formatPercent(bond.PercentByYear) + "\n" +
				"🎯 <b>Купонная доходность в год:</b> " + formatPercent(bond.CouponPercentByYear) + "\n" +
				"📈 <b>Прибыль/год:</b> " + formatMoney(bond.ManyByYear) + "₽\n" +
				"💳 <b>НКД:</b> " + formatMoney(bond.Nkd) + "₽\n" +
				"🏢 <b>Сектор:</b> " + sectorOrDash(bond.Sector) + "\n" +
				"⏰ <b>Погашение:</b> " + bond.ExecutionDate.Format("02.01.2006") + "\n")
```

3. Добавить хелпер (пустой сектор — прочерк):

```go
func sectorOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return htmlEscape(s)
}
```

(Строку про `FinalSum` «Доходность к погашению» убрать — метка переехала на YTM; `FinalSum` больше не выводим, чтобы не путать деньги с процентами.)

- [ ] **Step 10: Прогнать все тесты пакета bonds, сборку и линт**

Run: `go test ./internal/... ./pkg/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, сборка успешна.

- [ ] **Step 11: Коммит**

```bash
git add internal/domain/bond_report.go internal/service/trading_strategy/bonds/factory/factory.go internal/service/trading_strategy/bonds/notification/telegram.go internal/service/trading_strategy/bonds/notification/telegram_test.go internal/service/trading_strategy/bonds/pipeline/sender.go internal/service/trading_strategy/bonds/pipeline/sender_test.go
git commit -m "feat(bonds): YTM/sector/policy in Telegram output, sort ladder rung by YTM"
```

---

### Финальная проверка (после всех задач)

- [ ] **Прогнать полный гейт CI**

Run: `./bin/mage ci`
Expected: линт зелёный, `go test -race ./...` проходит, mock-drift отсутствует.

Если `./bin/mage` не установлен: `go test -race ./internal/... ./pkg/... ./cmd/...` и `golangci-lint run` (при наличии).
