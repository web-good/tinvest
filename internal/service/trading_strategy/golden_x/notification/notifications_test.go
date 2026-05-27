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

func TestTrade_RendersScore(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"score-id": {
				InstrumentName: "Lukoil",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				Score:          7,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	if !strings.Contains(got, "Score:</b> 7") {
		t.Errorf("expected Score: 7 in output, got:\n%s", got)
	}
}

func TestTrade_SortsByScoreDesc(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"a-low": {
				InstrumentName: "LowScore",
				RSI:            20,
				BuyTier:        model.TierGreen,
				Score:          3,
			},
			"b-high": {
				InstrumentName: "HighScore",
				RSI:            18,
				BuyTier:        model.TierGreen,
				Score:          7,
			},
			"c-mid": {
				InstrumentName: "MidScore",
				RSI:            22,
				BuyTier:        model.TierYellow,
				Score:          5,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindGrowth,
	}

	got := Trade(r)

	highIdx := strings.Index(got, "HighScore")
	midIdx := strings.Index(got, "MidScore")
	lowIdx := strings.Index(got, "LowScore")

	if highIdx < 0 || midIdx < 0 || lowIdx < 0 {
		t.Fatalf("missing share names in output:\n%s", got)
	}
	if !(highIdx < midIdx && midIdx < lowIdx) {
		t.Errorf("expected HighScore(%d) < MidScore(%d) < LowScore(%d) in output:\n%s", highIdx, midIdx, lowIdx, got)
	}
}

func TestTrade_RendersCappedSection(t *testing.T) {
	r := model.TradeResult{
		BuyShares:  make(map[string]model.ShareResult),
		SellShares: make(map[string]model.ShareResult),
		CappedBuyShares: map[string]model.ShareResult{
			"oil-1": {
				InstrumentName: "Lukoil",
				RSI:            22,
				BuyTier:        model.TierGreen,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				Sector:         "Нефть",
				Score:          5,
			},
			"oil-2": {
				InstrumentName: "Rosneft",
				RSI:            25,
				BuyTier:        model.TierYellow,
				Thresholds:     model.Thresholds{P5: 24, P15: 31},
				Sector:         "Нефть",
				Score:          3,
			},
		},
		Kind: model.StrategyKindDividend,
	}

	got := Trade(r)

	if !strings.Contains(got, "⏸️") {
		t.Errorf("expected capped section emoji, got:\n%s", got)
	}
	if !strings.Contains(got, "Лимит сектора «Нефть»") {
		t.Errorf("expected sector name in capped section, got:\n%s", got)
	}
	if !strings.Contains(got, "Lukoil") || !strings.Contains(got, "Rosneft") {
		t.Errorf("expected both capped shares in output, got:\n%s", got)
	}
	// Lukoil (score 5) should appear before Rosneft (score 3)
	lukoilIdx := strings.Index(got, "Lukoil")
	rosneftIdx := strings.Index(got, "Rosneft")
	if lukoilIdx > rosneftIdx {
		t.Errorf("expected Lukoil (score 5) before Rosneft (score 3), got:\n%s", got)
	}
}

func TestTrade_NoCappedSectionWhenEmpty(t *testing.T) {
	r := model.TradeResult{
		BuyShares: map[string]model.ShareResult{
			"buy-id": {
				InstrumentName: "Sber",
				RSI:            20,
				BuyTier:        model.TierGreen,
			},
		},
		SellShares: make(map[string]model.ShareResult),
		Kind:       model.StrategyKindDividend,
	}

	got := Trade(r)

	body := bodyWithoutLegend(got)
	if strings.Contains(body, "⏸️") {
		t.Errorf("expected no capped section when CappedBuyShares is empty, got:\n%s", got)
	}
}
