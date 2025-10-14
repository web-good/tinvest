package notification

import (
	"fmt"
	"strings"
	"tinvest/internal/domain"
)

func Trade(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>Акции с просевшим RSI:</b></u>\n\n\n<code>")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + "\n")
		notifyMessageBuilder.WriteString("  <b>Див.дох./к тек.цене≈</b>" + fmt.Sprintf("%.2f", log.ProcentPrice) + "%\n\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
