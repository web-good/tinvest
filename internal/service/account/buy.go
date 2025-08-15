package account

import (
	"github.com/pkg/errors"
	"math"
	"tinvest/internal/domain/backtest"
)

func (a *account) Buy(ID string, line *backtest.OrderLine) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ExistInstrument(ID) {
		return errors.New("instrument already exists")
	}

	a.account.OrderLines[ID] = line
	procentPrice := a.account.Amount * 5 / 100

	line.Quantity = int(math.Floor(procentPrice / line.PurchasePrice))
	if line.Quantity <= 0 {
		return ErrQuantity
	}

	price := float64(line.Quantity) * line.PurchasePrice

	if a.account.Amount < price {
		return ErrNoMany
	}

	a.account.Amount = a.account.Amount - price

	return nil
}
