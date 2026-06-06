package indicators

import (
	"math"
	"testing"
)

// approxEq returns true if a and b differ by at most tol.
func approxEq(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

// approxSliceEq returns true when two slices have the same length and every
// element pair is within tol of each other.
func approxSliceEq(a, b []float64, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !approxEq(a[i], b[i], tol) {
			return false
		}
	}
	return true
}

func TestComputeVolumeProfile_Degenerate(t *testing.T) {
	zero := VolumeProfile{}

	tests := []struct {
		name    string
		highs   []float64
		lows    []float64
		volumes []int64
		bins    int
	}{
		{
			name:    "bins <= 0",
			highs:   []float64{12},
			lows:    []float64{10},
			volumes: []int64{100},
			bins:    0,
		},
		{
			name:    "negative bins",
			highs:   []float64{12},
			lows:    []float64{10},
			volumes: []int64{100},
			bins:    -1,
		},
		{
			name:    "empty inputs",
			highs:   nil,
			lows:    nil,
			volumes: nil,
			bins:    5,
		},
		{
			name:    "mismatched lengths highs shorter",
			highs:   []float64{12},
			lows:    []float64{10, 11},
			volumes: []int64{100, 50},
			bins:    5,
		},
		{
			name:    "mismatched lengths volumes shorter",
			highs:   []float64{12, 14},
			lows:    []float64{10, 12},
			volumes: []int64{100},
			bins:    5,
		},
		{
			name:    "no price range: max(highs) <= min(lows) when all equal",
			highs:   []float64{10, 10},
			lows:    []float64{10, 10},
			volumes: []int64{100, 50},
			bins:    5,
		},
		{
			name:    "total volume zero",
			highs:   []float64{12, 14},
			lows:    []float64{10, 12},
			volumes: []int64{0, 0},
			bins:    5,
		},
		{
			name:    "negative volume entry",
			highs:   []float64{12, 14},
			lows:    []float64{10, 12},
			volumes: []int64{100, -1},
			bins:    5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeVolumeProfile(tc.highs, tc.lows, tc.volumes, tc.bins)
			if len(got.BinCenter) != 0 || len(got.BinVol) != 0 ||
				got.POC != zero.POC || got.VAHigh != zero.VAHigh || got.VALow != zero.VALow {
				t.Fatalf("expected zero VolumeProfile, got %+v", got)
			}
		})
	}
}

func TestComputeVolumeProfile_ZeroRangeCandle(t *testing.T) {
	// Single candle with low == high == 15; all volume goes into the bin containing 15.
	// Range [15,15] would be degenerate (max==min). But we have two candles so the
	// price range opens up: one at [14,16], one with low==high==15.
	//
	// 3 bins over [14,16], width = 2/3 ≈ 0.6667:
	//   Bin 0: [14.0000, 14.6667), center ≈ 14.3333
	//   Bin 1: [14.6667, 15.3333), center ≈ 15.0000
	//   Bin 2: [15.3333, 16.0000], center ≈ 15.6667
	//
	// Candle A [14,16] vol=60: overlap with each bin = 0.6667/2 = 1/3; vol/bin = 20 each.
	// Candle B low==high==15, vol=90: bin containing 15 is Bin 1 → all 90 to Bin 1.
	//
	// BinVol: [20, 110, 20]. Total = 150. POC = Bin1 center ≈ 15.0.
	// VA: POC contrib = 110 (> 70% of 150 = 105) → VA done immediately.
	// VALow = VAHigh = 15.0.
	highs := []float64{16, 15}
	lows := []float64{14, 15}
	volumes := []int64{60, 90}

	got := ComputeVolumeProfile(highs, lows, volumes, 3)

	const tol = 1e-6
	wantCenters := []float64{14.0 + 1.0/3.0, 15.0, 15.0 + 2.0/3.0}
	wantVols := []float64{20, 110, 20}

	if !approxSliceEq(got.BinCenter, wantCenters, tol) {
		t.Errorf("BinCenter = %v, want %v", got.BinCenter, wantCenters)
	}
	if !approxSliceEq(got.BinVol, wantVols, tol) {
		t.Errorf("BinVol = %v, want %v", got.BinVol, wantVols)
	}
	if !approxEq(got.POC, 15.0, tol) {
		t.Errorf("POC = %v, want 15.0", got.POC)
	}
	if !approxEq(got.VALow, 15.0, tol) {
		t.Errorf("VALow = %v, want 15.0", got.VALow)
	}
	if !approxEq(got.VAHigh, 15.0, tol) {
		t.Errorf("VAHigh = %v, want 15.0", got.VAHigh)
	}
}

