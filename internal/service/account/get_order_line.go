package account

import "tinvest/internal/domain/backtest"

func (a *account) GetOrderLine(ID string) *backtest.OrderLine {
	return a.account.OrderLines[ID]
}
