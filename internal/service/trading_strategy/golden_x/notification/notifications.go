package notification

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tinvest/internal/service/trading_strategy/golden_x/model"
)

type shareEntry struct {
	id string
	sr model.ShareResult
}

func sortedByScore(m map[string]model.ShareResult) []shareEntry {
	entries := make([]shareEntry, 0, len(m))
	for id, sr := range m {
		entries = append(entries, shareEntry{id: id, sr: sr})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].sr.Score != entries[j].sr.Score {
			return entries[i].sr.Score > entries[j].sr.Score
		}
		return entries[i].id < entries[j].id
	})
	return entries
}

func Trade(r model.TradeResult) string {
	b := strings.Builder{}
	if medal := r.Kind.Medal(); medal != "" {
		b.WriteString(medal + "\n\n")
	}
	b.WriteString(legendBlock)

	if len(r.BuyShares) > 0 {
		b.WriteString("<u><b>Сигналы на покупку:</b></u>\n\n\n<code>")
		for _, e := range sortedByScore(r.BuyShares) {
			sr := e.sr
			trendMark := sr.TrendStatus.Mark()
			if trendMark != "" {
				trendMark = " " + trendMark
			}
			b.WriteString("• <b>Акция:</b> " + sr.InstrumentName + buyTierEmoji(sr.BuyTier) + trendMark + divergenceBadge(sr.DivergenceOK) + volumeBadge(sr.VolumeOK) + fundamentalBadge(sr.FundamentalBonus) + "\n")
			b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(sr.RSI)) + thresholdSuffix(sr.Thresholds) + "\n")
			if sr.Score > 0 {
				b.WriteString("  <b>Score:</b> " + strconv.Itoa(sr.Score) + "\n")
			}
			b.WriteString("\n")
		}
		b.WriteString("</code>\n\n")
	}

	if len(r.CappedBuyShares) > 0 {
		sectors := make(map[string][]shareEntry)
		for id, sr := range r.CappedBuyShares {
			sectors[sr.Sector] = append(sectors[sr.Sector], shareEntry{id: id, sr: sr})
		}
		sectorNames := make([]string, 0, len(sectors))
		for name := range sectors {
			sectorNames = append(sectorNames, name)
		}
		sort.Strings(sectorNames)

		for _, sectorName := range sectorNames {
			entries := sectors[sectorName]
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].sr.Score != entries[j].sr.Score {
					return entries[i].sr.Score > entries[j].sr.Score
				}
				return entries[i].id < entries[j].id
			})
			b.WriteString("⏸️ <b>Лимит сектора «" + sectorName + "»:</b>\n<code>")
			for _, e := range entries {
				sr := e.sr
				b.WriteString("• <b>Акция:</b> " + sr.InstrumentName + buyTierEmoji(sr.BuyTier) + "\n")
				b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(sr.RSI)) + thresholdSuffix(sr.Thresholds) + "\n")
				b.WriteString("\n")
			}
			b.WriteString("</code>\n\n")
		}
	}

	if len(r.SellShares) > 0 {
		b.WriteString("<u><b>Сигналы на продажу:</b></u>\n\n\n<code>")
		for _, e := range sortedByScore(r.SellShares) {
			sr := e.sr
			b.WriteString("• <b>Акция:</b> " + sr.InstrumentName + sellTierEmoji(sr.SellTier) + "\n")
			b.WriteString("  <b>RSI Value:</b>" + strconv.Itoa(int(sr.RSI)) + sellThresholdSuffix(sr.SellThresholds) + "\n")
			if sr.Score > 0 {
				b.WriteString("  <b>Score:</b> " + strconv.Itoa(sr.Score) + "\n")
			}
			b.WriteString("\n")
		}
		b.WriteString("</code>")
	}

	return b.String()
}

const legendBlock = "<b>Легенда:</b>\n" +
	"🟢 сильно перепродан\n" +
	"🟡 перепродан\n" +
	"🟠 перекуплен\n" +
	"🔴 сильно перекуплен\n" +
	"🚨 экстремум сверху\n" +
	"✅ тренд за нас\n" +
	"🚫 тренд против\n" +
	"📈 бычья дивергенция\n" +
	"🔊 подтверждение объёмом\n" +
	"🏆 фунд-рейтинг (+1..+3 к Score)\n" +
	"⏸️ лимит сектора\n\n"

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

// fundamentalBadge returns " 🏆" when the share's dividend-screener percentile
// rank contributed a positive bonus to its Score, "" otherwise. The badge is
// purely additive.
func fundamentalBadge(bonus int) string {
	if bonus > 0 {
		return " 🏆"
	}
	return ""
}

// buyTierEmoji maps a pre-computed buy AlertTier to the corresponding colored
// circle emoji. TierNone (and any unexpected value) renders no emoji.
func buyTierEmoji(tier model.AlertTier) string {
	switch tier {
	case model.TierGreen:
		return " 🟢"
	case model.TierYellow:
		return " 🟡"
	default:
		return ""
	}
}

// sellTierEmoji maps a pre-computed sell AlertTier to the corresponding colored
// circle emoji. TierNone (and any unexpected value) renders no emoji.
func sellTierEmoji(tier model.AlertTier) string {
	switch tier {
	case model.TierSellRed:
		return " 🚨"
	case model.TierSellOrange:
		return " 🔴"
	case model.TierSellYellow:
		return " 🟠"
	default:
		return ""
	}
}

// thresholdSuffix renders the buy percentile annotation appended to the RSI
// line, e.g. "  (p5=24.0, p15=31.0)". Empty Thresholds renders nothing.
func thresholdSuffix(th model.Thresholds) string {
	if th.P5 == 0 && th.P15 == 0 {
		return ""
	}
	return fmt.Sprintf("  (p5=%.1f, p15=%.1f)", th.P5, th.P15)
}

// sellThresholdSuffix renders the sell percentile annotation appended to the
// RSI line, e.g. "  (p80=60.0, p90=70.0, p95=80.0)". Empty SellThresholds
// renders nothing.
func sellThresholdSuffix(st model.SellThresholds) string {
	if st.P80 == 0 && st.P90 == 0 && st.P95 == 0 {
		return ""
	}
	return fmt.Sprintf("  (p80=%.1f, p90=%.1f, p95=%.1f)", st.P80, st.P90, st.P95)
}
