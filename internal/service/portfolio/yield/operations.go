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
