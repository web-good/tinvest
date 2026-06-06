package levels

import (
	"math"
	"testing"

	"tinvest/pkg/indicators"
)

// makeProfile builds a minimal VolumeProfile directly (no candles needed).
func makeProfile(centers, vols []float64) indicators.VolumeProfile {
	return indicators.VolumeProfile{
		BinCenter: centers,
		BinVol:    vols,
	}
}

const eps = 1e-9

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < eps
}

// ---------------------------------------------------------------------------
// ExtractLevels
// ---------------------------------------------------------------------------

func TestExtractLevels_DegenerateProfile_NilResult(t *testing.T) {
	vp := indicators.VolumeProfile{} // no bins
	got := ExtractLevels(vp, 1.5)
	if got != nil {
		t.Fatalf("degenerate profile: want nil, got %v", got)
	}
}

func TestExtractLevels_NonPositiveHvnFactor_NilResult(t *testing.T) {
	vp := makeProfile(
		[]float64{10, 20, 30},
		[]float64{100, 200, 150},
	)
	for _, f := range []float64{0, -1, -0.001} {
		if got := ExtractLevels(vp, f); got != nil {
			t.Fatalf("hvnFactor=%v: want nil, got %v", f, got)
		}
	}
}

// Two isolated HVN clusters separated by a low-volume bin:
//
//	bins:  [10, 20, 30, 40, 50]
//	vols:  [300, 400, 50, 250, 350]
//	mean = (300+400+50+250+350)/5 = 270
//	hvnFactor = 1.0 → threshold = 270
//	HVN: bins 0(300>=270), 1(400>=270), 2(50<270), 3(250<270), 4(350>=270)
//	Clusters: [0,1] and [4]
//	Cluster A price = (10*300 + 20*400)/(300+400) = (3000+8000)/700 = 11000/700 ≈ 15.714286
//	Cluster A strength = 700 / 1350 ≈ 0.518519
//	Cluster B price = 50
//	Cluster B strength = 350 / 1350 ≈ 0.259259
func TestExtractLevels_TwoHVNClusters(t *testing.T) {
	vp := makeProfile(
		[]float64{10, 20, 30, 40, 50},
		[]float64{300, 400, 50, 250, 350},
	)
	got := ExtractLevels(vp, 1.0)

	if len(got) != 2 {
		t.Fatalf("want 2 levels, got %d: %v", len(got), got)
	}

	totalVol := 300.0 + 400 + 50 + 250 + 350 // 1350

	wantPrice0 := (10*300.0 + 20*400.0) / (300.0 + 400.0)
	wantStrength0 := (300.0 + 400.0) / totalVol
	wantPrice1 := 50.0
	wantStrength1 := 350.0 / totalVol

	if !approxEq(got[0].Price, wantPrice0) {
		t.Errorf("level[0].Price = %v, want %v", got[0].Price, wantPrice0)
	}
	if !approxEq(got[0].Strength, wantStrength0) {
		t.Errorf("level[0].Strength = %v, want %v", got[0].Strength, wantStrength0)
	}
	if !approxEq(got[1].Price, wantPrice1) {
		t.Errorf("level[1].Price = %v, want %v", got[1].Price, wantPrice1)
	}
	if !approxEq(got[1].Strength, wantStrength1) {
		t.Errorf("level[1].Strength = %v, want %v", got[1].Strength, wantStrength1)
	}

	// Must be ascending by price.
	if got[0].Price >= got[1].Price {
		t.Errorf("levels not ascending: %v >= %v", got[0].Price, got[1].Price)
	}
}

// All bins qualify as HVN → one merged level.
//
//	bins:  [10, 20, 30]
//	vols:  [100, 200, 300]
//	mean = 200, threshold at factor=0.5 → 100; all >= 100
//	Cluster price = (10*100 + 20*200 + 30*300) / 600 = (1000+4000+9000)/600 = 14000/600 ≈ 23.333
//	Cluster strength = 600/600 = 1.0
func TestExtractLevels_AllBinsMergeIntoOne(t *testing.T) {
	vp := makeProfile(
		[]float64{10, 20, 30},
		[]float64{100, 200, 300},
	)
	got := ExtractLevels(vp, 0.5)

	if len(got) != 1 {
		t.Fatalf("want 1 level, got %d: %v", len(got), got)
	}
	wantPrice := (10*100.0 + 20*200.0 + 30*300.0) / 600.0
	if !approxEq(got[0].Price, wantPrice) {
		t.Errorf("level[0].Price = %v, want %v", got[0].Price, wantPrice)
	}
	if !approxEq(got[0].Strength, 1.0) {
		t.Errorf("level[0].Strength = %v, want 1.0", got[0].Strength)
	}
}