func TestComputeVolumeProfile_UniformSingleCandle(t *testing.T) {
	// One candle spanning the full range: all bins get equal volume.
	// 4 bins over [10, 20], width = 2.5 each.
	// Each bin gets 400/4 = 100 vol.
	// POC = first bin (all tied → lowest index → BinCenter[0] = 11.25).
	// Total = 400; 70% = 280. POC alone = 100, expand to include neighbors
	// until cumulative >= 280. Order of expansion (all tied): pick higher
	// or lower arbitrarily when tied; the implementation breaks ties by
	// choosing the higher bin. Step-by-step:
	//   - Start with bin 0 (POC, all bins have vol 100, index 0 is lowest
	//     and chosen because it's first max). Actually when all bins are equal
	//     the first max found at index 0 is POC.
	//   - Accumulated = 100 < 280. Neighbors: below = none, above = bin1 (100).
	//     Add bin1. Accumulated = 200 < 280. Neighbors: below = none, above = bin2 (100).
	//     Add bin2. Accumulated = 300 >= 280. Done. VALow = bin0 center, VAHigh = bin2 center.
	// BinCenters: [11.25, 13.75, 16.25, 18.75]
	// VALow = 11.25, VAHigh = 16.25
	highs := []float64{20}
	lows := []float64{10}
	volumes := []int64{400}

	got := ComputeVolumeProfile(highs, lows, volumes, 4)

	const tol = 1e-6
	wantCenters := []float64{11.25, 13.75, 16.25, 18.75}
	wantVols := []float64{100, 100, 100, 100}

	if !approxSliceEq(got.BinCenter, wantCenters, tol) {
		t.Errorf("BinCenter = %v, want %v", got.BinCenter, wantCenters)
	}
	if !approxSliceEq(got.BinVol, wantVols, tol) {
		t.Errorf("BinVol = %v, want %v", got.BinVol, wantVols)
	}
	if !approxEq(got.POC, 11.25, tol) {
		t.Errorf("POC = %v, want 11.25", got.POC)
	}
	// VA must cover at least 70% = 280; at 3 bins (300) the expansion stops.
	if !approxEq(got.VALow, 11.25, tol) {
		t.Errorf("VALow = %v, want 11.25", got.VALow)
	}
	if !approxEq(got.VAHigh, 16.25, tol) {
		t.Errorf("VAHigh = %v, want 16.25", got.VAHigh)
	}
}

func TestComputeVolumeProfile_ClearPOCAndVA(t *testing.T) {
	// Three candles, 5 bins over [10, 20], width = 2 each.
	// Bins: [10,12) center 11, [12,14) center 13, [14,16) center 15,
	//        [16,18) center 17, [18,20] center 19.
	//
	// Candle A: low=10, high=20, vol=100 → spans all 5 bins uniformly.
	//   Each bin fraction = 2/10 = 0.2; vol/bin = 20.
	//
	// Candle B: low=13, high=15, vol=200 → overlaps Bin1 and Bin2.
	//   Candle range = 2; Bin1 overlap = min(14,15)-max(12,13)=1; Bin2 overlap = min(16,15)-max(14,13)=1.
	//   vol/bin = 200 * (1/2) = 100 each.
	//
	// Candle C: low=16, high=18, vol=60 → exactly Bin3.
	//   Candle range = 2; Bin3 overlap = 2/2 = 1; vol = 60.
	//
	// BinVol: [20, 120, 120, 80, 20]. Total = 360. 70% = 252.
	//
	// Tied POC at Bin1 and Bin2 (both 120). First max encountered = Bin1, center=13.
	// VA expansion from Bin1:
	//   accumulated = 120 < 252.
	//   First VA iteration from POC=Bin1: canExpandDown=true (binVol[0]=20), canExpandUp=true (binVol[2]=120);
	//   upVol(120) >= downVol(20) -> expand up to Bin2, accumulated=240.
	//   Below: Bin0(20), Above: Bin3(80). 80 > 20 → add Bin3: accumulated = 320 >= 252.
	//   Done. VALow = 13 (Bin1), VAHigh = 17 (Bin3).
	highs := []float64{20, 15, 18}
	lows := []float64{10, 13, 16}
	volumes := []int64{100, 200, 60}

	got := ComputeVolumeProfile(highs, lows, volumes, 5)

	const tol = 1e-6
	wantCenters := []float64{11, 13, 15, 17, 19}
	wantVols := []float64{20, 120, 120, 80, 20}

	if !approxSliceEq(got.BinCenter, wantCenters, tol) {
		t.Errorf("BinCenter = %v, want %v", got.BinCenter, wantCenters)
	}
	if !approxSliceEq(got.BinVol, wantVols, tol) {
		t.Errorf("BinVol = %v, want %v", got.BinVol, wantVols)
	}
	if !approxEq(got.POC, 13, tol) {
		t.Errorf("POC = %v, want 13", got.POC)
	}
	if !approxEq(got.VALow, 13, tol) {
		t.Errorf("VALow = %v, want 13", got.VALow)
	}
	if !approxEq(got.VAHigh, 17, tol) {
		t.Errorf("VAHigh = %v, want 17", got.VAHigh)
	}
}

