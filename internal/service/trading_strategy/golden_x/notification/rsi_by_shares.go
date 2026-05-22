package notification

import (
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func RSIList(
	info *domain.Info,
	kind dto.StrategyKind,
	trends map[string]dto.TrendStatus,
	thresholds map[string]dto.Thresholds,
	sellThresholds map[string]dto.SellThresholds,
	divergences map[string]bool,
	volumesConfirmed map[string]bool,
	stops map[string]dto.Stop,
) string {
	b := strings.Builder{}
	if medal := kind.Medal(); medal != "" {
		b.WriteString(medal + "\n\n")
	}
	b.WriteString(legendBlock)
	b.WriteString("🧾\n<u><b>Промежуточные значения RSI и % цены к див доход:</b></u>\n\n\n<code>")

	for id, log := range info.Items() {
		trendMark := trends[id].Mark()
		if trendMark != "" {
			trendMark = " " + trendMark
		}

		tier := tierEmoji(log.RSIValue, thresholds[id])
		suffix := thresholdSuffix(thresholds[id])
		if tier == "" {
			if sellTier := sellTierEmoji(log.RSIValue, sellThresholds[id]); sellTier != "" {
				tier = sellTier
				suffix = sellThresholdSuffix(sellThresholds[id])
			}
		}

		b.WriteString("• <b>Акция:</b> " + log.InstrumentName + tier + trendMark + divergenceBadge(divergences[id]) + volumeBadge(volumesConfirmed[id]) + "\n")
		b.WriteString("  <b>RSI Length:</b>" + strconv.Itoa(log.RSILength) + "\n")
		b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + suffix + "\n")
		b.WriteString(stopLine(stops[id]))
		b.WriteString("\n")
	}

	b.WriteString("</code>")
	return b.String()
}
