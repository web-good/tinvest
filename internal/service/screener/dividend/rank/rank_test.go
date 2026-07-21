package rank

import (
	"testing"

	"tinvest/internal/model"
)

// aboveCapFloor — капитализация заведомо выше DefaultConfig().MinMarketCap,
// чтобы фильтр ликвидности не мешал тестам, проверяющим другое поведение.
// Читает актуальный сид, поэтому устойчива к калибровке порога.
var aboveCapFloor = DefaultConfig().MinMarketCap + 1

// aboveFloatFloor — free-float заведомо выше DefaultConfig().MinFreeFloat,
// чтобы фильтр тонкого флоата не мешал тестам про другое поведение.
// Читает актуальный сид, поэтому устойчива к калибровке порога.
var aboveFloatFloor = DefaultConfig().MinFreeFloat + 0.01

func byUID(scored []ScoredCompany) map[string]ScoredCompany {
	m := make(map[string]ScoredCompany, len(scored))
	for _, s := range scored {
		m[s.AssetUID] = s
	}
	return m
}

func TestRank_GateHighLeverage(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUID: "lev", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 50, NetDebtToEbitda: 5, EbitdaTtm: 100, Roic: 0.1, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, DefaultConfig()))
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
	got := byUID(Rank(u, nil, DefaultConfig()))
	if got["nodiv"].GateReason == "" {
		t.Fatalf("expected gate for no dividend")
	}
}

func TestRank_YieldTrap(t *testing.T) {
	u := []*model.Fundamentals{
		{AssetUID: "trap", ForwardAnnualDividendYield: 25, DividendPayoutRatioFy: 110, NetDebtToEbitda: 3.5, EbitdaTtm: 100, FreeCashFlowTtm: -50, Roic: 0.05, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, DefaultConfig()))
	if !got["trap"].YieldTrap || got["trap"].GateReason == "" {
		t.Fatalf("expected yield-trap gate, got %+v", got["trap"])
	}
}

func TestRank_MissingFundamentalsNoLongerGated(t *testing.T) {
	// Платит дивиденд (yield 8), но EBITDA и payout не пришли от API (0).
	// Раньше отсеивалось как "нет ключевых данных" — теперь должно выживать.
	u := []*model.Fundamentals{
		{AssetUID: "nodata", ForwardAnnualDividendYield: 8, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, DefaultConfig()))
	if got["nodata"].GateReason != "" {
		t.Fatalf("dividend payer must survive, gated: %q", got["nodata"].GateReason)
	}
}

func TestRank_KeepsBankLikeDividendPayer(t *testing.T) {
	// SBER-подобный: банк платит дивиденд (yield через TTM 27),
	// но EBITDA у банка отсутствует (0) и payout пришёл 0.
	u := []*model.Fundamentals{
		{AssetUID: "bank", DividendYieldDailyTtm: 27, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0, EbitdaTtm: 0, Roe: 0.22, FreeCashFlowTtm: 100, EvToEbitdaMrq: 0, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, DefaultConfig()))
	if got["bank"].GateReason != "" {
		t.Fatalf("bank dividend payer must survive, gated: %q", got["bank"].GateReason)
	}
}