func TestComputeVolumeProfile_SingleBin(t *testing.T) {
	// One bin covering the full range. All volume goes there.
	// POC = that bin's center. VA = same (trivially >= 70%).
	highs := []float64{20, 18}
	lows := []float64{10, 12}
	volumes := []int64{100, 50}

	got := ComputeVolumeProfile(highs, lows, volumes, 1)

	const tol = 1e-6
	// Range [10, 20], single bin center = 15.
	if len(got.BinCenter) != 1 {
		t.Fatalf("len(BinCenter) = %d, want 1", len(got.BinCenter))
	}
	if !approxEq(got.BinCenter[0], 15, tol) {
		t.Errorf("BinCenter[0] = %v, want 15", got.BinCenter[0])
	}
	if !approxEq(got.BinVol[0], 150, tol) {
		t.Errorf("BinVol[0] = %v, want 150", got.BinVol[0])
	}
	if !approxEq(got.POC, 15, tol) {
		t.Errorf("POC = %v, want 15", got.POC)
	}
	if !approxEq(got.VALow, 15, tol) {
		t.Errorf("VALow = %v, want 15", got.VALow)
	}
	if !approxEq(got.VAHigh, 15, tol) {
		t.Errorf("VAHigh = %v, want 15", got.VAHigh)
	}
}

func TestComputeVolumeProfile_BinCentersAscending(t *testing.T) {
	// Verify BinCenter is strictly ascending for a normal profile.
	highs := []float64{100}
	lows := []float64{0}
	volumes := []int64{1000}

	got := ComputeVolumeProfile(highs, lows, volumes, 10)

	if len(got.BinCenter) != 10 {
		t.Fatalf("len(BinCenter) = %d, want 10", len(got.BinCenter))
	}
	for i := 1; i < len(got.BinCenter); i++ {
		if got.BinCenter[i] <= got.BinCenter[i-1] {
			t.Errorf("BinCenter not ascending at index %d: %v <= %v",
				i, got.BinCenter[i], got.BinCenter[i-1])
		}
	}
}

func TestComputeVolumeProfile_VAExpansionBothDirections(t *testing.T) {
	// 5 bins over [0, 10], width=2, centers=[1,3,5,7,9].
	// Bins: [0,2) [2,4) [4,6) [6,8) [8,10].
	//
	// Candle spanning full range [0,10] vol=10 → each bin gets 2/10 * 10 = 2 vol.
	// Candle [4,6] vol=298 → exactly Bin2; vol = 298.
	// Candle [2,4] vol=98 → exactly Bin1; vol = 98.
	// Candle [6,8] vol=78 → exactly Bin3; vol = 78.
	//
	// BinVol: [2, 100, 300, 80, 2]. Total = 484. 70% = 338.8.
	// POC = Bin2, center=5.
	// VA expansion:
	//   accumulated = 300 < 338.8.
	//   Below: Bin1(100), Above: Bin3(80). 100 > 80 → add Bin1: accumulated = 400 >= 338.8.
	//   Done. VALow = 3 (Bin1), VAHigh = 5 (Bin2, POC).
	highs := []float64{10, 6, 4, 8}
	lows := []float64{0, 4, 2, 6}
	volumes := []int64{10, 298, 98, 78}

	got := ComputeVolumeProfile(highs, lows, volumes, 5)

	const tol = 1e-6
	wantCenters := []float64{1, 3, 5, 7, 9}
	wantVols := []float64{2, 100, 300, 80, 2}

	if !approxSliceEq(got.BinCenter, wantCenters, tol) {
		t.Errorf("BinCenter = %v, want %v", got.BinCenter, wantCenters)
	}
	if !approxSliceEq(got.BinVol, wantVols, tol) {
		t.Errorf("BinVol = %v, want %v", got.BinVol, wantVols)
	}
	if !approxEq(got.POC, 5, tol) {
		t.Errorf("POC = %v, want 5", got.POC)
	}
	if !approxEq(got.VALow, 3, tol) {
		t.Errorf("VALow = %v, want 3", got.VALow)
	}
	if !approxEq(got.VAHigh, 5, tol) {
		t.Errorf("VAHigh = %v, want 5", got.VAHigh)
	}
}
