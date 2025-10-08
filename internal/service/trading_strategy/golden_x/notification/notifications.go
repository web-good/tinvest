package notification

import (
	"fmt"
	"strconv"
	"strings"
	"tinvest/internal/domain"
)

func Trade(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>Акции которые просели до минимума RSI:</b></u>\n\n\n<code>")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Length:</b> " + strconv.Itoa(log.RSIValue) + "\n")
		notifyMessageBuilder.WriteString("  <b>Див.доход≈</b>" + fmt.Sprintf("%.2f", log.ProcentPrice) + "%\n\n")

		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
