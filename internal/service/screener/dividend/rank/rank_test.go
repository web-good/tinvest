package rank

import (
	"testing"

	"tinvest/internal/model"
)

func byUID(scored []ScoredCompany) map[string]ScoredCompany {
	m := make(map[string]ScoredCompany, len(scored))
	for _, s := range scored {
		m[s.AssetUID] = s
	}
	return m
}

func TestRank_GateHighLeverage(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUID: "lev", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 50, NetDebtToEbitda: 5, EbitdaTtm: 100, Roic: 0.1},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["lev"].GateReason == "" {
		t.Fatalf("expected gate for high leverage, got none")
	}
	if got["lev"].Composite != 0 {
		t.Fatalf("gated composite = %v, want 0", got["lev"].Composite)
	}
}

func TestRank_GateNoDividend(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUID: "nodiv", ForwardAnnualDividendYield: 0, DividendYieldDailyTtm: 0, DividendPayoutRatioFy: 50, NetDebtToEbitda: 1, EbitdaTtm: 100},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nodiv"].GateReason == "" {
		t.Fatalf("expected gate for no dividend")
	}
}

func TestRank_YieldTrap(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUID: "trap", ForwardAnnualDividendYield: 25, DividendPayoutRatioFy: 110, NetDebtToEbitda: 3.5, EbitdaTtm: 100, FreeCashFlowTtm: -50, Roic: 0.05},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if !got["trap"].YieldTrap || got["trap"].GateReason == "" {
		t.Fatalf("expected yield-trap gate, got %+v", got["trap"])
	}
}

func TestRank_MissingFundamentalsNoLongerGated(t *testing.T) {
	// Платит дивиденд (yield 8), но EBITDA и payout не пришли от API (0).
	// Раньше отсеивалось как "нет ключевых данных" — теперь должно выживать.
	u := []*model.Fundamentals{
		{AssetUID: "nodata", ForwardAnnualDividendYield: 8, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nodata"].GateReason != "" {
		t.Fatalf("dividend payer must survive, gated: %q", got["nodata"].GateReason)
	}
}

func TestRank_KeepsBankLikeDividendPayer(t *testing.T) {
	// SBER-подобный: банк платит дивиденд (yield через TTM 27),
	// но EBITDA у банка отсутствует (0) и payout пришёл 0.
	u := []*model.Fundamentals{
		{AssetUID: "bank", DividendYieldDailyTtm: 27, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0, Roe: 0.22, FreeCashFlowTtm: 100, EvToEbitdaMrq: 0},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["bank"].GateReason != "" {
		t.Fatalf("bank dividend payer must survive, gated: %q", got["bank"].GateReason)
	}
}

func TestRank_NeutralSustainabilityWhenPayoutMissing(t *testing.T) {
	// payout отсутствует (0) → нейтральный payoutFit 0.5, а не 0;
	// FCF > 0 добавляет 0.3. Sustainability = 0.7*0.5 + 0.3 = 0.65.
	u := []*model.Fundamentals{
		{AssetUID: "nopayout", DividendYieldDailyTtm: 12, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0.5, EbitdaTtm: 0, Roe: 0.2, FreeCashFlowTtm: 100, EvToEbitdaMrq: 5},
	}
	got := byUID(Rank(u, DefaultConfig()))
	if got["nopayout"].GateReason != "" {
		t.Fatalf("should survive, gated: %q", got["nopayout"].GateReason)
	}
	if diff := got["nopayout"].Sustainability - 0.65; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Sustainability = %v, want 0.65 (neutral payout)", got["nopayout"].Sustainability)
	}
}

func TestRank_OrdersSurvivorsByComposite(t *testing.T) {
	u := []*model.Fundamentals{
		// сильная: низкий долг, идеальный payout, высокий ROIC, дешёвая, рост дивиденда
		{AssetUID: "strong", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, EbitdaTtm: 100, Roic: 0.25, EvToEbitdaMrq: 3, FreeCashFlowTtm: 500, FiveYearAnnualDividendGrowthRate: 0.15},
		// слабая: высокий долг (но <4), высокий payout, низкий ROIC, дорогая
		{AssetUID: "weak", ForwardAnnualDividendYield: 7, DividendPayoutRatioFy: 95, NetDebtToEbitda: 3.5, EbitdaTtm: 100, Roic: 0.03, EvToEbitdaMrq: 12, FreeCashFlowTtm: 10, FiveYearAnnualDividendGrowthRate: -0.05},
	}
	scored := Rank(u, DefaultConfig())
	if scored[0].AssetUID != "strong" {
		t.Fatalf("order = %v, want strong first", []string{scored[0].AssetUID, scored[1].AssetUID})
	}
	if scored[0].Composite <= scored[1].Composite {
		t.Fatalf("composite not ordered: %v <= %v", scored[0].Composite, scored[1].Composite)
	}
}
