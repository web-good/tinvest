package notification

import (
	"fmt"
	"strings"
	"tinvest/internal/domain"
)

func Trade(info *domain.Info) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>RSI MINIMUM GOLD:</b></u>\n\n\n")

	for _, log := range info.Items() {
		notifyMessageBuilder.WriteString("<u>" + log.InstrumentName + "</u>")
		notifyMessageBuilder.WriteString("\n(процент цены относительно средних дивидентов <u>" + fmt.Sprintf("%.2f", log.ProcentPrice) + "</u>)\n\n")
		notifyMessageBuilder.WriteString("\n")
	}

	return notifyMessageBuilder.String()
}