func TestRank_NeutralSustainabilityWhenPayoutMissing(t *testing.T) {
	// payout отсутствует (0) → нейтральный payoutFit 0.5, а не 0;
	// FCF > 0 добавляет 0.3. Sustainability = 0.7*0.5 + 0.3 = 0.65.
	u := []*model.Fundamentals{
		{AssetUID: "nopayout", DividendYieldDailyTtm: 12, DividendPayoutRatioFy: 0, NetDebtToEbitda: 0.5, EbitdaTtm: 0, Roe: 0.2, FreeCashFlowTtm: 100, EvToEbitdaMrq: 5, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, DefaultConfig()))
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
		{AssetUID: "strong", ForwardAnnualDividendYield: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, EbitdaTtm: 100, Roic: 0.25, EvToEbitdaMrq: 3, FreeCashFlowTtm: 500, FiveYearAnnualDividendGrowthRate: 0.15, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
		// слабая: высокий долг (но <4), высокий payout, низкий ROIC, дорогая
		{AssetUID: "weak", ForwardAnnualDividendYield: 7, DividendPayoutRatioFy: 95, NetDebtToEbitda: 3.5, EbitdaTtm: 100, Roic: 0.03, EvToEbitdaMrq: 12, FreeCashFlowTtm: 10, FiveYearAnnualDividendGrowthRate: -0.05, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	scored := Rank(u, nil, DefaultConfig())
	if scored[0].AssetUID != "strong" {
		t.Fatalf("order = %v, want strong first", []string{scored[0].AssetUID, scored[1].AssetUID})
	}
	if scored[0].Composite <= scored[1].Composite {
		t.Fatalf("composite not ordered: %v <= %v", scored[0].Composite, scored[1].Composite)
	}
}

func TestRank_DegenerateGrowthPoolIsNeutral(t *testing.T) {
	// Весь пул выживших имеет FiveYearAnnualDividendGrowthRate == 0 (поля нет у API).
	// DivGrowth должен быть нейтральным 0.5, а не 0.
	u := []*model.Fundamentals{
		{AssetUID: "a", DividendYieldDailyTtm: 10, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, EbitdaTtm: 100, Roic: 0.2, EvToEbitdaMrq: 4, FreeCashFlowTtm: 100, FiveYearAnnualDividendGrowthRate: 0, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
		{AssetUID: "b", DividendYieldDailyTtm: 9, DividendPayoutRatioFy: 50, NetDebtToEbitda: 1.0, EbitdaTtm: 100, Roic: 0.1, EvToEbitdaMrq: 6, FreeCashFlowTtm: 100, FiveYearAnnualDividendGrowthRate: 0, MarketCapitalization: aboveCapFloor, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, DefaultConfig()))
	for _, id := range []string{"a", "b"} {
		if diff := got[id].DivGrowth - 0.5; diff > 1e-9 || diff < -1e-9 {
			t.Fatalf("%s DivGrowth = %v, want 0.5 (neutral degenerate pool)", id, got[id].DivGrowth)
		}
	}
}

func TestRank_GateIlliquid(t *testing.T) {
	cfg := DefaultConfig()
	u := []*model.Fundamentals{
		// Платит дивиденд, но капа ниже порога → отсев по ликвидности.
		{AssetUID: "small", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: cfg.MinMarketCap - 1},
		// Платит дивиденд, но капа не пришла (0) → отсев по ликвидности.
		{AssetUID: "nocap", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: 0},
		// Платит дивиденд, капа выше порога → проходит.
		{AssetUID: "big", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: cfg.MinMarketCap + 1, FreeFloat: aboveFloatFloor},
	}
	got := byUID(Rank(u, nil, cfg))
	if got["small"].GateReason != reasonIlliquid {
		t.Fatalf("small: GateReason = %q, want %q", got["small"].GateReason, reasonIlliquid)
	}
	if got["nocap"].GateReason != reasonIlliquid {
		t.Fatalf("nocap: GateReason = %q, want %q", got["nocap"].GateReason, reasonIlliquid)
	}
	if got["big"].GateReason != "" {
		t.Fatalf("big must survive, gated: %q", got["big"].GateReason)
	}
}

func TestRank_GateThinFloat(t *testing.T) {
	cfg := DefaultConfig()
	u := []*model.Fundamentals{
		// Платит дивиденд, капа выше порога, но free-float ниже порога → отсев.
		{AssetUID: "thin", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: aboveCapFloor, FreeFloat: cfg.MinFreeFloat - 0.001},
		// free-float ровно на пороге → проходит (сравнение строгое <).
		{AssetUID: "edge", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: aboveCapFloor, FreeFloat: cfg.MinFreeFloat},
		// free-float 0 (нет данных / нулевой) при валидной капе и yield → отсев.
		{AssetUID: "zero", ForwardAnnualDividendYield: 12, DividendPayoutRatioFy: 45, NetDebtToEbitda: 0.5, MarketCapitalization: aboveCapFloor, FreeFloat: 0},
	}
	got := byUID(Rank(u, nil, cfg))
	if got["thin"].GateReason != reasonThinFloat {
		t.Fatalf("thin: GateReason = %q, want %q", got["thin"].GateReason, reasonThinFloat)
	}
	if got["zero"].GateReason != reasonThinFloat {
		t.Fatalf("zero: GateReason = %q, want %q", got["zero"].GateReason, reasonThinFloat)
	}
	if got["edge"].GateReason != "" {
		t.Fatalf("edge must survive, gated: %q", got["edge"].GateReason)
	}
}

func TestRank_FinancialSustainabilityIgnoresFCF(t *testing.T) {
	cfg := DefaultConfig()
	base := func(fcf float64) *model.Fundamentals {
		return &model.Fundamentals{
			AssetUID:                   "bank",
			ForwardAnnualDividendYield: 10,
			DividendPayoutRatioFy:      50, // идеальная зона → payoutFit 1.0
			FreeCashFlowTtm:            fcf,
			MarketCapitalization:       100e9,
			FreeFloat:                  0.3,
			PriceToBookTtm:             1.0,
		}
	}
	sec := map[string]SectorKind{"bank": SectorFinancial}

	withZeroFCF := Rank([]*model.Fundamentals{base(0)}, sec, cfg)
	withPosFCF := Rank([]*model.Fundamentals{base(100)}, sec, cfg)

	if withZeroFCF[0].Sustainability != withPosFCF[0].Sustainability {
		t.Fatalf("bank Sustainability must ignore FCF: fcf0=%v fcfPos=%v",
			withZeroFCF[0].Sustainability, withPosFCF[0].Sustainability)
	}
	// payout 50 в идеальной зоне → sustainabilityPayout = 1.0
	if withZeroFCF[0].Sustainability != 1.0 {
		t.Fatalf("bank Sustainability = %v, want 1.0 (payoutFit only)", withZeroFCF[0].Sustainability)
	}
}

func TestRank_FinancialSafetyNeutral(t *testing.T) {
	cfg := DefaultConfig()
	f := &model.Fundamentals{
		AssetUID:                   "bank",
		ForwardAnnualDividendYield: 10,
		DividendPayoutRatioFy:      50,
		// NetDebtToEbitda ниже cfg.MaxNetDebtToEbitda (4.0), чтобы не попасть под
		// (сектор-слепой) gate reasonHighLeverage — иначе фикстура отсеивается
		// до пиллярного цикла и Safety остаётся нулевым значением, а не 0.5.
		// При этом 3 всё ещё даёт нетривиальный leverageScore(3)=0.4 для
		// небанка, демонстрируя, что для банка это значение игнорируется.
		NetDebtToEbitda:      3,
		MarketCapitalization: 100e9,
		FreeFloat:            0.3,
		PriceToBookTtm:       1.0,
	}
	got := Rank([]*model.Fundamentals{f}, map[string]SectorKind{"bank": SectorFinancial}, cfg)
	if got[0].Safety != 0.5 {
		t.Fatalf("bank Safety = %v, want 0.5", got[0].Safety)
	}
}

func TestRank_FinancialValuationByPB(t *testing.T) {
	cfg := DefaultConfig()
	f := &model.Fundamentals{
		AssetUID:                   "bank",
		ForwardAnnualDividendYield: 10,
		DividendPayoutRatioFy:      50,
		EvToEbitdaMrq:              0,   // неприменимо; не должно давать 0.98
		PriceToBookTtm:             1.0, // ≤ IdealHigh → 1.0
		MarketCapitalization:       100e9,
		FreeFloat:                  0.3,
	}
	got := Rank([]*model.Fundamentals{f}, map[string]SectorKind{"bank": SectorFinancial}, cfg)
	if got[0].Valuation != 1.0 {
		t.Fatalf("bank Valuation = %v, want 1.0 (P/B band)", got[0].Valuation)
	}
}

func TestRank_NonFinancialUnchanged(t *testing.T) {
	cfg := DefaultConfig()
	f := &model.Fundamentals{
		AssetUID:                   "oil",
		ForwardAnnualDividendYield: 10,
		DividendPayoutRatioFy:      50,
		FreeCashFlowTtm:            100,  // FCF>0 → +0.3
		NetDebtToEbitda:            -0.1, // чистый кэш → leverageScore 1.0
		MarketCapitalization:       100e9,
		FreeFloat:                  0.3,
	}
	got := Rank([]*model.Fundamentals{f}, map[string]SectorKind{"oil": SectorOther}, cfg)
	// Sustainability = 0.7*1.0 + 0.3*1.0 = 1.0 (старая формула)
	if got[0].Sustainability != 1.0 {
		t.Fatalf("non-bank Sustainability = %v, want 1.0 (0.7*payout + 0.3*FCF)", got[0].Sustainability)
	}
	// Safety = leverageScore(-0.1) = 1.0
	if got[0].Safety != 1.0 {
		t.Fatalf("non-bank Safety = %v, want 1.0", got[0].Safety)
	}
}

func TestBankValuation(t *testing.T) {
	cfg := DefaultConfig() // IdealHigh=1.0, Zero=2.5
	cases := []struct {
		name string
		pb   float64
		want float64
	}{
		{"no data → neutral", 0, 0.5},
		{"negative → neutral", -1, 0.5},
		{"cheap at ideal high → max", 1.0, 1.0},
		{"below ideal high → max", 0.6, 1.0},
		{"at zero point → 0", 2.5, 0.0},
		{"above zero point → 0", 3.0, 0.0},
		{"midpoint → 0.5", 1.75, 0.5}, // (2.5-1.75)/(2.5-1.0)=0.5
	}
	for _, c := range cases {
		if got := bankValuation(c.pb, cfg); got != c.want {
			t.Errorf("%s: bankValuation(%v) = %v, want %v", c.name, c.pb, got, c.want)
		}
	}
}
