package pipeline

import (
	"testing"
	"tinvest/internal/domain"
)

func TestTopByYTM(t *testing.T) {
	in := []domain.BondReport{
		{Name: "A", PercentByYear: 10},
		{Name: "B", PercentByYear: 14},
		{Name: "C", PercentByYear: 12},
	}
	got := topByYTM(in, 2)
	if len(got) != 2 {
		t.Fatalf("ожидалось 2, получено %d", len(got))
	}
	if got[0].Name != "B" || got[1].Name != "C" {
		t.Fatalf("порядок по YTM неверен: %+v", got)
	}
}
