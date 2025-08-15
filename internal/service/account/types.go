package account

import (
	"sync"
	"tinvest/internal/domain/backtest"
)

type account struct {
	mu      sync.RWMutex
	account backtest.Account
}

func NewAccount(amount float64) *account {
	return &account{
		mu: sync.RWMutex{},
		account: backtest.Account{
			Amount:     amount,
			OrderLines: map[string]*backtest.OrderLine{},
			Report: backtest.Report{
				Win:  0,
				Lose: 0,
				Logs: make([]backtest.OrderLine, 0, 5),
			},
		},
	}
}
