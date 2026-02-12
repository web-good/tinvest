package notification

import (
	"strconv"
	"strings"
	"time"
	"tinvest/internal/domain"
)

func Send(bonds []domain.BondReport, dateFrom time.Time, dateTo time.Time) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("<u><b>🏛Облигации - с датой погашения ")
	notifyMessageBuilder.WriteString("от ")
	notifyMessageBuilder.WriteString(strconv.Itoa(dateFrom.Year()))
	notifyMessageBuilder.WriteString(" - до ")
	notifyMessageBuilder.WriteString(strconv.Itoa(dateTo.Year()))
	notifyMessageBuilder.WriteString(" года:</b></u>\n\n\n<code>")
	notifyMessageBuilder.WriteString("🔶 " + string(bonds[0].Type) + "\n\n")

	for _, bond := range bonds {
		notifyMessageBuilder.WriteString("• <b>Название:</b><u>" + bond.Name + "</u>\n")
		notifyMessageBuilder.WriteString("- <b>Средне годовая доходность к погашению с учетом налога:</b>" +
			strconv.FormatFloat(bond.PercentByYear, 'f', 1, 64) +
			"% (" + strconv.FormatFloat(bond.ManyByYear, 'f', 1, 64) + "₽)\n")
		notifyMessageBuilder.WriteString("- <b>НКД:</b>" + strconv.FormatFloat(bond.Nkd, 'f', 1, 64) + "₽\n")
		notifyMessageBuilder.WriteString("- <b>Доходность к погашению с учетом налога:</b>" +
			strconv.FormatFloat(bond.FinalSum, 'f', 1, 64) + "₽\n")
		notifyMessageBuilder.WriteString("- <b>Купонная доходность с учетом налога:</b>" +
			strconv.FormatFloat(bond.CouponPercentByYear, 'f', 1, 64) + "%\n")
		notifyMessageBuilder.WriteString("- <b>Дата погащения:</b>" + bond.ExecutionDate.Format(time.DateOnly) + "\n")
		notifyMessageBuilder.WriteString("====================\n\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
