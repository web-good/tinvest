package notification

import (
	"strings"
	"testing"

	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestTrade_RendersTrendMarkers(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("with-id", domain.Item{InstrumentName: "Yandex", RSIValue: 30})
	info.WriteToMap("against-id", domain.Item{InstrumentName: "Polyus", RSIValue: 30})
	info.WriteToMap("nofilter-id", domain.Item{InstrumentName: "T-Tech", RSIValue: 30})

	trends := map[string]dto.TrendStatus{
		"with-id":    dto.TrendWith,
		"against-id": dto.TrendAgainst,
		// "nofilter-id" intentionally missing → no marker rendered.
	}

	got := Trade(info, dto.StrategyKindGrowth, trends)

	if !strings.Contains(got, "Yandex 🟢 ✅") {
		t.Errorf("expected 'Yandex 🟢 ✅' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Polyus 🟢 🚫") {
		t.Errorf("expected 'Polyus 🟢 🚫' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "T-Tech 🟢\n") {
		t.Errorf("expected 'T-Tech 🟢' without trailing marker, got:\n%s", got)
	}
}

func TestTrade_NoTrendsArgumentSafeForGoldKind(t *testing.T) {
	info := domain.NewInfo()
	info.WriteToMap("any-id", domain.Item{InstrumentName: "Lukoil", RSIValue: 28})

	// Gold flow passes a nil/empty trends map. Should render without markers
	// and without panicking on lookup.
	got := Trade(info, dto.StrategyKindDividend, nil)

	if !strings.Contains(got, "🥇") {
		t.Errorf("expected gold medal in header, got:\n%s", got)
	}
	if !strings.Contains(got, "Lukoil 🟢\n") {
		t.Errorf("expected 'Lukoil 🟢' without trailing marker, got:\n%s", got)
	}
	if strings.Contains(got, "✅") || strings.Contains(got, "🚫") {
		t.Errorf("Gold output should have no trend markers, got:\n%s", got)
	}
}
