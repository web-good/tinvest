package notification

import (
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func RSIList(info *domain.Info, kind dto.StrategyKind, thresholds map[string]dto.Thresholds) string {
	notifyMessageBuilder := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		notifyMessageBuilder.WriteString(medal + "\n\n")
	}

	notifyMessageBuilder.WriteString("🧾\n<u><b>Промежуточные значения RSI и % цены к див доход:</b></u>\n\n\n<code>")

	for id, log := range info.Items() {
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + tierEmoji(log.RSIValue, thresholds[id]) + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Length:</b>" + strconv.Itoa(log.RSILength) + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + thresholdSuffix(thresholds[id]) + "\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
