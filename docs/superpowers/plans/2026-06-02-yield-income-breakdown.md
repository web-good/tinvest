# Разбивка дохода в YTD-алерте — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в Telegram-алерт «Доходность портфеля (с начала года)» три информационные строки — чистые купоны, чистые дивиденды и реализованную прибыль от продаж за период.

**Architecture:** Подход A — один расширенный cursor-запрос операций на счёт. Существующий метод клиента расширяется по списку типов и переименовывается; XIRR/PeriodReturn не затрагиваются (агрегатор дохода читает те же операции, что и `toCashFlows`, но по другим типам). Реализованная прибыль берётся из `OperationItem.Yield` (считает Тинькофф).

**Tech Stack:** Go 1.25, gRPC (Tinkoff Invest API), стандартный `testing`.

Спека: `docs/superpowers/specs/2026-06-02-yield-income-breakdown-design.md`

---

## File Structure

- `pkg/client/grpc/model/operation.go` — модель `CashOperation`: +поле `Yield`.
- `pkg/client/grpc/converter/operation.go` — заполнение `Yield` из `OperationItem`.
- `pkg/client/grpc/converter/operation_test.go` — новый тест на `Yield`.
- `pkg/client/grpc/operations_service_client.go` — расширение списка типов, переименование `GetCashFlowOperations` → `GetCashOperations`.
- `internal/service/portfolio/yield/operations.go` — новая `aggregateIncome`.
- `internal/service/portfolio/yield/operations_test.go` — тест `aggregateIncome`.
- `internal/domain/portfolio_yield.go` — +поля `CouponsNet`, `DividendsNet`, `RealizedSaleProfit`.
- `internal/service/portfolio/yield/yield.go` — вызов `aggregateIncome`, заполнение полей; переименованный вызов клиента.
- `internal/service/portfolio/yield/yield_test.go` — фейк: переименованный метод.
- `internal/service/portfolio/yield/notification/telegram.go` — три новые строки.
- `internal/service/portfolio/yield/notification/telegram_test.go` — проверка трёх строк.

---

## Task 1: Поле `Yield` в модели и конвертере

**Files:**
- Modify: `pkg/client/grpc/model/operation.go:25-30`
- Modify: `pkg/client/grpc/converter/operation.go:58-76`
- Test: `pkg/client/grpc/converter/operation_test.go`

- [ ] **Step 1: Написать падающий тест на Yield**

Добавить новую тест-функцию в конец `pkg/client/grpc/converter/operation_test.go`:

```go
func TestConvertCursorItemsToCashOperations_Yield(t *testing.T) {
	ts := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	items := []*investapi.OperationItem{
		{
			Date:    timestamppb.New(ts),
			Type:    investapi.OperationType_OPERATION_TYPE_SELL,
			Payment: &investapi.MoneyValue{Units: 1000, Nano: 0},
			Yield:   &investapi.MoneyValue{Units: 120, Nano: 500000000},
		},
		{
			Date:    timestamppb.New(ts),
			Type:    investapi.OperationType_OPERATION_TYPE_COUPON,
			Payment: &investapi.MoneyValue{Units: 50, Nano: 0},
			Yield:   nil, // нет Yield → 0
		},
	}

	got := ConvertCursorItemsToCashOperations(items)
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}

	const tol = 1e-9
	if d := got[0].Yield - 120.5; d < -tol || d > tol {
		t.Errorf("item[0].Yield = %v, want 120.5", got[0].Yield)
	}
	if d := got[1].Yield - 0; d < -tol || d > tol {
		t.Errorf("item[1].Yield = %v, want 0", got[1].Yield)
	}
}
```

- [ ] **Step 2: Запустить тест — должен не компилироваться/падать**

Run: `go test ./pkg/client/grpc/converter/ -run TestConvertCursorItemsToCashOperations_Yield`
Expected: FAIL — `got[0].Yield undefined (type model.CashOperation has no field Yield)`.

- [ ] **Step 3: Добавить поле `Yield` в модель**

В `pkg/client/grpc/model/operation.go` в структуру `CashOperation`:

