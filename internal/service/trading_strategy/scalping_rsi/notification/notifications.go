package notification

import (
	"strings"
	"tinvest/internal/domain"
)

func Trade(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>RSI MINIMUM GOLD:</b></u>\n\n\n")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("<u>" + log.InstrumentName + "</u>")
		notifyMessageBuilder.WriteString("\n")
	}

	return notifyMessageBuilder.String()
}
