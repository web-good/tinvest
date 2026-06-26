// Package sizing computes whole-lot order quantities for the reversion runner from a
// percentage of total account value, capped by available cash.
package sizing

import (
	"fmt"
	"math"
)

// Lots returns the number of lots to buy. It sizes by buyPct% of accountValue, then
// caps by what cash can afford. ok is false (with a human-readable reason) when the
// budget buys fewer than one lot or cash cannot cover a single lot. Inputs that make
// a lot priceless (price<=0, lot<=0) also yield ok=false.
func Lots(buyPct, accountValue, cash, price float64, lot int32) (int64, bool, string) {
	if price <= 0 || lot <= 0 {
		return 0, false, fmt.Sprintf("некорректная цена/лот (price=%.4f, lot=%d)", price, lot)
	}
	lotCost := price * float64(lot)
	budget := buyPct / 100 * accountValue
	lots := int64(math.Floor(budget / lotCost))
	if lots <= 0 {
		return 0, false, fmt.Sprintf("бюджета %.2f не хватает на 1 лот (%.2f)", budget, lotCost)
	}
	affordable := int64(math.Floor(cash / lotCost))
	if affordable <= 0 {
		return 0, false, fmt.Sprintf("кэша %.2f не хватает на 1 лот (%.2f)", cash, lotCost)
	}
	if lots > affordable {
		lots = affordable
	}
	return lots, true, ""
}
