package lagboost

import (
	"math"
	"strings"
	"testing"
)

// series строит ряд закрытий длиной days+1: первая цена base, последняя base·(1+ret/100),
// середина — base (Decide смотрит только на концы окна).
func series(days int, base, retPct float64) []float64 {
	out := make([]float64, days+1)
	for i := range out {
		out[i] = base
	}
	out[days] = base * (1 + retPct/100)
	return out
}

var on = Config{Enabled: true, Mult: 2, Pct: 5, Days: 5}

func TestDecide(t *testing.T) {
	cases := []struct {
		name        string
		ticker      []float64
		index       []float64
		cfg         Config
		wantMult    float64
		wantApplied bool
		wantReason  string // подстрока; "" — не проверяется
	}{
		{"отстал на 6 п.п.", series(5, 100, -4), series(5, 2000, 2), on, 2, true, ""},
		{"ровно на пороге — буст", series(5, 100, -5), series(5, 2000, 0), on, 2, true, ""},
		{"чуть не дотянул", series(5, 100, -4.99), series(5, 2000, 0), on, 1, false, ""},
		{"тикер обогнал индекс", series(5, 100, 3), series(5, 2000, -3), on, 1, false, ""},
		{"выключено", series(5, 100, -10), series(5, 2000, 0), Config{Mult: 2, Pct: 5, Days: 5}, 1, false, "выключено"},
		{"мало истории у тикера", series(3, 100, -10), series(5, 2000, 0), on, 1, false, "истории"},
		{"мало истории у индекса", series(5, 100, -10), series(2, 2000, 0), on, 1, false, "истории"},
		{"нулевая цена", []float64{0, 100, 100, 100, 100, 90}, series(5, 2000, 0), on, 1, false, "цена"},
		{"длинный ряд — берутся последние Days+1",
			append([]float64{1, 1, 1}, series(5, 100, -6)...), append([]float64{9, 9}, series(5, 2000, 0)...), on, 2, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.ticker, tc.index, tc.cfg)
			if got.Mult != tc.wantMult || got.Applied != tc.wantApplied {
				t.Fatalf("Decide = %+v, want Mult %v Applied %v", got, tc.wantMult, tc.wantApplied)
			}
			if tc.wantReason != "" && !strings.Contains(got.Reason, tc.wantReason) {
				t.Fatalf("Reason = %q, want подстроку %q", got.Reason, tc.wantReason)
			}
		})
	}
}

func TestDecideLagValue(t *testing.T) {
	got := Decide(series(5, 100, -4), series(5, 2000, 2), on)
	if math.Abs(got.Lag-(-6)) > 1e-9 {
		t.Fatalf("Lag = %v, want -6", got.Lag)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		ok   bool
	}{
		{"дефолт", on, true},
		{"выключено с мусором", Config{Mult: 20, Pct: -1, Days: 0}, true},
		{"множитель 1", Config{Enabled: true, Mult: 1, Pct: 5, Days: 5}, true},
		{"множитель 3", Config{Enabled: true, Mult: 3, Pct: 5, Days: 5}, true},
		{"множитель 20", Config{Enabled: true, Mult: 20, Pct: 5, Days: 5}, false},
		{"множитель 0.5", Config{Enabled: true, Mult: 0.5, Pct: 5, Days: 5}, false},
		{"порог 0", Config{Enabled: true, Mult: 2, Pct: 0, Days: 5}, false},
		{"окно 0", Config{Enabled: true, Mult: 2, Pct: 5, Days: 0}, false},
		{"окно 61", Config{Enabled: true, Mult: 2, Pct: 5, Days: 61}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); (err == nil) != tc.ok {
				t.Fatalf("Validate(%+v) = %v, want ok=%v", tc.cfg, err, tc.ok)
			}
		})
	}
}

func TestNote(t *testing.T) {
	d := Decide(series(5, 100, -4), series(5, 2000, 2.3), on)
	if got := d.Note(5); got != "Размер ×2: отстал от IMOEX на 6.3% за 5д" {
		t.Fatalf("Note = %q", got)
	}
	if got := (Decision{Mult: 1}).Note(5); got != "" {
		t.Fatalf("Note без буста = %q, want пусто", got)
	}
}
