package yield

import (
	"testing"
	"time"

	"tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

func makeOp(typeID int32, payment float64) model.CashOperation {
	return model.CashOperation{
		Date:    time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		TypeID:  typeID,
		Payment: payment,
	}
}

func TestToCashFlows_DepositPositivePayment(t *testing.T) {
	// Deposit (TypeID 1 = INPUT) with positive payment -> flow Amount -1000.
	flows, deposits, withdrawals := toCashFlows([]model.CashOperation{makeOp(1, 1000)})

	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].Amount != -1000 {
		t.Errorf("flow Amount = %v, want -1000", flows[0].Amount)
	}
	if deposits != 1000 {
		t.Errorf("deposits = %v, want 1000", deposits)
	}
	if withdrawals != 0 {
		t.Errorf("withdrawals = %v, want 0", withdrawals)
	}
}

func TestToCashFlows_DepositNegativePayment(t *testing.T) {
	// Deposit whose API Payment is already negative -> still Amount -1000 (abs normalization).
	flows, deposits, _ := toCashFlows([]model.CashOperation{makeOp(1, -1000)})

	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].Amount != -1000 {
		t.Errorf("flow Amount = %v, want -1000 (abs normalization)", flows[0].Amount)
	}
	if deposits != 1000 {
		t.Errorf("deposits = %v, want 1000", deposits)
	}
}

func TestToCashFlows_Withdrawal(t *testing.T) {
	// Withdrawal (TypeID 9 = OUTPUT) with Payment 500 -> Amount +500; withdrawals 500.
	flows, deposits, withdrawals := toCashFlows([]model.CashOperation{makeOp(9, 500)})

	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if flows[0].Amount != 500 {
		t.Errorf("flow Amount = %v, want 500", flows[0].Amount)
	}
	if deposits != 0 {
		t.Errorf("deposits = %v, want 0", deposits)
	}
	if withdrawals != 500 {
		t.Errorf("withdrawals = %v, want 500", withdrawals)
	}
}

func TestToCashFlows_UnknownTypeIgnored(t *testing.T) {
	// TypeID 15 (BUY) is not a deposit/withdrawal -> ignored.
	flows, deposits, withdrawals := toCashFlows([]model.CashOperation{makeOp(15, 1000)})

	if len(flows) != 0 {
		t.Errorf("expected 0 flows for unknown type, got %d", len(flows))
	}
	if deposits != 0 {
		t.Errorf("deposits = %v, want 0", deposits)
	}
	if withdrawals != 0 {
		t.Errorf("withdrawals = %v, want 0", withdrawals)
	}
}

func TestToCashFlows_MixedList(t *testing.T) {
	// Mixed list: deposit, withdrawal, unknown, all deposit variants, all withdrawal variants.
	ops := []model.CashOperation{
		makeOp(1, 1000),  // INPUT deposit
		makeOp(51, 2000), // INPUT_SWIFT deposit
		makeOp(54, 500),  // INPUT_ACQUIRING deposit
		makeOp(60, 300),  // INP_MULTI deposit
		makeOp(9, 800),   // OUTPUT withdrawal
		makeOp(50, 600),  // OUTPUT_SWIFT withdrawal
		makeOp(53, 400),  // OUTPUT_ACQUIRING withdrawal
		makeOp(59, 200),  // OUT_MULTI withdrawal
		makeOp(15, 9999), // BUY - ignored
	}

	flows, deposits, withdrawals := toCashFlows(ops)

	// 8 classified ops (4 deposits + 4 withdrawals), 1 ignored.
	if len(flows) != 8 {
		t.Fatalf("expected 8 flows, got %d", len(flows))
	}

	// Deposits: 1000+2000+500+300 = 3800
	wantDeposits := 1000.0 + 2000.0 + 500.0 + 300.0
	if deposits != wantDeposits {
		t.Errorf("deposits = %v, want %v", deposits, wantDeposits)
	}

	// Withdrawals: 800+600+400+200 = 2000
	wantWithdrawals := 800.0 + 600.0 + 400.0 + 200.0
	if withdrawals != wantWithdrawals {
		t.Errorf("withdrawals = %v, want %v", withdrawals, wantWithdrawals)
	}

	// Verify order preserved: deposits come first (negative), then withdrawals (positive).
	for i, f := range flows[:4] {
		if f.Amount >= 0 {
			t.Errorf("flows[%d].Amount = %v, want negative (deposit)", i, f.Amount)
		}
	}
	for i, f := range flows[4:] {
		if f.Amount <= 0 {
			t.Errorf("flows[%d+4].Amount = %v, want positive (withdrawal)", i, f.Amount)
		}
	}
}

func TestToCashFlows_EmptyInput(t *testing.T) {
	flows, deposits, withdrawals := toCashFlows(nil)

	if len(flows) != 0 {
		t.Errorf("expected empty flows, got %d", len(flows))
	}
	if deposits != 0 {
		t.Errorf("deposits = %v, want 0", deposits)
	}
	if withdrawals != 0 {
		t.Errorf("withdrawals = %v, want 0", withdrawals)
	}
}

func TestToCashFlows_DatePreserved(t *testing.T) {
	// Verify that the Date field is forwarded correctly into the CashFlow.
	date := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	op := model.CashOperation{Date: date, TypeID: 1, Payment: 5000}

	flows, _, _ := toCashFlows([]model.CashOperation{op})

	if len(flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(flows))
	}
	if !flows[0].Date.Equal(date) {
		t.Errorf("flow Date = %v, want %v", flows[0].Date, date)
	}
}

func TestToCashFlows_ReturnType(t *testing.T) {
	// Ensure returned type is []indicators.CashFlow (compile-time check via assignment).
	var _ []indicators.CashFlow
	flows, _, _ := toCashFlows([]model.CashOperation{makeOp(1, 100)})
	var _ = flows // use flows to satisfy compiler
}

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
