package analyze

import (
	"context"
	"fmt"
	"time"

	"tinvest/internal/utils"
)

func (s service) BondsPortfolio(ctx context.Context, chatID int64) error {
	accounts, err := s.usersServiceClient.GetAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get accounts: %w", err)
	}
	if len(accounts) == 0 {
		return nil
	}

	// Buckets: 0-3 years, 3-6 years, 6+ years
	now := time.Now()
	level1End := now.AddDate(3, 0, 0) // 3 years
	level2End := now.AddDate(6, 0, 0) // 6 years

	var level1Sum, level2Sum, level3Sum float64
	issuerSums := make(map[string]float64)
	var totalBondSum float64

	for _, account := range accounts {
		portfolio, err := s.operationsServiceClient.GetPortfolio(ctx, account.ID)
		if err != nil {
			return fmt.Errorf("failed to get portfolio for account %s: %w", account.ID, err)
		}
		if len(portfolio) == 0 {
			continue
		}

		for _, position := range portfolio {
			if position.InstrumentType != "bond" {
				continue
			}
			bond, err := s.instrumentServiceGrpcClient.BondByID(ctx, position.ShareID)
			if err != nil {
				return fmt.Errorf("failed to get bond %s: %w", position.ShareID, err)
			}
			marketValue := float64(position.Quantity) * utils.CombinePrice(position.Price.Units, position.Price.Nano)
			totalBondSum += marketValue

			// Determine bucket
			switch {
			case bond.MaturityDate.Before(level1End) || bond.MaturityDate.Equal(level1End):
				level1Sum += marketValue
			case bond.MaturityDate.Before(level2End) || bond.MaturityDate.Equal(level2End):
				level2Sum += marketValue
			default:
				level3Sum += marketValue
			}
			// Accumulate per issuer (using bond.Name as issuer)
			issuerSums[bond.Name] += marketValue
		}
	}

	if totalBondSum == 0 {
		msg := "No bonds found in portfolio"
		if s.tgClient != nil {
			_ = s.tgClient.SendMessageToChat(chatID, msg)
		}
		return nil
	}

	// Build message
	msg := fmt.Sprintf("📊 *Бонд-портфель анализ*\n\n")
	msg += fmt.Sprintf("💰 *Общая стоимость:* %.2f RUB\n", totalBondSum)
	msg += fmt.Sprintf("📅 *0-3 года:* %.2f RUB (%.2f%%)\n", level1Sum, level1Sum/totalBondSum*100)
	msg += fmt.Sprintf("📅 *3-6 лет:* %.2f RUB (%.2f%%)\n", level2Sum, level2Sum/totalBondSum*100)
	msg += fmt.Sprintf("📅 *6+ лет:* %.2f RUB (%.2f%%)\n\n", level3Sum, level3Sum/totalBondSum*100)
	msg += fmt.Sprintf("🏢 *По эмитентам:*\n")
	for issuer, sum := range issuerSums {
		msg += fmt.Sprintf("  • %s: %.2f RUB (%.2f%%)\n", issuer, sum, sum/totalBondSum*100)
	}

	// Send via Telegram
	if s.tgClient != nil {
		if err := s.tgClient.SendMessageToChat(chatID, msg); err != nil {
			return fmt.Errorf("failed to send telegram message: %w", err)
		}
	}

	return nil
}
