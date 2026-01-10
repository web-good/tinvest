package notification

import (
	"strconv"
	"strings"
	"time"
	"tinvest/internal/domain"
)

func Send(bonds []domain.BondReport) string {
	notifyMessageBuilder := strings.Builder{}

	notifyMessageBuilder.WriteString("<u><b>🏛Облигации(AAA,фикс) - с наивысшей доходностью:</b></u>\n\n\n<code>")
	notifyMessageBuilder.WriteString("🔶 " + string(bonds[0].Type) + "\n\n")

	for _, bond := range bonds {
		notifyMessageBuilder.WriteString("• <b>Название:</b><u>" + bond.Name + "</u>\n")
		notifyMessageBuilder.WriteString("- <b>Процент дох./год с учетом налога:</b>" +
			strconv.FormatFloat(bond.PercentByYear, 'f', 1, 64) +
			"% (" + strconv.FormatFloat(bond.ManyByYear, 'f', 1, 64) + "₽)\n")
		notifyMessageBuilder.WriteString("- <b>НКД:</b>" + strconv.FormatFloat(bond.Nkd, 'f', 1, 64) + "₽\n")
		notifyMessageBuilder.WriteString("- <b>Дох. за все время с учетом налога:</b>" +
			strconv.FormatFloat(bond.FinalSum, 'f', 1, 64) + "₽\n")
		notifyMessageBuilder.WriteString("- <b>Дата погащения:</b>" + bond.ExecutionDate.Format(time.DateOnly) + "\n")
		notifyMessageBuilder.WriteString("====================\n\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
