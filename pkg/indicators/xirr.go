package indicators

import (
	"errors"
	"math"
	"sort"
	"time"
)

// Solver tuning constants.
const (
	xirrMaxNewtonIter  = 100       // max Newton–Raphson iterations
	xirrNPVTol         = 1e-7      // NPV convergence tolerance (shared by Newton and bisection)
	xirrDerivZeroGuard = 1e-15     // treat derivative as zero below this threshold
	xirrBisectLo       = -0.999999 // bisection initial lower bound
	xirrBisectHi       = 100.0     // bisection initial upper bound
	xirrBisectWidenCap = 1e6       // widen hi up to this cap
	xirrBisectWidthTol = 1e-9      // stop bisection when interval width falls below this
)

// Sentinel errors for XIRR.
var (
	ErrXIRRNoFlows       = errors.New("xirr: need at least 2 cash flows")
	ErrXIRRSameSign      = errors.New("xirr: all cash flows have the same sign — no root")
	ErrXIRRNoConvergence = errors.New("xirr: solver failed to converge")
)

// CashFlow is a dated monetary amount.
// Negative amounts represent outflows (investments); positive amounts represent inflows.
type CashFlow struct {
	Date   time.Time
	Amount float64
}

// XIRR returns the annualised money-weighted internal rate of return (Excel XIRR,
// Actual/365 day count). A result of 0.123 means 12.3 %.
func XIRR(flows []CashFlow) (float64, error) {
	if len(flows) < 2 {
		return 0, ErrXIRRNoFlows
	}

	// Validate sign mix.
	hasNeg, hasPos := false, false
	for _, f := range flows {
		if f.Amount < 0 {
			hasNeg = true
		}
		if f.Amount > 0 {
			hasPos = true
		}
	}
	if !hasNeg || !hasPos {
		return 0, ErrXIRRSameSign
	}

	// Sort a copy by date ascending.
	sorted := make([]CashFlow, len(flows))
	copy(sorted, flows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	base := sorted[0].Date

	// Newton's method.
	r := 0.1
	for i := 0; i < xirrMaxNewtonIter; i++ {
		if r <= -1 || math.IsNaN(r) || math.IsInf(r, 0) {
			break
		}
		n := npv(sorted, base, r)
		if math.Abs(n) < xirrNPVTol {
			return r, nil
		}
		d := npvDerivative(sorted, base, r)
		if math.Abs(d) < xirrDerivZeroGuard {
			break
		}
		next := r - n/d
		if math.IsNaN(next) || math.IsInf(next, 0) || next <= -1 {
			break
		}
		r = next
	}

	// Bisection fallback on [xirrBisectLo, xirrBisectHi].
	lo, hi := xirrBisectLo, xirrBisectHi
	nLo := npv(sorted, base, lo)
	nHi := npv(sorted, base, hi)

	// Widen hi until signs differ, up to a cap.
	for math.Signbit(nLo) == math.Signbit(nHi) && hi < xirrBisectWidenCap {
		hi *= 10
		nHi = npv(sorted, base, hi)
	}
	if math.Signbit(nLo) == math.Signbit(nHi) {
		return 0, ErrXIRRNoConvergence
	}

	for hi-lo > xirrBisectWidthTol {
		mid := (lo + hi) / 2
		nMid := npv(sorted, base, mid)
		if math.Abs(nMid) < xirrNPVTol {
			return mid, nil
		}
		if math.Signbit(nLo) == math.Signbit(nMid) {
			lo = mid
			nLo = nMid
		} else {
			hi = mid
		}
	}

	result := (lo + hi) / 2
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, ErrXIRRNoConvergence
	}
	return result, nil
}

// npv computes Net Present Value for a given rate r, using base as the time origin.
func npv(flows []CashFlow, base time.Time, r float64) float64 {
	sum := 0.0
	for _, f := range flows {
		years := f.Date.Sub(base).Hours() / 24.0 / 365.0 // Actual/365 — Excel XIRR day-count convention
		sum += f.Amount / math.Pow(1+r, years)
	}
	return sum
}

// npvDerivative computes dNPV/dr.
func npvDerivative(flows []CashFlow, base time.Time, r float64) float64 {
	sum := 0.0
	for _, f := range flows {
		years := f.Date.Sub(base).Hours() / 24.0 / 365.0
		sum += -years * f.Amount / math.Pow(1+r, years+1)
	}
	return sum
}
