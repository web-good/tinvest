package notification

import (
	"fmt"
	"strconv"
	"strings"
	"tinvest/internal/domain"
)

func RSIList(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🧾\n<u><b>Промежуточные значения RSI и % цены к див доход:</b></u>\n\n\n<code>")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Length:</b>" + strconv.Itoa(log.RSILength) + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + "\n")
		notifyMessageBuilder.WriteString("  <b>Див.дох./к тек.цене≈</b>" + fmt.Sprintf("%.2f", log.ProcentPrice) + "%\n\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
