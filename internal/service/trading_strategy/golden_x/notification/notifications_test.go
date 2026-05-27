package notification

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/golden_x/model"
)

// bodyWithoutLegend strips the static legend block so negative assertions
// (expecting an emoji to be absent) only inspect the actual share rows.
func bodyWithoutLegend(s string) string {
	return strings.Replace(s, legendBlock, "", 1)
}

func TestTrade_RendersLegendBlock(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-leg": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

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
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"green-id": {
				InstrumentName: "Yandex",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				TrendStatus:    model.TrendWith,
			},
			"yellow-id": {
				InstrumentName: "Polyus",
				RSI:            28,
				BuyTier:        model.TierYellow,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				TrendStatus:    model.TrendAgainst,
			},
			"notrend-id": {
				InstrumentName: "Sber",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				TrendStatus:    model.TrendUnknown,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindGrowth,
	}

	got := Trade(r)

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
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"any-id": {
				InstrumentName: "Lukoil",
				RSI:            28,
				BuyTier:        model.TierNone,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

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
	r := model.TradeResult{
		BuyShares: make(map[string]model.ShareResult),
		SellShares: map[string]model.ShareResult{
			"yellow-sell": {
				InstrumentName: "Lukoil",
				RSI:            65,
				SellTier:       model.TierSellYellow,
				SellThresholds: model.SellThresholds{P80: 60, P90: 70, P95: 80},
			},
			"orange-sell": {
				InstrumentName: "Gazprom",
				RSI:            75,
				SellTier:       model.TierSellOrange,
				SellThresholds: model.SellThresholds{P80: 60, P90: 70, P95: 80},
			},
			"red-sell": {
				InstrumentName: "Phosagro",
				RSI:            85,
				SellTier:       model.TierSellRed,
				SellThresholds: model.SellThresholds{P80: 60, P90: 70, P95: 80},
			},
		},
		Kind: model.StrategyKindDividend,
	}

	got := Trade(r)

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
		t.Errorf("expected no buy-section header when buyShares is empty, got:\n%s", got)
	}
}

func TestTrade_RendersSellSectionGrowth(t *testing.T) {
	r := model.TradeResult{
		BuyShares: make(map[string]model.ShareResult),
		SellShares: map[string]model.ShareResult{
			"growth-exit": {
				InstrumentName: "Yandex",
				RSI:            80,
				SellTier:       model.TierSellOrange,
				SellThresholds: model.SellThresholds{P80: 60, P90: 70, P95: 80},
			},
		},
		Kind: model.StrategyKindGrowth,
	}

	got := Trade(r)

	if !strings.Contains(got, "Yandex 🔴") {
		t.Errorf("expected Growth sell tier 🔴 for Yandex, got:\n%s", got)
	}
	if strings.Contains(got, "Сигналы на покупку") {
		t.Errorf("expected no buy-section header when buyShares is empty, got:\n%s", got)
	}
}

func TestTrade_BuyAndSellTogether(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"buy-id": {
				InstrumentName: "Sber",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
			},
		},
		SellShares: map[string]model.ShareResult{
			"sell-id": {
				InstrumentName: "Phosagro",
				RSI:            85,
				SellTier:       model.TierSellRed,
				SellThresholds: model.SellThresholds{P80: 60, P90: 70, P95: 80},
			},
		},
		Kind: model.StrategyKindDividend,
	}

	got := Trade(r)

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
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-div": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				DivergenceOK:   true,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	if !strings.Contains(got, "Lukoil 🟢 📈") {
		t.Errorf("expected 'Lukoil 🟢 📈' (tier + badge), got:\n%s", got)
	}
}

func TestTrade_NoBadgeWhenNotDivergent(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-plain": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	if strings.Contains(bodyWithoutLegend(got), "📈") {
		t.Errorf("expected no divergence badge, got:\n%s", got)
	}
}

func TestTrade_BadgeAfterTrendMark(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-both": {
				InstrumentName: "Yandex",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				TrendStatus:    model.TrendWith,
				DivergenceOK:   true,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindGrowth,
	}

	got := Trade(r)

	// Expect order: name, tier emoji, trend mark, divergence badge.
	if !strings.Contains(got, "Yandex 🟢 ✅ 📈") {
		t.Errorf("expected 'Yandex 🟢 ✅ 📈' (ordered), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeAppendedAfterDivergence(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-both": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				DivergenceOK:   true,
				VolumeOK:       true,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	// Expect order: name, tier emoji, divergence badge, volume badge.
	if !strings.Contains(got, "Lukoil 🟢 📈 🔊") {
		t.Errorf("expected 'Lukoil 🟢 📈 🔊' (ordered), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeAbsentWhenNotConfirmed(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-no-vol": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				DivergenceOK:   true,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	if strings.Contains(bodyWithoutLegend(got), "🔊") {
		t.Errorf("expected no volume badge, got:\n%s", got)
	}
	if !strings.Contains(got, "Lukoil 🟢 📈") {
		t.Errorf("expected 'Lukoil 🟢 📈' (divergence kept), got:\n%s", got)
	}
}

func TestTrade_VolumeBadgeWithoutDivergence(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"share-vol-only": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				VolumeOK:       true,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	if !strings.Contains(got, "Lukoil 🟢 🔊") {
		t.Errorf("expected 'Lukoil 🟢 🔊' (volume only, no divergence), got:\n%s", got)
	}
	if strings.Contains(bodyWithoutLegend(got), "📈") {
		t.Errorf("expected no divergence badge, got:\n%s", got)
	}
}
