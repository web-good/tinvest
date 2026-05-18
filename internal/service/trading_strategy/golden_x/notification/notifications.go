package notification

import (
	"fmt"
	"strconv"
	"strings"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func Trade(
	buyInfo *domain.Info,
	sellInfo *domain.Info,
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

	if buyInfo != nil && len(buyInfo.Items()) > 0 {
		b.WriteString("<u><b>Акции находящиеся в локальных минимумах:</b></u>\n\n\n<code>")
		for id, log := range buyInfo.Items() {
			trendMark := trends[id].Mark()
			if trendMark != "" {
				trendMark = " " + trendMark
			}
			b.WriteString("• <b>Акция:</b> " + log.InstrumentName + tierEmoji(log.RSIValue, thresholds[id]) + trendMark + divergenceBadge(divergences[id]) + volumeBadge(volumesConfirmed[id]) + "\n")
			b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + thresholdSuffix(thresholds[id]) + "\n")
			b.WriteString(stopLine(stops[id]))
			b.WriteString("\n")
		}
		b.WriteString("</code>\n\n")
	}

	if sellInfo != nil && len(sellInfo.Items()) > 0 {
		b.WriteString("<u><b>Акции находящиеся в локальных максимумах:</b></u>\n\n\n<code>")
		for id, log := range sellInfo.Items() {
			b.WriteString("• <b>Акция:</b> " + log.InstrumentName + sellTierEmoji(log.RSIValue, sellThresholds[id]) + "\n")
			b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(log.RSIValue)) + sellThresholdSuffix(sellThresholds[id]) + "\n")
			b.WriteString("\n")
		}
		b.WriteString("</code>")
	}

	return b.String()
}

// divergenceBadge returns " 📈" when the share's row should display the
// bullish RSI divergence annotation, "" otherwise. Empty input map renders
// nothing (the badge is purely additive).
func divergenceBadge(divergent bool) string {
	if divergent {
		return " 📈"
	}
	return ""
}

// volumeBadge returns " 🔊" when the share's row should display the volume
// confirmation annotation, "" otherwise. The badge is purely additive — it
// never participates in dedup and never replaces an existing emoji.
func volumeBadge(confirmed bool) string {
	if confirmed {
		return " 🔊"
	}
	return ""
}

// stopLine renders the ATR-derived stop suggestion as its own line:
// "  <b>Stop:</b> <price> (−<pct>%)\n". The zero-value dto.Stop{} renders
// "" — same convention as Thresholds and SellThresholds, keeps the indicator
// additive and silent on insufficient history.
func stopLine(s dto.Stop) string {
	if s.Price == 0 && s.DistancePct == 0 {
		return ""
	}
	return fmt.Sprintf("  <b>Stop:</b> %.2f (−%.1f%%)\n", s.Price, s.DistancePct)
}

// tierEmoji renders the colored circle based on adaptive buy thresholds. Empty
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

// thresholdSuffix renders the buy percentile annotation appended to the RSI
// line, e.g. "  (p5=24.0, p15=31.0)". Empty Thresholds renders nothing.
func thresholdSuffix(th dto.Thresholds) string {
	if th.P5 == 0 && th.P15 == 0 {
		return ""
	}
	return fmt.Sprintf("  (p5=%.1f, p15=%.1f)", th.P5, th.P15)
}

// sellTierEmoji renders the colored circle for a sell-side row using the
// share's own adaptive upper percentiles. Strict `>` comparisons mirror
// sellTierFromAdaptive. Empty SellThresholds renders no emoji.
func sellTierEmoji(rsi float64, st dto.SellThresholds) string {
	if st.P80 == 0 && st.P90 == 0 && st.P95 == 0 {
		return ""
	}
	switch {
	case rsi > st.P95:
		return " 🚨"
	case rsi > st.P90:
		return " 🔴"
	case rsi > st.P80:
		return " 🟠"
	default:
		return ""
	}
}

// sellThresholdSuffix renders the sell percentile annotation appended to the
// RSI line, e.g. "  (p80=60.0, p90=70.0, p95=80.0)". Empty SellThresholds
// renders nothing.
func sellThresholdSuffix(st dto.SellThresholds) string {
	if st.P80 == 0 && st.P90 == 0 && st.P95 == 0 {
		return ""
	}
	return fmt.Sprintf("  (p80=%.1f, p90=%.1f, p95=%.1f)", st.P80, st.P90, st.P95)
}
