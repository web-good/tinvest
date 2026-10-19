package core

import (
	"testing"
	"time"
)

// volSeries is n main-session bars from Monday 10:00, every volume 100 except bar i at spike.
func volSeries(n, i int, spike int64) series {
	ser := series{c: make([]float64, n), v: make([]int64, n), t: make([]time.Time, n)}
	for j := range ser.v {
		ser.v[j] = 100
		ser.t[j] = at("2026-09-14 10:00").Add(time.Duration(j) * step)
	}
	ser.v[i] = spike
	return ser
}

// The average skips the morning and evening sessions, the closing auction bar and weekends: it
// reaches back over them to the previous main-session bars.
func TestVolumeAveragesMainSessionOnly(t *testing.T) {
	stamps := []string{
		"2026-09-11 18:15", "2026-09-11 18:30", // Friday main session
		"2026-09-11 18:45", "2026-09-11 20:00", // closing auction, evening
		"2026-09-12 12:00",                     // Saturday
		"2026-09-14 07:00", "2026-09-14 09:45", // Monday morning session
		"2026-09-14 10:00", // the breakout bar
	}
	ser := series{c: make([]float64, len(stamps)), v: make([]int64, len(stamps)), t: make([]time.Time, len(stamps))}
	for j, s := range stamps {
		ser.t[j], ser.v[j] = at(s), 1
	}
	ser.v[0], ser.v[1] = 100, 200
	ser.v[len(stamps)-1] = 151
	vol, avg, ok := volP(1, 2).volumeOK(ser, len(stamps)-1)
	if !ok || avg != 150 || vol != 151 {
		t.Fatalf("vol %v avg %v ok %v, want 151 150 true", vol, avg, ok)
	}
	if _, _, ok := volP(1, 3).volumeOK(ser, len(stamps)-1); ok {
		t.Fatal("only two main-session bars before: want fail")
	}
}

func volP(mult float64, bars int) *Strategy {
	p := DefaultParams()
	p.VolumeMult, p.VolBars = mult, bars
	return newS(p)
}

func TestVolumeAboveAverage(t *testing.T) {
	for spike, want := range map[int64]bool{151: true, 150: false, 100: false} {
		if _, _, ok := volP(1.5, 3).volumeOK(volSeries(10, 9, spike), 9); ok != want {
			t.Errorf("spike %d: ok = %v, want %v", spike, ok, want)
		}
	}
}

// The average covers exactly the VolBars bars before the bar: a spike earlier is outside.
func TestVolumeAveragesPreviousBars(t *testing.T) {
	ser := volSeries(10, 9, 151)
	ser.v[5] = 100000 // four bars back, outside VolBars 3
	if _, avg, ok := volP(1.5, 3).volumeOK(ser, 9); !ok || avg != 100 {
		t.Fatalf("spike outside: avg %v ok %v, want 100 true", avg, ok)
	}
	ser.v[6] = 100000 // three bars back, inside
	if _, _, ok := volP(1.5, 3).volumeOK(ser, 9); ok {
		t.Fatal("spike inside: want fail")
	}
}

func TestVolumeFailsClosed(t *testing.T) {
	if _, _, ok := volP(1.5, 20).volumeOK(volSeries(10, 9, 1000), 9); ok {
		t.Fatal("too few bars: want fail")
	}
	if _, _, ok := volP(1.5, 3).volumeOK(series{c: make([]float64, 10)}, 9); ok {
		t.Fatal("no volumes: want fail")
	}
}

func TestVolumeOffPasses(t *testing.T) {
	if _, _, ok := volP(0, 20).volumeOK(series{c: make([]float64, 10)}, 9); !ok {
		t.Fatal("VolumeMult 0: want pass")
	}
}
