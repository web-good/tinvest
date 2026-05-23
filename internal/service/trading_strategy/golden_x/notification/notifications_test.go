package notification

import (
	"strings"
	"testing"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

// bodyWithoutLegend strips the static legend block so negative assertions
// (expecting an emoji to be absent) only inspect the actual share rows.
func bodyWithoutLegend(s string) string {
	return strings.Replace(s, legendBlock, "", 1)
}

func TestTrade_RendersLegendBlock(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-leg", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})
	thresholds := map[string]dto.Thresholds{
		"share-leg": {P5: 24, P15: 31},
	}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, nil)

	if !strings.Contains(got, legendBlock) {
		t.Errorf("expected legend block in output, got:\n%s", got)
	}
	// Legend must sit between the medal and the buy-section header.
	medalIdx := strings.Index(got, "🥇")
	legendIdx := strings.Index(got, "<b>Легенда:</b>")
	buyHeaderIdx := strings.Index(got, "Сигналы на покупку")
	if medalIdx < 0 || legendIdx < 0 || buyHeaderIdx < 0 {
		t.Fatalf("missing landmarks: medal=%d legend=%d header=%d in:\n%s", medalIdx, legendIdx, buyHeaderIdx, got)
	}
	if !(medalIdx < legendIdx && legendIdx < buyHeaderIdx) {
		t.Errorf("expected order medal < legend < header, got medal=%d legend=%d header=%d", medalIdx, legendIdx, buyHeaderIdx)
	}
}

func TestTrade_AdaptiveTiersAndThresholdSuffix(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("green-id", domain.Item{InstrumentName: "Yandex", RSIValue: 20})
	info.WriteToMap("yellow-id", domain.Item{InstrumentName: "Polyus", RSIValue: 28})
	info.WriteToMap("notrend-id", domain.Item{InstrumentName: "Sber", RSIValue: 20})

	trends := map[string]dto.TrendStatus{
		"green-id":   dto.TrendWith,
		"yellow-id":  dto.TrendAgainst,
		"notrend-id": dto.TrendUnknown,
	}
	thresholds := map[string]dto.Thresholds{
		"green-id":   {P5: 24, P15: 31},
		"yellow-id":  {P5: 24, P15: 31},
		"notrend-id": {P5: 24, P15: 31},
	}

	got := Trade(info, nil, dto.StrategyKindGrowth, trends, thresholds, nil, nil, nil, nil)

	if !strings.Contains(got, "Yandex 🟢 ✅") {
		t.Errorf("expected 'Yandex 🟢 ✅', got:\n%s", got)
	}
	if !strings.Contains(got, "Polyus 🟡 🚫") {
		t.Errorf("expected 'Polyus 🟡 🚫', got:\n%s", got)
	}
	if !strings.Contains(got, "Sber 🟢\n") {
		t.Errorf("expected 'Sber 🟢' without trend mark, got:\n%s", got)
	}
	if !strings.Contains(got, "(p5=24.0, p15=31.0)") {
		t.Errorf("expected threshold suffix '(p5=24.0, p15=31.0)', got:\n%s", got)
	}
}

func TestTrade_NoThresholdsRendersNoEmojiOrSuffix(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("any-id", domain.Item{InstrumentName: "Lukoil", RSIValue: 28})

	got := Trade(info, nil, dto.StrategyKindDividend, nil, nil, nil, nil, nil, nil)

	if !strings.Contains(got, "🥇") {
		t.Errorf("expected gold medal, got:\n%s", got)
	}
	body := bodyWithoutLegend(got)
	if strings.Contains(body, "🟢") || strings.Contains(body, "🟡") {
		t.Errorf("expected no tier emoji when thresholds missing, got:\n%s", got)
	}
	if strings.Contains(got, "p5=") {
		t.Errorf("expected no threshold suffix, got:\n%s", got)
	}
}