```go
type CashOperation struct {
	Date    time.Time
	Type    string  // OperationType.String()
	TypeID  int32   // raw OperationType enum value, for robust classification
	Payment float64 // RUB amount from MoneyValue (units + nano/1e9)
	Yield   float64 // realized P&L of the operation (units + nano/1e9); meaningful for trades (SELL)
}
```

- [ ] **Step 4: Заполнить `Yield` в конвертере**

В `pkg/client/grpc/converter/operation.go`, функция `ConvertCursorItemsToCashOperations`, в теле цикла после вычисления `payment`:

```go
		var yield float64
		if y := item.GetYield(); y != nil {
			yield = float64(y.GetUnits()) + float64(y.GetNano())/1e9
		}

		res = append(res, model.CashOperation{
			Date:    item.GetDate().AsTime(),
			Type:    item.GetType().String(),
			TypeID:  int32(item.GetType()),
			Payment: payment,
			Yield:   yield,
		})
```

- [ ] **Step 5: Запустить тесты конвертера — должны пройти**

Run: `go test ./pkg/client/grpc/converter/`
Expected: PASS (включая существующий `TestConvertCursorItemsToCashOperations`).

- [ ] **Step 6: Commit**

```bash
git add pkg/client/grpc/model/operation.go pkg/client/grpc/converter/operation.go pkg/client/grpc/converter/operation_test.go
git commit -m "feat(operations): carry operation Yield into CashOperation model

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Расширить типы операций и переименовать метод клиента

**Files:**
- Modify: `pkg/client/grpc/operations_service_client.go:15-20` (интерфейс), `:67-84` (типы + метод)
- Modify: `internal/service/portfolio/yield/yield.go:41` (вызов)
- Modify: `internal/service/portfolio/yield/yield_test.go:28-30` (фейк)

- [ ] **Step 1: Расширить список типов и переименовать переменную**

В `pkg/client/grpc/operations_service_client.go` заменить блок `depositWithdrawalTypes` (строки 67-80) на:

```go
// relevantOperationTypes lists the operation types fetched for portfolio-yield
// calculation: cash deposits/withdrawals (for XIRR) plus income operations
// (coupons, dividends, sales) and their taxes (for the income breakdown).
var relevantOperationTypes = []investapi.OperationType{
	// Deposits
	investapi.OperationType_OPERATION_TYPE_INPUT,
	investapi.OperationType_OPERATION_TYPE_INPUT_SWIFT,
	investapi.OperationType_OPERATION_TYPE_INPUT_ACQUIRING,
	investapi.OperationType_OPERATION_TYPE_INP_MULTI,
	// Withdrawals
	investapi.OperationType_OPERATION_TYPE_OUTPUT,
	investapi.OperationType_OPERATION_TYPE_OUTPUT_SWIFT,
	investapi.OperationType_OPERATION_TYPE_OUTPUT_ACQUIRING,
	investapi.OperationType_OPERATION_TYPE_OUT_MULTI,
	// Coupons + tax
	investapi.OperationType_OPERATION_TYPE_COUPON,
	investapi.OperationType_OPERATION_TYPE_BOND_TAX,
	investapi.OperationType_OPERATION_TYPE_BOND_TAX_PROGRESSIVE,
	// Dividends + tax
	investapi.OperationType_OPERATION_TYPE_DIVIDEND,
	investapi.OperationType_OPERATION_TYPE_DIV_EXT,
	investapi.OperationType_OPERATION_TYPE_DIVIDEND_TAX,
	investapi.OperationType_OPERATION_TYPE_DIVIDEND_TAX_PROGRESSIVE,
	// Sales (realized profit via Yield)
	investapi.OperationType_OPERATION_TYPE_SELL,
	investapi.OperationType_OPERATION_TYPE_SELL_CARD,
	investapi.OperationType_OPERATION_TYPE_SELL_MARGIN,
}
```

- [ ] **Step 2: Переименовать метод в интерфейсе**

В `pkg/client/grpc/operations_service_client.go` в интерфейсе `OperationsServiceClient` заменить строку:

```go
	GetCashFlowOperations(ctx context.Context, accountID string, from, to time.Time) ([]model.CashOperation, error)
