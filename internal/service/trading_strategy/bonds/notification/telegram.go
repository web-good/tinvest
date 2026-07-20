package notification

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"tinvest/internal/domain"
)

func Send(bonds []domain.BondReport, dateFrom time.Time, dateTo time.Time) string {
	if len(bonds) == 0 {
		return ""
	}

	var notifyMessageBuilder strings.Builder

	// Заголовок сообщения
	fmt.Fprintf(&notifyMessageBuilder, "<b>📊 Облигации </b>\n\n"+
		"🗓️ <i>Период: %s - %s</i>\n\n",
		dateFrom.Format("02.01.2006"),
		dateTo.Format("02.01.2006"))

	// Тип облигаций
	bondType := strings.ToUpper(string(bonds[0].Type))
	fmt.Fprintf(&notifyMessageBuilder, "🏛️ <b>%s</b>\n\n", bondType)

	// Строка политики отбора облигаций
	notifyMessageBuilder.WriteString("🛡️ <i>Отбор: только LOW risk, несубординированные, ликвидные</i>\n\n")

	// Список облигаций
	for i, bond := range bonds {
		// Разделитель между облигациями
		if i > 0 {
			notifyMessageBuilder.WriteString("\n" +
				"──────────────────────\n\n")
		}

		// Название облигации
		fmt.Fprintf(&notifyMessageBuilder, "<b>%d. %s</b>\n\n",
			i+1,
			htmlEscape(bond.Name))

		// Ключевые метрики в компактном виде
		notifyMessageBuilder.WriteString(
			"💰 <b>Доходность к погашению (YTM):</b> " + formatPercent(bond.PercentByYear) + "\n" +
				"🎯 <b>Купонная доходность в год:</b> " + formatPercent(bond.CouponPercentByYear) + "\n" +
				"📈 <b>Прибыль/год:</b> " + formatMoney(bond.ManyByYear) + "₽\n" +
				"💳 <b>НКД:</b> " + formatMoney(bond.Nkd) + "₽\n" +
				"🏢 <b>Сектор:</b> " + sectorOrDash(bond.Sector) + "\n" +
				"⏰ <b>Погашение:</b> " + bond.ExecutionDate.Format("02.01.2006") + "\n")
	}

	// Подвал сообщения
	notifyMessageBuilder.WriteString(
		"\n──────────────────────\n" +
			"<i>📌 Данные актуальны на " + time.Now().Format("02.01.2006 15:04") + "</i>",
	)

	return notifyMessageBuilder.String()
}

// sectorOrDash возвращает экранированный сектор, либо прочерк, если сектор не задан
func sectorOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return htmlEscape(s)
}

// htmlEscape экранирует HTML-символы для безопасности
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// formatPercent форматирует процент с правильным знаком
func formatPercent(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64) + "%"
}

// formatMoney форматирует денежную сумму с разделителями
func formatMoney(value float64) string {
	// Для больших чисел добавляем разделитель тысяч
	str := strconv.FormatFloat(value, 'f', 1, 64)
	if len(str) > 6 {
		// Простое форматирование для тысяч
		parts := strings.Split(str, ".")
		intPart := parts[0]
		if len(intPart) > 3 {
			var result strings.Builder
			for i, char := range intPart {
				if i > 0 && (len(intPart)-i)%3 == 0 {
					result.WriteString(" ")
				}
				result.WriteRune(char)
			}
			str = result.String() + "." + parts[1]
		}
	}
	return str
}
