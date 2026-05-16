package notification

import (
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func Trade(info *domain.Info, kind dto.StrategyKind, trends map[string]dto.TrendStatus) string {
	notifyMessageBuilder := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		notifyMessageBuilder.WriteString(medal + "\n\n")
	}

	notifyMessageBuilder.WriteString("<u><b>Акции находящиеся в локальных минимумах:</b></u>\n\n\n<code>")
	for id, log := range info.Items() {
		trendMark := trends[id].Mark()
		if trendMark != "" {
			trendMark = " " + trendMark
		}

		var rsiColor string
		switch {
		case log.RSIValue <= 40 && log.RSIValue >= 35:
			rsiColor = " 🟤"
		case log.RSIValue >= 31 && log.RSIValue < 35:
			rsiColor = " 🟡"
		case log.RSIValue < 31:
			rsiColor = " 🟢"
		}

		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + rsiColor + trendMark + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>")
		notifyMessageBuilder.WriteString(strconv.Itoa(int(log.RSIValue)) + "\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
