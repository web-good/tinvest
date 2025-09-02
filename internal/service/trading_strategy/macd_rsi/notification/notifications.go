package notification

import (
	"strings"
	"tinvest/internal/domain"
)

func Trade(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>RSI MACD NEW:</b></u>\n\n\n<code>")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("<u><b>" + log.InstrumentName + "</b></u>")
		notifyMessageBuilder.WriteString("\n" + log.ATR.ToString() + "\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
