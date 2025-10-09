package notification

import (
	"strconv"
	"strings"
	"tinvest/internal/domain"
)

func RSIList(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("\n<u><b>Акции и их текущий RSI:</b></u>\n\n\n<code>")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Length:</b> " + strconv.Itoa(log.RSIValue) + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b> " + strconv.Itoa(int(log.ProcentPrice)) + "\n\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
