package yield

import (
	"context"
	"fmt"
	"time"

	"tinvest/internal/domain"
	"tinvest/internal/service/portfolio/yield/notification"
	"tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

// PortfolioYieldYTD computes the year-to-date portfolio yield and sends the
// result to the given Telegram chat.
func (s *service) PortfolioYieldYTD(ctx context.Context, chatID int64) error {
	now := time.Now()
	year := now.Year()
	periodStart := time.Date(year, time.January, 1, 0, 0, 0, 0, now.Location())
	periodEnd := now

	accounts, err := s.usersServiceClient.GetAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}
	if len(accounts) == 0 {
		return nil
	}

	// Sum end value and collect cash-flow operations across all accounts.
	var vEnd float64
	var allOps []model.CashOperation

	for _, acc := range accounts {
		total, err := s.operationsServiceClient.GetPortfolioTotal(ctx, acc.ID)
		if err != nil {
			return fmt.Errorf("failed to get portfolio total for account %s: %w", acc.ID, err)
		}
		vEnd += total

		ops, err := s.operationsServiceClient.GetCashOperations(ctx, acc.ID, periodStart, periodEnd)
		if err != nil {
			return fmt.Errorf("failed to get cash operations for account %s: %w", acc.ID, err)
		}
		allOps = append(allOps, ops...)
	}

	flows, deposits, withdrawals := toCashFlows(allOps)
	couponsNet, dividendsNet, realizedSaleProfit := aggregateIncome(allOps)
	netDeposits := deposits - withdrawals

	// Start-of-year portfolio value is supplied manually via configuration
	// (PORTFOLIO_YTD_START_VALUE): the Tinkoff API does not expose historical
	// portfolio value, so it cannot be derived automatically.
	vStart := s.manualStartValue
	ok := vStart > 0

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

	if !ok {
		y.XIRRAvailable = false
		y.Note = "Недостаточно данных: укажите стоимость портфеля на начало года в переменной PORTFOLIO_YTD_START_VALUE."
	} else {
		y.StartValue = vStart

		// Non-annualized period return (simple Dietz).
		denom := vStart + netDeposits
		if denom != 0 {
			y.PeriodReturn = (vEnd - vStart - netDeposits) / denom
		}

		// Build XIRR cash-flow series:
		//   -V_start on Jan 1 (money "invested" at year start)
		//   signed deposit/withdrawal flows during the period
		//   +V_end today (terminal value returned to investor)
		series := []indicators.CashFlow{{Date: periodStart, Amount: -vStart}}
		series = append(series, flows...)
		series = append(series, indicators.CashFlow{Date: periodEnd, Amount: vEnd})

		// Guard: period must be long enough to annualize meaningfully.
		years := periodEnd.Sub(periodStart).Hours() / 24 / 365
		if years < 0.1 {
			y.XIRRAvailable = false
			y.Note = "Период слишком короткий для расчёта годовой доходности (XIRR)."
		} else {
			rate, xerr := indicators.XIRR(series)
			if xerr != nil {
				y.XIRRAvailable = false
				y.Note = "Годовую доходность (XIRR) рассчитать не удалось."
			} else {
				y.AnnualizedXIRR = rate
				y.XIRRAvailable = true
			}
		}
	}

	msg := notification.Send(y)
	if s.tgClient != nil {
		if err := s.tgClient.SendMessageToChat(chatID, msg); err != nil {
			return fmt.Errorf("failed to send telegram message: %w", err)
		}
	}

	return nil
}
