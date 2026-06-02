package notification_test

import (
	"strings"
	"testing"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/service/portfolio/yield/notification"
)

func makeYield(xirr float64, xirrAvailable bool, note string) domain.PortfolioYield {
	return domain.PortfolioYield{
		PeriodStart:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		StartValue:     500000.0,
		EndValue:       620000.0,
		Deposits:       50000.0,
		Withdrawals:    10000.0,
		NetDeposits:    40000.0,
		AnnualizedXIRR:     xirr,
		PeriodReturn:       0.16,
		XIRRAvailable:      xirrAvailable,
		Note:               note,
		CouponsNet:         3200.0,
		DividendsNet:       5800.0,
		RealizedSaleProfit: -1500.0,
	}
}

func TestSend_XIRRAvailable(t *testing.T) {
	y := makeYield(0.154, true, "")
	result := notification.Send(y)

	checks := []string{
		// Header
		"Доходность портфеля",
		// Period dates
		"01.01.2026",
		"01.06.2026",
		// Start value: 500 000.0 ₽
		"500 000.0",
		// End value: 620 000.0 ₽
		"620 000.0",
		// Deposits: 50 000.0 ₽
		"50 000.0",
		// Withdrawals: 10 000.0 ₽
		"10 000.0",
		// Period return: 0.16 * 100 = 16.0%
		"16.0%",
		// Annualized XIRR: 0.154 * 100 = 15.4%
		"15.4%",
		// XIRR label appears when available
		"Годовая (XIRR):",
		// Period profit in rubles: 620000 - 500000 - 40000 = 80000
		"+80 000 ₽",
		// Annual XIRR projection: 0.154 * (500000 + 40000) = 83160
		"+83 160 ₽/год",
		// Income breakdown
		"Купоны (чистыми):",
		"3 200",
		"Дивиденды (чистыми):",
		"5 800",
		"Прибыль от продаж:",
		"−1 500 ₽",
		// Footer
		"📌 Данные актуальны на",
	}

	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("Send() output missing expected substring %q\nGot:\n%s", want, result)
		}
	}
}

func TestSend_XIRRNotAvailable_WithNote(t *testing.T) {
	note := "Недостаточно данных <test>"
	y := makeYield(0.0, false, note)
	result := notification.Send(y)

	// Note text must be HTML-escaped and present
	escapedNote := "Недостаточно данных &lt;test&gt;"
	if !strings.Contains(result, escapedNote) {
		t.Errorf("Send() output missing escaped note %q\nGot:\n%s", escapedNote, result)
	}

	// Period return is always shown
	if !strings.Contains(result, "Доходность за период") {
		t.Errorf("Send() output missing 'Доходность за период'\nGot:\n%s", result)
	}

	// Footer present
	if !strings.Contains(result, "📌 Данные актуальны на") {
		t.Errorf("Send() output missing footer\nGot:\n%s", result)
	}

	// The "Годовая (XIRR):" label with a number must NOT appear
	if strings.Contains(result, "Годовая (XIRR):") {
		t.Errorf("Send() output must NOT contain 'Годовая (XIRR):' when XIRR unavailable\nGot:\n%s", result)
	}
}

func TestSend_XIRRNotAvailable_EmptyNote(t *testing.T) {
	y := makeYield(0.0, false, "")
	result := notification.Send(y)

	// Should fall back to a default message — not an empty italic tag
	if !strings.Contains(result, "Годовая доходность пока недоступна") {
		t.Errorf("Send() output missing default XIRR unavailable text\nGot:\n%s", result)
	}

	// Must not show numeric XIRR label
	if strings.Contains(result, "Годовая (XIRR):") {
		t.Errorf("Send() output must NOT contain 'Годовая (XIRR):' when XIRR unavailable\nGot:\n%s", result)
	}
}