// No bins qualify → nil (not an empty slice; nil is fine per spec, but
// an empty slice is also acceptable — test for "no levels").
//
//	bins:  [10, 20, 30]
//	vols:  [10, 10, 10]
//	mean = 10, hvnFactor = 2.0 → threshold = 20; none qualify
func TestExtractLevels_NoBinsQualify_EmptyOrNil(t *testing.T) {
	vp := makeProfile(
		[]float64{10, 20, 30},
		[]float64{10, 10, 10},
	)
	got := ExtractLevels(vp, 2.0)
	if len(got) != 0 {
		t.Fatalf("no bins qualify: want 0 levels, got %d: %v", len(got), got)
	}
}

// Single-bin profile that qualifies.
func TestExtractLevels_SingleBinProfile(t *testing.T) {
	vp := makeProfile([]float64{100}, []float64{500})
	got := ExtractLevels(vp, 1.0)

	if len(got) != 1 {
		t.Fatalf("want 1 level, got %d", len(got))
	}
	if !approxEq(got[0].Price, 100.0) {
		t.Errorf("Price = %v, want 100", got[0].Price)
	}
	if !approxEq(got[0].Strength, 1.0) {
		t.Errorf("Strength = %v, want 1.0", got[0].Strength)
	}
}

// Adjacent HVN bins that span the whole profile merge correctly.
//
//	bins:  [5, 15, 25, 35]
//	vols:  [400, 600, 200, 100]
//	mean = 325, hvnFactor = 1.2 → threshold = 390
//	HVN: 0(400>=390✓), 1(600>=390✓), 2(200<390), 3(100<390)
//	Cluster: [0,1]  price = (5*400+15*600)/1000 = (2000+9000)/1000 = 11
//	strength = 1000/1300
func TestExtractLevels_AdjacentHVNMerge(t *testing.T) {
	vp := makeProfile(
		[]float64{5, 15, 25, 35},
		[]float64{400, 600, 200, 100},
	)
	got := ExtractLevels(vp, 1.2)

	if len(got) != 1 {
		t.Fatalf("want 1 level (merged adjacent HVNs), got %d: %v", len(got), got)
	}
	totalVol := 400.0 + 600 + 200 + 100 // 1300
	wantPrice := (5*400.0 + 15*600.0) / 1000.0
	wantStrength := 1000.0 / totalVol
	if !approxEq(got[0].Price, wantPrice) {
		t.Errorf("Price = %v, want %v", got[0].Price, wantPrice)
	}
	if !approxEq(got[0].Strength, wantStrength) {
		t.Errorf("Strength = %v, want %v", got[0].Strength, wantStrength)
	}
}

// hvnFactor=1.0 exactly: a bin whose volume == mean is still HVN (>= boundary).
//
//	bins:  [10, 20, 30]
//	vols:  [100, 200, 300]
//	mean = 200; hvnFactor=1.0 → threshold=200; bin 1 (200>=200) qualifies.
func TestExtractLevels_ExactMeanIsHVN(t *testing.T) {
	vp := makeProfile(
		[]float64{10, 20, 30},
		[]float64{100, 200, 300},
	)
	got := ExtractLevels(vp, 1.0)
	// All qualify: 100 < 200 → no; 200 >= 200 → yes; 300 >= 200 → yes.
	// Cluster [1,2]: price = (20*200+30*300)/500 = (4000+9000)/500 = 26
	// strength = 500/600
	if len(got) != 1 {
		t.Fatalf("want 1 level, got %d: %v", len(got), got)
	}
	wantPrice := (20*200.0 + 30*300.0) / 500.0
	wantStrength := 500.0 / 600.0
	if !approxEq(got[0].Price, wantPrice) {
		t.Errorf("Price = %v, want %v", got[0].Price, wantPrice)
	}
	if !approxEq(got[0].Strength, wantStrength) {
		t.Errorf("Strength = %v, want %v", got[0].Strength, wantStrength)
	}
}

