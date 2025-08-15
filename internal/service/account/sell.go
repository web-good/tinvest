package account

import "time"

func (a *account) Sell(ID string, price float64, sellingTime time.Time) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.ExistInstrument(ID) {
		return nil
	}

	pricePosition := price * float64(a.account.OrderLines[ID].Quantity)

	if price > a.account.OrderLines[ID].PurchasePrice {
		a.account.Report.Win = a.account.Report.Win + 1
	} else {
		a.account.Report.Lose = a.account.Report.Lose + 1
	}

	orderLine := *a.account.OrderLines[ID]
	orderLine.SellingPrice = price
	orderLine.SellingTime = sellingTime
	a.account.Report.Logs = append(a.account.Report.Logs, orderLine)
	delete(a.account.OrderLines, ID)
	a.account.Amount = a.account.Amount + pricePosition

	return nil
}
