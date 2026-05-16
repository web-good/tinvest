package notification

import (
	"strings"
	"testing"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

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

	got := Trade(info, dto.StrategyKindGrowth, trends, thresholds)

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

	got := Trade(info, dto.StrategyKindDividend, nil, nil)

	if !strings.Contains(got, "🥇") {
		t.Errorf("expected gold medal, got:\n%s", got)
	}
	if strings.Contains(got, "🟢") || strings.Contains(got, "🟡") {
		t.Errorf("expected no tier emoji when thresholds missing, got:\n%s", got)
	}
	if strings.Contains(got, "p5=") {
		t.Errorf("expected no threshold suffix, got:\n%s", got)
	}
}

func TestRSIList_RendersAdaptiveTier(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("any-id", domain.Item{InstrumentName: "Lukoil", RSILength: 11, RSIValue: 28})

	thresholds := map[string]dto.Thresholds{
		"any-id": {P5: 24, P15: 31},
	}

	got := RSIList(info, dto.StrategyKindDividend, thresholds)

	if !strings.Contains(got, "Lukoil 🟡") {
		t.Errorf("expected 'Lukoil 🟡' in RSIList, got:\n%s", got)
	}
	if !strings.Contains(got, "(p5=24.0, p15=31.0)") {
		t.Errorf("expected threshold suffix in RSIList, got:\n%s", got)
	}
}
