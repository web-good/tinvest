package notification

import (
	"math"
	"strconv"
	"strings"
	"time"
	"tinvest/internal/domain"
)

// Send formats a PortfolioYield as an HTML Telegram message.
func Send(y domain.PortfolioYield) string {
	var b strings.Builder

	// Header
	b.WriteString("📈 <b>Доходность портфеля (с начала года)</b>\n\n")

	// Period
	b.WriteString("🗓️ <i>Период: " +
		y.PeriodStart.Format("02.01.2006") + " — " +
		y.PeriodEnd.Format("02.01.2006") + "</i>\n\n")

	// Portfolio values
	b.WriteString("💼 <b>Стоимость на начало года:</b> " + formatMoney(y.StartValue) + " ₽\n")
	b.WriteString("💼 <b>Текущая стоимость:</b> " + formatMoney(y.EndValue) + " ₽\n")
	b.WriteString("➕ <b>Пополнения:</b> " + formatMoney(y.Deposits) + " ₽\n")
	b.WriteString("➖ <b>Выводы:</b> " + formatMoney(y.Withdrawals) + " ₽\n")
	b.WriteString("💰 <b>Купоны (чистыми):</b> " + formatMoney(y.CouponsNet) + " ₽\n")
	b.WriteString("💎 <b>Дивиденды (чистыми):</b> " + formatMoney(y.DividendsNet) + " ₽\n")
	b.WriteString("📈 <b>Прибыль от продаж:</b> " + formatSignedMoney(y.RealizedSaleProfit) + "\n\n")

	// Returns
	periodLine := "📊 <b>Доходность за период:</b> " + formatPercent(y.PeriodReturn*100)
	if y.StartValue > 0 {
		// Actual realized profit since the start of the year, in rubles.
		periodProfit := y.EndValue - y.StartValue - y.NetDeposits
		periodLine += " (≈ " + formatSignedMoney(periodProfit) + ")"
	}
	b.WriteString(periodLine + "\n")

	if y.XIRRAvailable {
		xirrLine := "📐 <b>Годовая (XIRR):</b> " + formatPercent(y.AnnualizedXIRR*100)
		if y.StartValue > 0 {
			// Projected full-year profit at the current annualized pace:
			// rate applied to the invested capital (start value + net deposits).
			annualProjection := y.AnnualizedXIRR * (y.StartValue + y.NetDeposits)
			xirrLine += " (≈ " + formatSignedMoney(annualProjection) + "/год)"
		}
		b.WriteString(xirrLine + "\n")
	} else {
		note := y.Note
		if note == "" {
			note = "Годовая доходность пока недоступна"
		}
		b.WriteString("📐 <i>" + htmlEscape(note) + "</i>\n")
	}

	// Footer
	b.WriteString("──────────────────────\n")
	b.WriteString("<i>📌 Данные актуальны на " + time.Now().Format("02.01.2006 15:04") + "</i>")

	return b.String()
}

// htmlEscape escapes HTML special characters.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// formatPercent formats a value already in percent units (e.g. 12.3 → "12.3%").
func formatPercent(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64) + "%"
}

// formatSignedMoney formats a signed amount rounded to whole rubles, with an explicit
// +/− sign and the ₽ symbol (e.g. 12858.7 → "+12 859 ₽", -500.4 → "−500 ₽").
// Used for the approximate (≈) figures shown next to the percentages.
func formatSignedMoney(value float64) string {
	sign := "+"
	if value < 0 {
		sign = "−" // U+2212 minus sign
	}
	rounded := strconv.FormatFloat(math.Round(math.Abs(value)), 'f', 0, 64)
	return sign + groupThousands(rounded) + " ₽"
}

// groupThousands inserts a space every three digits of an unsigned integer string.
func groupThousands(intPart string) string {
	if len(intPart) <= 3 {
		return intPart
	}
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteString(" ")
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// formatMoney formats a number with thousands separators and one decimal place.
func formatMoney(value float64) string {
	str := strconv.FormatFloat(value, 'f', 1, 64)
	parts := strings.Split(str, ".")
	intPart := parts[0]
	if len(intPart) > 3 {
		var result strings.Builder
		for i, ch := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result.WriteString(" ")
			}
			result.WriteRune(ch)
		}
		str = result.String() + "." + parts[1]
	}
	return str
}
