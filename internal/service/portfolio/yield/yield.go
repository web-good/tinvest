package yield

import (
	"context"
	"fmt"
	"time"

	"tinvest/internal/domain"
	"tinvest/internal/service/portfolio/yield/notification"
	"tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
	"tinvest/pkg/logger"
)

// PortfolioYieldYTD computes the year-to-date portfolio yield, stores a snapshot,
// and sends the result to the given Telegram chat.
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

		ops, err := s.operationsServiceClient.GetCashFlowOperations(ctx, acc.ID, periodStart, periodEnd)
		if err != nil {
			return fmt.Errorf("failed to get cash flow operations for account %s: %w", acc.ID, err)
		}
		allOps = append(allOps, ops...)
	}

	flows, deposits, withdrawals := toCashFlows(allOps)
	netDeposits := deposits - withdrawals

	// Read start-of-year value BEFORE appending today's snapshot.
	vStart, ok, err := s.snapshots.valueAtYearStart(year, s.manualStartValue)
	if err != nil {
		return fmt.Errorf("failed to read snapshot store: %w", err)
	}

	// Always append today's snapshot (best-effort; don't fail the whole run).
	if appendErr := s.snapshots.append(Snapshot{Date: now, TotalValue: vEnd}); appendErr != nil {
		logger.ErrorContext(ctx, "failed to append portfolio snapshot", "error", appendErr)
	}

	y := domain.PortfolioYield{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		EndValue:    vEnd,
		Deposits:    deposits,
		Withdrawals: withdrawals,
		NetDeposits: netDeposits,
	}

	if !ok {
		y.XIRRAvailable = false
		y.Note = "Недостаточно данных: стоимость портфеля на начало года неизвестна. Снимок сохранён — расчёт станет доступен в следующем периоде."
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
