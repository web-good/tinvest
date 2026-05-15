package notification

import (
	"strconv"
	"strings"
	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func Trade(info *domain.Info, kind dto.StrategyKind) string {
	notifyMessageBuilder := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		notifyMessageBuilder.WriteString(medal + "\n\n")
	}

	notifyMessageBuilder.WriteString("<u><b>Акции находящиеся в локальных минимумах:</b></u>\n\n\n<code>")
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
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