func TestTrade_RendersSellSectionGold(t *testing.T) {
	buyInfo := domain.NewInfo()
	sellInfo := domain.NewInfo()
	sellInfo.WriteToMap("yellow-sell", domain.Item{InstrumentName: "Lukoil", RSIValue: 65})
	sellInfo.WriteToMap("orange-sell", domain.Item{InstrumentName: "Gazprom", RSIValue: 75})
	sellInfo.WriteToMap("red-sell", domain.Item{InstrumentName: "Phosagro", RSIValue: 85})

	sellThresholds := map[string]dto.SellThresholds{
		"yellow-sell": {P80: 60, P90: 70, P95: 80},
		"orange-sell": {P80: 60, P90: 70, P95: 80},
		"red-sell":    {P80: 60, P90: 70, P95: 80},
	}

	got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil, nil, nil)

	if !strings.Contains(got, "Сигналы на продажу") {
		t.Errorf("expected sell-section header, got:\n%s", got)
	}
	if !strings.Contains(got, "Lukoil 🟠") {
		t.Errorf("expected 'Lukoil 🟠', got:\n%s", got)
	}
	if !strings.Contains(got, "Gazprom 🔴") {
		t.Errorf("expected 'Gazprom 🔴', got:\n%s", got)
	}
	if !strings.Contains(got, "Phosagro 🚨") {
		t.Errorf("expected 'Phosagro 🚨', got:\n%s", got)
	}
	if !strings.Contains(got, "(p80=60.0, p90=70.0, p95=80.0)") {
		t.Errorf("expected sell threshold suffix, got:\n%s", got)
	}
	if strings.Contains(got, "Сигналы на покупку") {
		t.Errorf("expected no buy-section header when buyInfo is empty, got:\n%s", got)
	}
}

func TestTrade_RendersSellSectionGrowth(t *testing.T) {
	buyInfo := domain.NewInfo()
	sellInfo := domain.NewInfo()
	sellInfo.WriteToMap("growth-exit", domain.Item{InstrumentName: "Yandex", RSIValue: 80})

	sellThresholds := map[string]dto.SellThresholds{
		"growth-exit": {P80: 60, P90: 70, P95: 80},
	}

	got := Trade(buyInfo, sellInfo, dto.StrategyKindGrowth, nil, nil, sellThresholds, nil, nil, nil)

	if !strings.Contains(got, "Yandex 🔴") {
		t.Errorf("expected Growth sell tier 🔴 for Yandex, got:\n%s", got)
	}
	if strings.Contains(got, "Сигналы на покупку") {
		t.Errorf("expected no buy-section header when buyInfo is empty, got:\n%s", got)
	}
}

func TestTrade_BuyAndSellTogether(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("buy-id", domain.Item{InstrumentName: "Sber", RSIValue: 20})
	sellInfo := domain.NewInfo()
	sellInfo.WriteToMap("sell-id", domain.Item{InstrumentName: "Phosagro", RSIValue: 85})

	thresholds := map[string]dto.Thresholds{
		"buy-id": {P5: 24, P15: 31},
	}
	sellThresholds := map[string]dto.SellThresholds{
		"sell-id": {P80: 60, P90: 70, P95: 80},
	}

	got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, thresholds, sellThresholds, nil, nil, nil)

	if !strings.Contains(got, "Сигналы на покупку") {
		t.Errorf("buy section missing, got:\n%s", got)
	}
	if !strings.Contains(got, "Сигналы на продажу") {
		t.Errorf("sell section missing, got:\n%s", got)
	}
	if !strings.Contains(got, "Sber 🟢") {
		t.Errorf("expected 'Sber 🟢' in buy section, got:\n%s", got)
	}
	if !strings.Contains(got, "Phosagro 🚨") {
		t.Errorf("expected 'Phosagro 🚨' in sell section, got:\n%s", got)
	}
}

func TestTrade_RendersBullishDivergenceBadge(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-div", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-div": {P5: 24, P15: 31},
	}
	divergences := map[string]bool{"share-div": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil, nil)

	if !strings.Contains(got, "Lukoil 🟢 📈") {
		t.Errorf("expected 'Lukoil 🟢 📈' (tier + badge), got:\n%s", got)
	}
}

func TestTrade_NoBadgeWhenNotDivergent(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-plain", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-plain": {P5: 24, P15: 31},
	}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, nil)

	if strings.Contains(bodyWithoutLegend(got), "📈") {
		t.Errorf("expected no divergence badge, got:\n%s", got)
	}
}