```

на:

```go
	GetCashOperations(ctx context.Context, accountID string, from, to time.Time) ([]model.CashOperation, error)
```

- [ ] **Step 3: Переименовать реализацию и обновить doc + ссылку на типы**

В `pkg/client/grpc/operations_service_client.go` заменить сигнатуру и комментарий метода (строки 82-84):

```go
// GetCashOperations fetches all executed operations relevant to portfolio-yield
// computation (deposits/withdrawals, coupons, dividends, sales and their taxes)
// for the given account and time range using cursor-based pagination.
func (o *operationsServiceClient) GetCashOperations(ctx context.Context, accountID string, from, to time.Time) ([]model.CashOperation, error) {
```

И внутри тела метода в `GetOperationsByCursorRequest` заменить `OperationTypes: depositWithdrawalTypes,` на `OperationTypes: relevantOperationTypes,`.

- [ ] **Step 4: Обновить вызов в сервисе**

В `internal/service/portfolio/yield/yield.go:41` заменить:

```go
		ops, err := s.operationsServiceClient.GetCashFlowOperations(ctx, acc.ID, periodStart, periodEnd)
```

на:

```go
		ops, err := s.operationsServiceClient.GetCashOperations(ctx, acc.ID, periodStart, periodEnd)
```

- [ ] **Step 5: Обновить фейк в тесте**

В `internal/service/portfolio/yield/yield_test.go:28-30` заменить метод фейка:

```go
func (f *fakeOperationsClient) GetCashOperations(_ context.Context, _ string, _, _ time.Time) ([]grpcmodel.CashOperation, error) {
	return f.cashOps, nil
}
```

- [ ] **Step 6: Собрать и прогнать тесты — компиляция и зелёные тесты**

Run: `go build ./... && go test ./pkg/client/grpc/... ./internal/service/portfolio/yield/...`
Expected: PASS, без ошибок компиляции (никаких ссылок на старое имя метода).

- [ ] **Step 7: Commit**

```bash
git add pkg/client/grpc/operations_service_client.go internal/service/portfolio/yield/yield.go internal/service/portfolio/yield/yield_test.go
git commit -m "feat(operations): fetch income operations; rename to GetCashOperations

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Агрегатор дохода `aggregateIncome`

**Files:**
- Modify: `internal/service/portfolio/yield/operations.go`
- Test: `internal/service/portfolio/yield/operations_test.go`

- [ ] **Step 1: Написать падающий тест**

Добавить в конец `internal/service/portfolio/yield/operations_test.go` (пакет `yield`):

```go
func TestAggregateIncome(t *testing.T) {
	ops := []model.CashOperation{
		{TypeID: 23, Payment: 1000}, // COUPON gross
		{TypeID: 23, Payment: 500},  // COUPON gross
		{TypeID: 2, Payment: -195},  // BOND_TAX (приходит отрицательным)
		{TypeID: 33, Payment: -30},  // BOND_TAX_PROGRESSIVE
		{TypeID: 21, Payment: 2000}, // DIVIDEND gross
		{TypeID: 43, Payment: 300},  // DIV_EXT gross
		{TypeID: 8, Payment: -260},  // DIVIDEND_TAX
		{TypeID: 34, Payment: -40},  // DIVIDEND_TAX_PROGRESSIVE
		{TypeID: 22, Yield: 1200},   // SELL realized profit
		{TypeID: 7, Yield: -300},    // SELL_CARD realized loss
		{TypeID: 18, Yield: 50},     // SELL_MARGIN
		{TypeID: 1, Payment: 99999}, // INPUT — должен игнорироваться
	}

	couponsNet, dividendsNet, saleProfit := aggregateIncome(ops)

	const tol = 1e-9
	if d := couponsNet - (1500 - 225); d < -tol || d > tol {
		t.Errorf("couponsNet = %v, want 1275", couponsNet)
	}
	if d := dividendsNet - (2300 - 300); d < -tol || d > tol {
		t.Errorf("dividendsNet = %v, want 2000", dividendsNet)
	}
	if d := saleProfit - (1200 - 300 + 50); d < -tol || d > tol {
		t.Errorf("saleProfit = %v, want 950", saleProfit)
	}
}
```

Если в `operations_test.go` ещё нет импорта модели — добавить `"tinvest/pkg/client/grpc/model"`.

- [ ] **Step 2: Запустить тест — должен падать**

Run: `go test ./internal/service/portfolio/yield/ -run TestAggregateIncome`
Expected: FAIL — `undefined: aggregateIncome`.

- [ ] **Step 3: Реализовать `aggregateIncome`**

Добавить в `internal/service/portfolio/yield/operations.go` (импорт `math` уже есть):

```go
// Operation-type IDs for the income breakdown (Tinkoff OperationType enum values).
var (
	couponTypeIDs      = map[int32]bool{23: true}            // COUPON
	couponTaxTypeIDs   = map[int32]bool{2: true, 33: true}   // BOND_TAX, BOND_TAX_PROGRESSIVE
	dividendTypeIDs    = map[int32]bool{21: true, 43: true}  // DIVIDEND, DIV_EXT
	dividendTaxTypeIDs = map[int32]bool{8: true, 34: true}   // DIVIDEND_TAX, DIVIDEND_TAX_PROGRESSIVE
	saleTypeIDs        = map[int32]bool{22: true, 7: true, 18: true} // SELL, SELL_CARD, SELL_MARGIN
)

// aggregateIncome sums the period income broken down by source. Coupons and
// dividends are returned net of withheld tax. Realized sale profit is the sum of
// the per-operation Yield (already computed by the broker) over sale operations
// and may be negative. Tax payments arrive negative from the API; magnitudes are
// taken via abs so the result does not depend on the API sign. Other operation
// types are ignored.
func aggregateIncome(ops []model.CashOperation) (couponsNet, dividendsNet, realizedSaleProfit float64) {
	var couponGross, couponTax, dividendGross, dividendTax float64
	for _, op := range ops {
		switch {
		case couponTypeIDs[op.TypeID]:
			couponGross += math.Abs(op.Payment)
		case couponTaxTypeIDs[op.TypeID]:
			couponTax += math.Abs(op.Payment)
		case dividendTypeIDs[op.TypeID]:
			dividendGross += math.Abs(op.Payment)
		case dividendTaxTypeIDs[op.TypeID]:
			dividendTax += math.Abs(op.Payment)
		case saleTypeIDs[op.TypeID]:
			realizedSaleProfit += op.Yield
		}
	}
	return couponGross - couponTax, dividendGross - dividendTax, realizedSaleProfit
}
```

- [ ] **Step 4: Запустить тест — должен пройти**

Run: `go test ./internal/service/portfolio/yield/ -run TestAggregateIncome`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/portfolio/yield/operations.go internal/service/portfolio/yield/operations_test.go
git commit -m "feat(yield): aggregate net coupons, dividends and realized sale profit

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Поля домена и проводка в сервисе

**Files:**
- Modify: `internal/domain/portfolio_yield.go:6-18`
- Modify: `internal/service/portfolio/yield/yield.go:48-64`

- [ ] **Step 1: Добавить поля в домен**

В `internal/domain/portfolio_yield.go` в структуру `PortfolioYield` (после `Note`) добавить:

```go
	CouponsNet         float64 // coupons received during the period, net of tax
	DividendsNet       float64 // dividends received during the period, net of tax
	RealizedSaleProfit float64 // realized profit from asset sales during the period (may be negative)
```

- [ ] **Step 2: Проводить агрегат в `yield.go`**

В `internal/service/portfolio/yield/yield.go` после строки `flows, deposits, withdrawals := toCashFlows(allOps)` (строка 48) добавить:

```go
	couponsNet, dividendsNet, realizedSaleProfit := aggregateIncome(allOps)
```

И в литерал `y := domain.PortfolioYield{...}` (строки 57-64) добавить новые поля:

```go
	y := domain.PortfolioYield{
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		EndValue:           vEnd,
		Deposits:           deposits,
		Withdrawals:        withdrawals,
		NetDeposits:        netDeposits,
		CouponsNet:         couponsNet,
		DividendsNet:       dividendsNet,
		RealizedSaleProfit: realizedSaleProfit,
	}
```

- [ ] **Step 3: Собрать и прогнать тесты пакета**

Run: `go build ./... && go test ./internal/service/portfolio/yield/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/portfolio_yield.go internal/service/portfolio/yield/yield.go
git commit -m "feat(yield): wire income breakdown into PortfolioYield

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Три строки в Telegram-уведомлении

**Files:**
- Modify: `internal/service/portfolio/yield/notification/telegram.go:25-26`
- Test: `internal/service/portfolio/yield/notification/telegram_test.go`

- [ ] **Step 1: Написать падающий тест**

В `internal/service/portfolio/yield/notification/telegram_test.go` дополнить фикстуру `makeYield` (после `Note: note,`) полями:

```go
		CouponsNet:         3200.0,
		DividendsNet:       5800.0,
		RealizedSaleProfit: -1500.0,
```

И в `TestSend_XIRRAvailable` в срез `checks` добавить:

```go
		// Income breakdown
		"Купоны (чистыми):",
		"3 200",
		"Дивиденды (чистыми):",
		"5 800",
		"Прибыль от продаж:",
		"−1 500 ₽", // отрицательная прибыль со знаком U+2212
```

- [ ] **Step 2: Запустить тест — должен падать**

Run: `go test ./internal/service/portfolio/yield/notification/ -run TestSend_XIRRAvailable`
Expected: FAIL — отсутствуют подстроки разбивки.

- [ ] **Step 3: Добавить строки в `Send`**

В `internal/service/portfolio/yield/notification/telegram.go` заменить строку вывода выводов (строка 26):

```go
	b.WriteString("➖ <b>Выводы:</b> " + formatMoney(y.Withdrawals) + " ₽\n\n")
```

на блок:

```go
	b.WriteString("➖ <b>Выводы:</b> " + formatMoney(y.Withdrawals) + " ₽\n")
	b.WriteString("💰 <b>Купоны (чистыми):</b> " + formatMoney(y.CouponsNet) + " ₽\n")
	b.WriteString("💎 <b>Дивиденды (чистыми):</b> " + formatMoney(y.DividendsNet) + " ₽\n")
	b.WriteString("📈 <b>Прибыль от продаж:</b> " + formatSignedMoney(y.RealizedSaleProfit) + "\n\n")
```

(`formatSignedMoney` уже добавляет знак и « ₽».)

- [ ] **Step 4: Запустить тесты нотификации — должны пройти**

Run: `go test ./internal/service/portfolio/yield/notification/`
Expected: PASS.

- [ ] **Step 5: Прогнать весь пакет yield и сборку**

Run: `go build ./... && go test ./internal/... ./pkg/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/portfolio/yield/notification/telegram.go internal/service/portfolio/yield/notification/telegram_test.go
git commit -m "feat(yield): show net coupons, dividends and sale profit in alert

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Финальная верификация (на реальных данных)

- [ ] Запустить приложение/триггер алерта на реальном аккаунте; убедиться, что три строки появились и значения правдоподобны.
- [ ] Сверить `Прибыль от продаж` с приложением Тинькофф на 1–2 реальных сделках SELL — подтвердить, что `OperationItem.Yield` действительно равен реализованной прибыли. Если расходится — зафиксировать находку и обсудить fallback на ручной cost-basis (вне текущего объёма).

---

## Self-Review (выполнено при написании плана)

- **Покрытие спеки:** типы операций (Task 2), Yield (Task 1), нетто купоны/дивиденды + прибыль продаж (Task 3), поля домена и проводка (Task 4), три строки UI (Task 5), верификация Yield на реальных данных (финальный раздел). Все пункты спеки покрыты.
- **Плейсхолдеры:** отсутствуют — каждый шаг содержит конкретный код/команду.
- **Согласованность типов:** `aggregateIncome` возвращает `(couponsNet, dividendsNet, realizedSaleProfit)`; ровно эти имена/поля используются в Task 4 и поля `CouponsNet/DividendsNet/RealizedSaleProfit` — в Task 5. Метод клиента переименован единообразно (`GetCashOperations`) в интерфейсе, реализации, вызове и фейке.