// All bin volumes are zero → totalVol == 0; ExtractLevels must return nil
// to avoid NaN from 0/0 division.
func TestExtractLevels_AllZeroVolumes_NilResult(t *testing.T) {
	vp := makeProfile([]float64{10, 20}, []float64{0, 0})
	got := ExtractLevels(vp, 1.5)
	if got != nil {
		t.Fatalf("all-zero volumes: want nil, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// NearestSupportBelow
// ---------------------------------------------------------------------------

func TestNearestSupportBelow_NilLevels(t *testing.T) {
	_, ok := NearestSupportBelow(nil, 50)
	if ok {
		t.Fatal("nil levels: want ok=false")
	}
}

func TestNearestSupportBelow_EmptyLevels(t *testing.T) {
	_, ok := NearestSupportBelow([]Level{}, 50)
	if ok {
		t.Fatal("empty levels: want ok=false")
	}
}

func TestNearestSupportBelow_AllLevelsAbove(t *testing.T) {
	lvls := []Level{{Price: 60}, {Price: 80}}
	_, ok := NearestSupportBelow(lvls, 50)
	if ok {
		t.Fatal("all levels above price: want ok=false")
	}
}

func TestNearestSupportBelow_PriceExactlyOnLevel_Excluded(t *testing.T) {
	// strict inequality — level at exactly the price must NOT be returned
	lvls := []Level{{Price: 50, Strength: 0.3}, {Price: 70, Strength: 0.5}}
	_, ok := NearestSupportBelow(lvls, 50)
	if ok {
		t.Fatal("level at exact price must be excluded (strict <)")
	}
}

func TestNearestSupportBelow_ReturnsHighestBelow(t *testing.T) {
	lvls := []Level{
		{Price: 30, Strength: 0.1},
		{Price: 50, Strength: 0.3},
		{Price: 70, Strength: 0.5},
	}
	lvl, ok := NearestSupportBelow(lvls, 65)
	if !ok {
		t.Fatal("want ok=true")
	}
	if !approxEq(lvl.Price, 50) {
		t.Errorf("Price = %v, want 50 (highest strictly below 65)", lvl.Price)
	}
	if !approxEq(lvl.Strength, 0.3) {
		t.Errorf("Strength = %v, want 0.3", lvl.Strength)
	}
}

func TestNearestSupportBelow_SingleLevelBelow(t *testing.T) {
	lvls := []Level{{Price: 40, Strength: 0.2}}
	lvl, ok := NearestSupportBelow(lvls, 41)
	if !ok {
		t.Fatal("want ok=true")
	}
	if !approxEq(lvl.Price, 40) {
		t.Errorf("Price = %v, want 40", lvl.Price)
	}
}

// ---------------------------------------------------------------------------
// NearestResistanceAbove
// ---------------------------------------------------------------------------

func TestNearestResistanceAbove_NilLevels(t *testing.T) {
	_, ok := NearestResistanceAbove(nil, 50)
	if ok {
		t.Fatal("nil levels: want ok=false")
	}
}

func TestNearestResistanceAbove_EmptyLevels(t *testing.T) {
	_, ok := NearestResistanceAbove([]Level{}, 50)
	if ok {
		t.Fatal("empty levels: want ok=false")
	}
}

func TestNearestResistanceAbove_AllLevelsBelow(t *testing.T) {
	lvls := []Level{{Price: 30}, {Price: 40}}
	_, ok := NearestResistanceAbove(lvls, 50)
	if ok {
		t.Fatal("all levels below price: want ok=false")
	}
}

func TestNearestResistanceAbove_PriceExactlyOnLevel_Excluded(t *testing.T) {
	// strict inequality — level at exactly the price must NOT be returned
	lvls := []Level{{Price: 50, Strength: 0.3}, {Price: 70, Strength: 0.5}}
	_, ok := NearestResistanceAbove(lvls, 70)
	if ok {
		t.Fatal("level at exact price must be excluded (strict >)")
	}
}

func TestNearestResistanceAbove_ReturnsLowestAbove(t *testing.T) {
	lvls := []Level{
		{Price: 30, Strength: 0.1},
		{Price: 50, Strength: 0.3},
		{Price: 70, Strength: 0.5},
	}
	lvl, ok := NearestResistanceAbove(lvls, 35)
	if !ok {
		t.Fatal("want ok=true")
	}
	if !approxEq(lvl.Price, 50) {
		t.Errorf("Price = %v, want 50 (lowest strictly above 35)", lvl.Price)
	}
	if !approxEq(lvl.Strength, 0.3) {
		t.Errorf("Strength = %v, want 0.3", lvl.Strength)
	}
}

func TestNearestResistanceAbove_SingleLevelAbove(t *testing.T) {
	lvls := []Level{{Price: 80, Strength: 0.4}}
	lvl, ok := NearestResistanceAbove(lvls, 79)
	if !ok {
		t.Fatal("want ok=true")
	}
	if !approxEq(lvl.Price, 80) {
		t.Errorf("Price = %v, want 80", lvl.Price)
	}
}
