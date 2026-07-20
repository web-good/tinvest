package dividend

import (
	"context"
	"testing"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/mocks"

	"github.com/stretchr/testify/mock"
)

func fundUniverse() ([]*model.Share, []*model.Fundamentals) {
	shares := []*model.Share{
		{ID: "i-strong", AssetUID: "a-strong", Name: "Strong", DivYieldFlag: true},
		{ID: "i-mid", AssetUID: "a-mid", Name: "Mid", DivYieldFlag: true},
		{ID: "i-weak", AssetUID: "a-weak", Name: "Weak", DivYieldFlag: true},
		{ID: "i-gated", AssetUID: "a-gated", Name: "Gated", DivYieldFlag: true},
	}
	funds := []*model.Fundamentals{
		{AssetUID: "a-strong", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.3, EbitdaTtm: 100, Roic: 0.25, EvToEbitdaMrq: 3, FreeCashFlowTtm: 500, FiveYearAnnualDividendGrowthRate: 0.2},
		{AssetUID: "a-mid", ForwardAnnualDividendYield: 8, DividendPayoutRatioFy: 55, NetDebtToEbitda: 1.5, EbitdaTtm: 100, Roic: 0.12, EvToEbitdaMrq: 6, FreeCashFlowTtm: 100, FiveYearAnnualDividendGrowthRate: 0.05},
		{AssetUID: "a-weak", ForwardAnnualDividendYield: 7, DividendPayoutRatioFy: 95, NetDebtToEbitda: 3.5, EbitdaTtm: 100, Roic: 0.03, EvToEbitdaMrq: 12, FreeCashFlowTtm: 10, FiveYearAnnualDividendGrowthRate: -0.05},
		{AssetUID: "a-gated", ForwardAnnualDividendYield: 30, DividendPayoutRatioFy: 130, NetDebtToEbitda: 5, EbitdaTtm: 100, FreeCashFlowTtm: -10},
	}
	return shares, funds
}

func newMockedService(t *testing.T) *service {
	shares, funds := fundUniverse()
	m := mocks.NewMockinstrumentsClient(t)
	m.On("Shares", mock.Anything).Return(shares, nil)
	m.On("GetAssetFundamentals", mock.Anything, mock.Anything).Return(funds, nil)
	return NewService(m)
}

func TestRankBonus_TopGetsMorePoints(t *testing.T) {
	svc := newMockedService(t)

	strong := svc.RankBonus("i-strong")
	weak := svc.RankBonus("i-weak")
	gated := svc.RankBonus("i-gated")
	unknown := svc.RankBonus("i-does-not-exist")

	if strong < weak {
		t.Fatalf("strong bonus %d should be >= weak %d", strong, weak)
	}
	if strong < 1 || strong > 3 {
		t.Fatalf("strong bonus %d out of [1,3]", strong)
	}
	if gated != 0 {
		t.Fatalf("gated bonus = %d, want 0", gated)
	}
	if unknown != 0 {
		t.Fatalf("unknown bonus = %d, want 0", unknown)
	}
}

func TestTop_ReportsGateStats(t *testing.T) {
	svc := newMockedService(t)
	ranked, stats, err := svc.Top(context.Background(), 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 3 {
		t.Fatalf("ranked = %d, want 3 (one gated)", len(ranked))
	}
	if stats.Gated != 1 || stats.Universe != 4 {
		t.Fatalf("stats = %+v, want Gated 1 / Universe 4", stats)
	}
	if ranked[0].Share.ID != "i-strong" {
		t.Fatalf("top = %s, want i-strong", ranked[0].Share.ID)
	}
}

func TestRankBonus_SharedAssetCoversAllInstruments(t *testing.T) {
	shares := []*model.Share{
		{ID: "i-ord", AssetUID: "a-shared", Name: "Shared Ordinary", DivYieldFlag: true},
		{ID: "i-pref", AssetUID: "a-shared", Name: "Shared Preferred", DivYieldFlag: true},
		{ID: "i-weak", AssetUID: "a-weak", Name: "Weak", DivYieldFlag: true},
	}
	funds := []*model.Fundamentals{
		{AssetUID: "a-shared", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.3, EbitdaTtm: 100, Roic: 0.25, EvToEbitdaMrq: 3, FreeCashFlowTtm: 500, FiveYearAnnualDividendGrowthRate: 0.2},
		{AssetUID: "a-weak", ForwardAnnualDividendYield: 7, DividendPayoutRatioFy: 95, NetDebtToEbitda: 3.5, EbitdaTtm: 100, Roic: 0.03, EvToEbitdaMrq: 12, FreeCashFlowTtm: 10, FiveYearAnnualDividendGrowthRate: -0.05},
	}

	m := mocks.NewMockinstrumentsClient(t)
	m.On("Shares", mock.Anything).Return(shares, nil)
	m.On("GetAssetFundamentals", mock.Anything, mock.Anything).Return(funds, nil)
	svc := NewService(m)

	ord := svc.RankBonus("i-ord")
	pref := svc.RankBonus("i-pref")
	if ord != pref {
		t.Fatalf("ord bonus %d != pref bonus %d, want equal (shared asset)", ord, pref)
	}
	if ord <= 0 {
		t.Fatalf("shared asset bonus = %d, want > 0", ord)
	}

	ranked, _, err := svc.Top(context.Background(), 15)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range ranked {
		if r.Scored.AssetUID == "a-shared" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("shared asset appeared %d times in ranked list, want 1", count)
	}
}

func TestTop_NonPositiveNFallsBackToDefault(t *testing.T) {
	svc := newMockedService(t)
	ctx := context.Background()

	// Get with default n
	defaultResult, _, err := svc.Top(ctx, defaultTopN)
	if err != nil {
		t.Fatal(err)
	}

	// Test n=0 falls back to default
	zeroResult, _, err := svc.Top(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(zeroResult) != len(defaultResult) {
		t.Fatalf("Top(ctx, 0) returned %d items, want %d (same as default)", len(zeroResult), len(defaultResult))
	}

	// Test n=-1 falls back to default
	negResult, _, err := svc.Top(ctx, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(negResult) != len(defaultResult) {
		t.Fatalf("Top(ctx, -1) returned %d items, want %d (same as default)", len(negResult), len(defaultResult))
	}

	// Test n=1 caps to 1 and returns top-ranked
	oneResult, _, err := svc.Top(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(oneResult) != 1 {
		t.Fatalf("Top(ctx, 1) returned %d items, want 1", len(oneResult))
	}
	if oneResult[0].Share.ID != "i-strong" {
		t.Fatalf("Top(ctx, 1) returned %s, want i-strong", oneResult[0].Share.ID)
	}
}
