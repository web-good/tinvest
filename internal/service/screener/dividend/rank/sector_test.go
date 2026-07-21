package rank

import "testing"

func TestClassifySector(t *testing.T) {
	cases := []struct {
		in   string
		want SectorKind
	}{
		{"financial", SectorFinancial},
		{"Financial", SectorFinancial},
		{"FINANCIAL", SectorFinancial},
		{"energy", SectorOther},
		{"", SectorOther},
		{"unknown_code", SectorOther},
	}
	for _, c := range cases {
		if got := ClassifySector(c.in); got != c.want {
			t.Errorf("ClassifySector(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
