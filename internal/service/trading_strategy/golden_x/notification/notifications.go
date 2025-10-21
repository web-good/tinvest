package notification

import (
	"fmt"
	"strconv"
	"strings"
	"tinvest/internal/domain"
)

func Trade(info *domain.Info, shareTip int) string {
	notifyMessageBuilder := strings.Builder{}
	if shareTip == 1 {
		notifyMessageBuilder.WriteString("🥇\n\n")
	}
	if shareTip == 2 {
		notifyMessageBuilder.WriteString("🥈\n\n")
	}

	notifyMessageBuilder.WriteString("<u><b>Акции находящтеся в локальных минимумах:</b></u>\n\n\n<code>")
	if shareTip == 1 {

	}
	for _, log := range info.Items() {
		if log.RSIValue <= 40 && log.RSIValue >= 35 {
			notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + " 🟤\n")
		} else if log.RSIValue >= 31 && log.RSIValue < 35 {
			notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + " 🟡\n")
		} else if log.RSIValue < 31 {
			notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + " 🟢\n")
		} else {
			notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + "\n")
		}

		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>")
		notifyMessageBuilder.WriteString(strconv.Itoa(int(log.RSIValue)) + "\n")
		notifyMessageBuilder.WriteString("  <b>Див.дох./к тек.цене≈</b>" + fmt.Sprintf("%.2f", log.ProcentPrice) + "%\n\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
