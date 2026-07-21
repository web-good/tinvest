package rank

import (
	"sort"

	"tinvest/internal/model"
	"tinvest/pkg/indicators"
)

// ScoredCompany — результат ранжирования одной компании. Отсеянные воротами
// имеют GateReason != "" и Composite == 0.
type ScoredCompany struct {
	AssetUID  string
	Composite float64

	Sustainability float64
	Safety         float64
	DivGrowth      float64
	Quality        float64
	Valuation      float64
	YieldScore     float64

	YieldTrap  bool
	GateReason string
}

// Причины отсева (стабильные строки для вывода и тестов).
const (
	reasonNoDividend    = "нет дивиденда"
	reasonIlliquid      = "низкая ликвидность"
	reasonHighLeverage  = "долг > порога"
	reasonUnsustainable = "payout > порога"
	reasonYieldTrap     = "yield trap"
)

func yieldOf(f *model.Fundamentals) float64 {
	if f.ForwardAnnualDividendYield > 0 {
		return f.ForwardAnnualDividendYield
	}
	return f.DividendYieldDailyTtm
}

// gate возвращает (reason, isTrap). Пустой reason => компания проходит.
// Жёсткие основания отсева по данным: нет дивиденда (yield <= 0) и низкая
// ликвидность (MarketCapitalization < MinMarketCap, в т.ч. 0 из-за proto3
// omitempty). Отсутствие EBITDA/payout (0) НЕ исключает компанию — оно
// нейтрально учитывается в пиллярах.
func gate(f *model.Fundamentals, cfg Config) (string, bool) {
	y := yieldOf(f)
	if y <= 0 {
		return reasonNoDividend, false
	}
	if f.MarketCapitalization < cfg.MinMarketCap {
		return reasonIlliquid, false
	}
	trap := y >= cfg.YieldTrapMinYield &&
		(f.DividendPayoutRatioFy > 100 || f.NetDebtToEbitda > 3 || f.FreeCashFlowTtm < 0)
	if trap {
		return reasonYieldTrap, true
	}
	if f.NetDebtToEbitda > cfg.MaxNetDebtToEbitda {
		return reasonHighLeverage, false
	}
	if f.DividendPayoutRatioFy > cfg.MaxPayoutPct {
		return reasonUnsustainable, false
	}
	return "", false
}

// GateDecision — экспортированная обёртка над gate для диагностики (cmd/divscreen):
// возвращает (reason, isTrap) без ранжирования. Пустой reason => компания проходит.
func GateDecision(f *model.Fundamentals, cfg Config) (string, bool) {
	return gate(f, cfg)
}

// payoutFit: 1.0 в идеальной зоне, линейно к 0 у краёв (0 и MaxPayoutPct).
func payoutFit(payout float64, cfg Config) float64 {
	switch {
	case payout >= cfg.PayoutIdealLow && payout <= cfg.PayoutIdealHigh:
		return 1.0
	case payout < cfg.PayoutIdealLow:
		return clamp01(payout / cfg.PayoutIdealLow)
	default: // > ideal high
		span := cfg.MaxPayoutPct - cfg.PayoutIdealHigh
		if span <= 0 {
			return 0
		}
		return clamp01((cfg.MaxPayoutPct - payout) / span)
	}
}

// sustainabilityPayout: как payoutFit, но при отсутствии данных о payout
// (0 из-за proto3 omitempty) возвращает нейтральные 0.5 — «неизвестно»
// не должно ни вознаграждать, ни штрафовать компанию, которая платит дивиденд.
func sustainabilityPayout(payout float64, cfg Config) float64 {
	if payout <= 0 {
		return 0.5
	}
	return payoutFit(payout, cfg)
}

// leverageScore: чем меньше долг, тем лучше. Чистый кэш (<0) — максимум.
func leverageScore(nd float64) float64 {
	switch {
	case nd < 0:
		return 1.0
	case nd <= 1:
		return 0.9
	case nd <= 2:
		return 0.7
	case nd <= 3:
		return 0.4
	default:
		return 0.15
	}
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func Rank(universe []*model.Fundamentals, cfg Config) []ScoredCompany {
	survivors := make([]*model.Fundamentals, 0, len(universe))
	gated := make([]ScoredCompany, 0)

	for _, f := range universe {
		reason, trap := gate(f, cfg)
		if reason != "" {
			gated = append(gated, ScoredCompany{AssetUID: f.AssetUID, GateReason: reason, YieldTrap: trap})
			continue
		}
		survivors = append(survivors, f)
	}

	// Перцентильные пулы по выжившим.
	divGrowth := make([]float64, len(survivors))
	roic := make([]float64, len(survivors))
	evEbitda := make([]float64, len(survivors))
	for i, f := range survivors {
		divGrowth[i] = f.FiveYearAnnualDividendGrowthRate
		roic[i] = qualityMetric(f)
		evEbitda[i] = f.EvToEbitdaMrq
	}

	scored := make([]ScoredCompany, 0, len(survivors))
	for _, f := range survivors {
		sc := ScoredCompany{AssetUID: f.AssetUID}
		sc.Sustainability = 0.7*sustainabilityPayout(f.DividendPayoutRatioFy, cfg) + 0.3*boolScore(f.FreeCashFlowTtm > 0)
		sc.Safety = leverageScore(f.NetDebtToEbitda)
		sc.DivGrowth = percentileOrNeutral(divGrowth, f.FiveYearAnnualDividendGrowthRate)
		sc.Quality = percentileOrNeutral(roic, qualityMetric(f))
		sc.Valuation = 1 - percentileOrNeutral(evEbitda, f.EvToEbitdaMrq) // ниже EV/EBITDA — лучше
		sc.YieldScore = clamp01(minf(yieldOf(f), cfg.YieldCapPct) / cfg.YieldCapPct)

		sc.Composite = 100 * (cfg.WeightSustainability*sc.Sustainability +
			cfg.WeightSafety*sc.Safety +
			cfg.WeightDivGrowth*sc.DivGrowth +
			cfg.WeightQuality*sc.Quality +
			cfg.WeightValuation*sc.Valuation +
			cfg.WeightYield*sc.YieldScore)
		scored = append(scored, sc)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Composite != scored[j].Composite {
			return scored[i].Composite > scored[j].Composite
		}
		return scored[i].AssetUID < scored[j].AssetUID
	})

	return append(scored, gated...)
}

// qualityMetric: ROIC, с фолбэком на ROE, если ROIC не задан.
func qualityMetric(f *model.Fundamentals) float64 {
	if f.Roic != 0 {
		return f.Roic
	}
	return f.Roe
}

// percentileOrNeutral возвращает 0.5, когда в пуле нет разброса (все значения
// равны — например, fundamental-поле отсутствует по всей вселенной), чтобы
// мёртвый сигнал не занижал composite. Иначе — обычный PercentileRank.
func percentileOrNeutral(pool []float64, x float64) float64 {
	if !hasSpread(pool) {
		return 0.5
	}
	return indicators.PercentileRank(pool, x)
}

// hasSpread сообщает, есть ли в пуле хотя бы два различных значения.
func hasSpread(pool []float64) bool {
	if len(pool) < 2 {
		return false
	}
	first := pool[0]
	for _, v := range pool[1:] {
		if v != first {
			return true
		}
	}
	return false
}

func boolScore(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
