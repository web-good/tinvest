package yield

import (
	"math"

	"tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

// depositTypeIDs maps Tinkoff OperationType enum values that represent money
// flowing INTO the brokerage account (deposits).
var depositTypeIDs = map[int32]bool{
	1:  true, // INPUT
	51: true, // INPUT_SWIFT
	54: true, // INPUT_ACQUIRING
	60: true, // INP_MULTI
}

// withdrawalTypeIDs maps Tinkoff OperationType enum values that represent money
// flowing OUT of the brokerage account back to the investor (withdrawals).
var withdrawalTypeIDs = map[int32]bool{
	9:  true, // OUTPUT
	50: true, // OUTPUT_SWIFT
	53: true, // OUTPUT_ACQUIRING
	59: true, // OUT_MULTI
}

// Operation-type IDs for the income breakdown (Tinkoff OperationType enum values).
var (
	couponTypeIDs      = map[int32]bool{23: true}                    // COUPON
	couponTaxTypeIDs   = map[int32]bool{2: true, 33: true}           // BOND_TAX, BOND_TAX_PROGRESSIVE
	dividendTypeIDs    = map[int32]bool{21: true, 43: true}          // DIVIDEND, DIV_EXT
	dividendTaxTypeIDs = map[int32]bool{8: true, 34: true}           // DIVIDEND_TAX, DIVIDEND_TAX_PROGRESSIVE
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

// toCashFlows converts deposit/withdrawal operations into signed XIRR cash flows.
// Deposits become negative amounts (money into the portfolio from the investor's
// perspective); withdrawals become positive amounts (money returning to the investor).
// The Payment magnitude is taken as the absolute value; the sign is derived from the
// operation type, not the API payment sign, to avoid double-sign bugs.
// Operations of any other type are silently ignored. Returns the flows plus the period
// totals (deposits and withdrawals as positive sums).
func toCashFlows(ops []model.CashOperation) (flows []indicators.CashFlow, deposits, withdrawals float64) {
	for _, op := range ops {
		abs := math.Abs(op.Payment)
		switch {
		case depositTypeIDs[op.TypeID]:
			flows = append(flows, indicators.CashFlow{Date: op.Date, Amount: -abs})
			deposits += abs
		case withdrawalTypeIDs[op.TypeID]:
			flows = append(flows, indicators.CashFlow{Date: op.Date, Amount: abs})
			withdrawals += abs
		}
		// All other types are ignored.
	}
	return flows, deposits, withdrawals
}
