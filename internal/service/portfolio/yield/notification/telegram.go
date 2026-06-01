package notification

import (
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
	b.WriteString("➖ <b>Выводы:</b> " + formatMoney(y.Withdrawals) + " ₽\n\n")

	// Returns
	b.WriteString("📊 <b>Доходность за период:</b> " + formatPercent(y.PeriodReturn*100) + "\n")

	if y.XIRRAvailable {
		b.WriteString("📐 <b>Годовая (XIRR):</b> " + formatPercent(y.AnnualizedXIRR*100) + "\n")
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
