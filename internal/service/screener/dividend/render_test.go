package dividend

import (
	"strings"
	"testing"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
)

func TestRender_ContainsHeaderTopAndStats(t *testing.T) {
	ranked := []RankedShare{
		{Share: &model.Share{Name: "Лукойл", Ticker: "LKOH"}, Scored: rank.ScoredCompany{Composite: 82.5}},
	}
	stats := Stats{Universe: 40, Ranked: 25, Gated: 15, ByReason: map[string]int{"yield trap": 5}}

	out := Render(ranked, stats)

	for _, want := range []string{"Дивидендный скринер", "Лукойл", "LKOH", "82", "Отсеяно", "yield trap"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n%s", want, out)
		}
	}
}

func TestRender_GroupsBySector(t *testing.T) {
	ranked := []RankedShare{
		{Share: &model.Share{Ticker: "MSRS", Name: "Rosseti", Sector: "utilities"},
			Scored: rank.ScoredCompany{Composite: 74}},
		{Share: &model.Share{Ticker: "TATN", Name: "Tatneft", Sector: "energy"},
			Scored: rank.ScoredCompany{Composite: 73}},
		{Share: &model.Share{Ticker: "BSPB", Name: "BSPB", Sector: "financial"},
			Scored: rank.ScoredCompany{Composite: 72}},
		{Share: &model.Share{Ticker: "SBER", Name: "Sber", Sector: "financial"},
			Scored: rank.ScoredCompany{Composite: 64}},
	}
	out := Render(ranked, Stats{Universe: 100, Ranked: 4, Gated: 96})

	// Финансовый сектор представлен подзаголовком с числом имён и средним.
	if !strings.Contains(out, "2 имени") && !strings.Contains(out, "2 имён") {
		t.Errorf("ожидался агрегат по числу имён финансов, got:\n%s", out)
	}
	// Сектор с #1 (utilities, MSRS 74) идёт раньше финансов (лучший 72).
	if strings.Index(out, "MSRS") > strings.Index(out, "SBER") {
		t.Errorf("сектор с лучшим композитом должен идти первым:\n%s", out)
	}
	// Неизвестный код не появляется как «Прочее», раз все коды известны.
	// (проверка ярлыка отдельно ниже)
}

func TestSectorLabel_Fallback(t *testing.T) {
	if got := sectorLabel("no_such_code_xyz"); !strings.Contains(got, "Прочее") {
		t.Errorf("неизвестный код → «Прочее», got %q", got)
	}
}
