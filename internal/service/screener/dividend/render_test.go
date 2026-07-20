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
