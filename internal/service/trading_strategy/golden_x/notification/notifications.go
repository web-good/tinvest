package notification

import (
	"fmt"
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func Trade(
	info *domain.Info,
	kind dto.StrategyKind,
	trends map[string]dto.TrendStatus,
	thresholds map[string]dto.Thresholds,
) string {
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
		notifyMessageBuilder.WriteString("• <b>Акция:</b> " + log.InstrumentName + tierEmoji(log.RSIValue, thresholds[id]) + trendMark + "\n")
		notifyMessageBuilder.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + thresholdSuffix(thresholds[id]) + "\n")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}

// tierEmoji renders the colored circle based on adaptive thresholds. Empty
// Thresholds (zero value, e.g. for shares filtered out before we could compute
// them) renders no emoji — the row still appears with the raw RSI value.
func tierEmoji(rsi float64, th dto.Thresholds) string {
	if th.P5 == 0 && th.P15 == 0 {
		return ""
	}
	switch {
	case rsi < th.P5:
		return " 🟢"
	case rsi < th.P15:
		return " 🟡"
	default:
		return ""
	}
}

// thresholdSuffix renders the percentile annotation appended to the RSI line,
// e.g. "  (p5=24, p15=31)". Empty Thresholds renders nothing.
func thresholdSuffix(th dto.Thresholds) string {
	if th.P5 == 0 && th.P15 == 0 {
		return ""
	}
	return fmt.Sprintf("  (p5=%.1f, p15=%.1f)", th.P5, th.P15)
}