func TestTrade_BadgeAfterTrendMark(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-both", domain.Item{InstrumentName: "Yandex", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-both": {P5: 24, P15: 31},
	}
	trends := map[string]dto.TrendStatus{
		"share-both": dto.TrendWith,
	}
	divergences := map[string]bool{"share-both": true}

	got := Trade(buyInfo, nil, dto.StrategyKindGrowth, trends, thresholds, nil, divergences, nil, nil)

	// Expect order: name, tier emoji, trend mark, divergence badge.
	if !strings.Contains(got, "Yandex 🟢 ✅ 📈") {
		t.Errorf("expected 'Yandex 🟢 ✅ 📈' (ordered), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeAppendedAfterDivergence(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-both", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-both": {P5: 24, P15: 31},
	}
	divergences := map[string]bool{"share-both": true}
	volumes := map[string]bool{"share-both": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, volumes, nil)

	// Expect order: name, tier emoji, divergence badge, volume badge.
	if !strings.Contains(got, "Lukoil 🟢 📈 🔊") {
		t.Errorf("expected 'Lukoil 🟢 📈 🔊' (ordered), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeAbsentWhenNotConfirmed(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-no-vol", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-no-vol": {P5: 24, P15: 31},
	}
	divergences := map[string]bool{"share-no-vol": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, divergences, nil, nil)

	if strings.Contains(bodyWithoutLegend(got), "🔊") {
		t.Errorf("expected no volume badge, got:\n%s", got)
	}
	if !strings.Contains(got, "Lukoil 🟢 📈") {
		t.Errorf("expected 'Lukoil 🟢 📈' (divergence kept), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeWithoutDivergence(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-vol-only", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-vol-only": {P5: 24, P15: 31},
	}
	volumes := map[string]bool{"share-vol-only": true}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, volumes, nil)

	if !strings.Contains(got, "Lukoil 🟢 🔊") {
		t.Errorf("expected 'Lukoil 🟢 🔊' (volume only, no divergence), got:\n%s", got)
	}
	if strings.Contains(bodyWithoutLegend(got), "📈") {
		t.Errorf("expected no divergence badge, got:\n%s", got)
	}
}

func TestTrade_StopLineRendersAfterRSI(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-stop", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-stop": {P5: 24, P15: 31},
	}
	stops := map[string]dto.Stop{
		"share-stop": {Price: 2410.5, DistancePct: 6.2},
	}

	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, stops)

	if !strings.Contains(got, "<b>Stop:</b> 2410.50 (−6.2%)") {
		t.Errorf("expected '<b>Stop:</b> 2410.50 (−6.2%%)', got:\n%s", got)
	}
	// Stop line must appear AFTER the RSI line of the same share.
	rsiIdx := strings.Index(got, "RSI Value:</b>20")
	stopIdx := strings.Index(got, "<b>Stop:</b>")
	if rsiIdx < 0 || stopIdx < 0 || stopIdx < rsiIdx {
		t.Errorf("expected Stop line after RSI line, got order RSI=%d Stop=%d in:\n%s", rsiIdx, stopIdx, got)
	}
}

func TestTrade_StopLineAbsentWhenEmpty(t *testing.T) {
	buyInfo := domain.NewInfo()
	buyInfo.WriteToMap("share-nostop", domain.Item{InstrumentName: "Lukoil", RSIValue: 20})

	thresholds := map[string]dto.Thresholds{
		"share-nostop": {P5: 24, P15: 31},
	}
	// Empty stops map.
	got := Trade(buyInfo, nil, dto.StrategyKindDividend, nil, thresholds, nil, nil, nil, map[string]dto.Stop{})

	if strings.Contains(got, "Stop:") {
		t.Errorf("expected no Stop line, got:\n%s", got)
	}
}

func TestTrade_StopLineAbsentOnSellRow(t *testing.T) {
	buyInfo := domain.NewInfo()
	sellInfo := domain.NewInfo()
	sellInfo.WriteToMap("sell-id", domain.Item{InstrumentName: "Phosagro", RSIValue: 85})

	sellThresholds := map[string]dto.SellThresholds{
		"sell-id": {P80: 60, P90: 70, P95: 80},
	}
	// Even if the map is wrongly populated for a sell-side ID, the renderer
	// must not surface a Stop line — stop is buy-only by construction.
	stops := map[string]dto.Stop{
		"sell-id": {Price: 1234.5, DistancePct: 9.9},
	}

	got := Trade(buyInfo, sellInfo, dto.StrategyKindDividend, nil, nil, sellThresholds, nil, nil, stops)

	if strings.Contains(got, "Stop:") {
		t.Errorf("expected no Stop line on sell row, got:\n%s", got)
	}
}
