package indicators

import (
	"math"
	"testing"
	"time"
)

// msk is the calendar the session anchor is defined in.
var msk = time.FixedZone("MSK", 3*60*60)

// bar is one synthetic candle for the VWAP tests.
type bar struct {
	t       time.Time
	h, l, c float64
	v       int64
}

// split explodes bars into the parallel slices SessionVWAP consumes.
func split(bars []bar) (h, l, c []float64, v []int64, ts []time.Time) {
	for _, b := range bars {
		h = append(h, b.h)
		l = append(l, b.l)
		c = append(c, b.c)
		v = append(v, b.v)
		ts = append(ts, b.t)
	}
	return
}

// at builds an MSK bar time on the given day.
func at(day, hour, min int) time.Time {
	return time.Date(2026, 3, day, hour, min, 0, 0, msk)
}

func TestSessionVWAPHandComputed(t *testing.T) {
	// Typical prices 100, 102, 104 with equal volume -> VWAP 102,
	// sigma = sqrt(((100-102)^2 + 0 + (104-102)^2)/3) = sqrt(8/3).
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(2, 10, 30), 103, 101, 102, 100},
		{at(2, 11, 0), 105, 103, 104, 100},
	}
	h, l, c, v, ts := split(bars)
	vwap, sigma, bfo := SessionVWAP(h, l, c, v, ts, msk)
	if len(vwap) != 3 || len(sigma) != 3 || len(bfo) != 3 {
		t.Fatalf("lengths = %d/%d/%d want 3/3/3", len(vwap), len(sigma), len(bfo))
	}
	if math.Abs(vwap[2]-102) > 1e-9 {
		t.Fatalf("vwap[2] = %v want 102", vwap[2])
	}
	if want := math.Sqrt(8.0 / 3.0); math.Abs(sigma[2]-want) > 1e-9 {
		t.Fatalf("sigma[2] = %v want %v", sigma[2], want)
	}
	if math.Abs(vwap[0]-100) > 1e-9 || sigma[0] != 0 {
		t.Fatalf("single-bar session: vwap=%v sigma=%v want 100/0", vwap[0], sigma[0])
	}
}

func TestSessionVWAPFirstSessionMarkedIncomplete(t *testing.T) {
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(2, 10, 30), 103, 101, 102, 100},
		{at(3, 10, 0), 201, 199, 200, 100},
		{at(3, 10, 30), 203, 201, 202, 100},
	}
	h, l, c, v, ts := split(bars)
	vwap, _, bfo := SessionVWAP(h, l, c, v, ts, msk)
	if bfo[0] != -1 || bfo[1] != -1 {
		t.Fatalf("first session bfo = %v/%v want -1/-1", bfo[0], bfo[1])
	}
	if bfo[2] != 0 || bfo[3] != 1 {
		t.Fatalf("second session bfo = %v/%v want 0/1", bfo[2], bfo[3])
	}
	// The anchor resets: day 2 must not be dragged by day 1's prices.
	if math.Abs(vwap[3]-201) > 1e-9 {
		t.Fatalf("vwap[3] = %v want 201 (day-2 anchor)", vwap[3])
	}
}

func TestSessionVWAPWeekendGapSplitsSessions(t *testing.T) {
	// 2026-03-06 is a Friday, 2026-03-09 the next Monday.
	bars := []bar{
		{time.Date(2026, 3, 5, 10, 0, 0, 0, msk), 101, 99, 100, 100},
		{time.Date(2026, 3, 6, 10, 0, 0, 0, msk), 101, 99, 100, 100},
		{time.Date(2026, 3, 9, 10, 0, 0, 0, msk), 301, 299, 300, 100},
		{time.Date(2026, 3, 9, 10, 30, 0, 0, msk), 301, 299, 300, 100},
	}
	h, l, c, v, ts := split(bars)
	vwap, _, bfo := SessionVWAP(h, l, c, v, ts, msk)
	if bfo[1] != 0 {
		t.Fatalf("Friday bfo = %d want 0 (own session)", bfo[1])
	}
	if bfo[2] != 0 || bfo[3] != 1 {
		t.Fatalf("Monday bfo = %d/%d want 0/1", bfo[2], bfo[3])
	}
	if math.Abs(vwap[3]-300) > 1e-9 {
		t.Fatalf("vwap[3] = %v want 300 (Monday anchor)", vwap[3])
	}
}

func TestSessionVWAPZeroVolumeSession(t *testing.T) {
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(3, 10, 0), 201, 199, 200, 0},
		{at(3, 10, 30), 203, 201, 202, 0},
	}
	h, l, c, v, ts := split(bars)
	vwap, sigma, _ := SessionVWAP(h, l, c, v, ts, msk)
	if vwap[2] != 0 || sigma[2] != 0 {
		t.Fatalf("zero-volume session: vwap=%v sigma=%v want 0/0", vwap[2], sigma[2])
	}
	if math.IsNaN(vwap[2]) || math.IsNaN(sigma[2]) {
		t.Fatalf("zero-volume session produced NaN")
	}
}

func TestSessionVWAPRejectsMisalignedInput(t *testing.T) {
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(2, 10, 30), 103, 101, 102, 100},
	}
	h, l, c, v, ts := split(bars)
	cases := map[string]func() (vwap, sigma []float64, bfo []int){
		"no times":      func() ([]float64, []float64, []int) { return SessionVWAP(h, l, c, v, nil, msk) },
		"short times":   func() ([]float64, []float64, []int) { return SessionVWAP(h, l, c, v, ts[:1], msk) },
		"short highs":   func() ([]float64, []float64, []int) { return SessionVWAP(h[:1], l, c, v, ts, msk) },
		"short volumes": func() ([]float64, []float64, []int) { return SessionVWAP(h, l, c, v[:1], ts, msk) },
		"empty":         func() ([]float64, []float64, []int) { return SessionVWAP(nil, nil, nil, nil, nil, msk) },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			vwap, sigma, bfo := call()
			if vwap != nil || sigma != nil || bfo != nil {
				t.Fatalf("want nil,nil,nil; got %v,%v,%v", vwap, sigma, bfo)
			}
		})
	}
}

func TestSessionVWAPNilLocationDoesNotPanic(t *testing.T) {
	bars := []bar{{at(2, 10, 0), 101, 99, 100, 100}}
	h, l, c, v, ts := split(bars)
	if vwap, _, _ := SessionVWAP(h, l, c, v, ts, nil); len(vwap) != 1 {
		t.Fatalf("nil loc must degrade to UTC, got len=%d", len(vwap))
	}
}
