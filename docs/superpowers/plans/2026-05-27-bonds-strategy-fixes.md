# Bonds Strategy Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 6 correctness and stability issues in the bonds trading strategy (`internal/service/trading_strategy/bonds/`).

**Architecture:** All fixes are localized within the bonds strategy package. Core calculation logic lives in `computable/calculate_profit.go`. Pipeline orchestration in `trade.go` and `pipeline/`. Notifications in `notification/telegram.go`. Changes don't affect any other strategies or packages.

**Tech Stack:** Go, Tinkoff Invest gRPC API, Telegram Bot API.

---

### Task 1: Fix coupon tax base and coupon yield denominator

**Context:** Two calculation bugs in `calculateProfit()`:
1. Coupon tax is computed on full coupon sum (`totalCoupons * 0.13`), but НКД paid at purchase reduces the tax base. Correct: `(totalCoupons - НКД) * 0.13`.
2. Coupon yield uses `bondPrice` as denominator, but real investment is `bondPrice + НКД`. Correct denominator: `bondPrice + bond.Nkd`.

**Files:**
- Modify: `internal/service/trading_strategy/bonds/computable/calculate_profit.go:85-88`
- Test: `internal/service/trading_strategy/bonds/computable/calculate_profit_test.go`

- [ ] **Step 1: Update test expected values to match corrected formulas**

In `calculate_profit_test.go`, update test cases:

**ОФЗ с дисконтом** (bondPrice=985, НКД=15.5, totalCoupons=60):
- couponTax: (60 − 15.5) × 0.13 = 5.785 (was 7.8)
- nominalPriceTax: 15 × 0.13 = 1.95 (unchanged)
- finalProfit: 1060 − 1000.5 − 5.785 − 1.95 = 51.765 → round to 51.8
- profitPerYear ≈ 51.8 (365/365), percentByYear ≈ 5.2
- couponPercentByYear: (30 × 100) / 1000.5 = 3.0%

**Корпоративная облигация с премией** (bondPrice=1020, НКД=25, totalCoupons=180):
- couponTax: (180 − 25) × 0.13 = 20.15 (was 23.4)
- finalProfit: 1180 − 1045 − 20.15 = 114.85 → round to 114.9
- profitPerYear = 114.85 × 365/730 = 57.425, percentByYear = 57.425 × 100 / 1045 = 5.5
- couponPercentByYear (only 1 coupon in current year): (45 × 100) / 1045 = 4.3%

**TestCalculateProfit_AllYearCouponsIncludingPaid** (bondPrice=990, НКД=10):
- couponPercentByYear: (100 × 100) / 1000 = 10.0% (now exact match, was relying on tolerance)

- [ ] **Step 2: Run tests to verify they fail with current code**

Run: `go test ./internal/service/trading_strategy/bonds/computable/ -v -run TestCalculateProfit`

- [ ] **Step 3: Fix `calculateProfit` — coupon tax and coupon yield**

In `calculate_profit.go`:
- Line 85: change `couponPercentByYear := (annualCouponIncome * 100) / bondPrice` → `couponPercentByYear := (annualCouponIncome * 100) / totalInvestment` (move after `totalInvestment` computation)
- Line 88: change `couponTax := totalCoupons * 0.13` → `couponTax := (totalCoupons - bond.Nkd) * 0.13` with guard `if couponTax < 0 { couponTax = 0 }`

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/bonds/computable/ -v -run TestCalculateProfit`

- [ ] **Step 5: Commit**

```
fix(bonds): correct coupon tax base and coupon yield denominator
```

---

### Task 2: Fetch coupons from year start + separate future vs year coupons

**Context:** `GetBondCoupons` starts from `time.Now()`, missing already-paid coupons in the current year. This underestimates `currentYearCoupons`. The fix:
1. Fetch coupons from January 1 of the current year.
2. In `calculateProfit`, split iteration: `totalCoupons` counts only future coupons (after now), `currentYearCoupons` counts all coupons in the current year.

**Files:**
- Modify: `internal/service/trading_strategy/bonds/computable/calculate_profit.go:41-47` (CalculateProfit method) and lines 50-69 (calculateProfit function)
- Test: `internal/service/trading_strategy/bonds/computable/calculate_profit_test.go`

- [ ] **Step 1: Update `calculateProfit` to split future vs year coupons**

Rename `totalCoupons` → `totalFutureCoupons`. Add filter `coupon.CouponDate.After(now)` for future-only sum. Keep `currentYearCoupons` logic unchanged (it already filters by year).

- [ ] **Step 2: Update `CalculateProfit` method to fetch from year start**

Change:
```go
coupons, _ := s.instrumentServiceGrpcClient.GetBondCoupons(bond.Id, time.Now(), bond.MaturityDate)
```
To:
```go
yearStart := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
coupons, _ := s.instrumentServiceGrpcClient.GetBondCoupons(bond.Id, yearStart, bond.MaturityDate)
```

- [ ] **Step 3: Run tests to verify**

Run: `go test ./internal/service/trading_strategy/bonds/computable/ -v`

- [ ] **Step 4: Commit**

```
fix(bonds): fetch coupons from year start, separate future vs annual coupon sums
```

---

### Task 3: Fix shared doneCh — prevent double-close panic

**Context:** All 4 goroutines in `Trade()` share one `doneCh` channel. If multiple goroutines encounter errors and call `close(doneCh)`, it panics. Each goroutine needs its own `doneCh`.

**Files:**
- Modify: `internal/service/trading_strategy/bonds/trade.go:18`

- [ ] **Step 1: Replace single `doneCh` with per-goroutine channels**

Replace `doneCh := make(chan struct{})` with a `doneCh` created inside each goroutine's closure.

- [ ] **Step 2: Run build to verify compilation**

Run: `go build ./internal/service/trading_strategy/bonds/...`

- [ ] **Step 3: Commit**

```
fix(bonds): use per-goroutine doneCh to prevent double-close panic
```

---

### Task 4: Fix sort criteria consistency

**Context:** Sort comparison function uses `i.ExecutionDate.Year()` to pick criteria, but only checks bond `i`, not `j`. Fix: use `dateFrom`/`dateTo` parameters (already available in Sender) to determine criteria for the entire group.

**Files:**
- Modify: `internal/service/trading_strategy/bonds/pipeline/sender.go:27-33`

- [ ] **Step 1: Replace per-bond criterion with per-group criterion**

Compute `yearsToEnd` from `dateTo` once, before the sort call. Use it as the consistent criterion.

- [ ] **Step 2: Run build to verify**

Run: `go build ./internal/service/trading_strategy/bonds/...`

- [ ] **Step 3: Commit**

```
fix(bonds): use period-based sort criterion instead of per-bond check
```

---

### Task 5: Fix notification label

**Context:** Label says "Доходность (с налогом)/год" which is ambiguous. The value is yield AFTER tax. Fix: change to "Чистая доходность/год".

**Files:**
- Modify: `internal/service/trading_strategy/bonds/notification/telegram.go:47`

- [ ] **Step 1: Update label text**

Change `"💰 <b>Доходность (с налогом)/год:</b> "` to `"💰 <b>Чистая доходность/год:</b> "`.

- [ ] **Step 2: Commit**

```
fix(bonds): clarify yield notification label to "Чистая доходность"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run all bonds tests**

Run: `go test ./internal/service/trading_strategy/bonds/... -v`

- [ ] **Step 2: Run full project build**

Run: `go build ./...`
