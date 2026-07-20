package pipeline

import (
	"regexp"
	"time"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func Finder(doneCh chan struct{}, bonds []*pkgmodel.Bond, isOfz bool, dateFrom, dateTo time.Time) <-chan *pkgmodel.Bond {
	c := make(chan *pkgmodel.Bond)
	reOfz := regexp.MustCompile(`ОФЗ`)
	reRegion := regexp.MustCompile(`Реги`)

	go func() {
		defer close(c)
		for _, bond := range bonds {
			time.Sleep(100 * time.Millisecond)

			if !PassesReliability(bond) {
				continue
			}

			if dateTo.Before(bond.MaturityDate) || dateFrom.After(bond.MaturityDate) {
				continue
			}

			if bond.FloatingCouponFlag || bond.AmortizationFlag || bond.Nkd == 0 {
				continue
			}

			if isOfz && (!reOfz.MatchString(bond.Name) && !reRegion.MatchString(bond.Name)) {
				continue
			}

			if !isOfz && (reOfz.MatchString(bond.Name) || reRegion.MatchString(bond.Name)) {
				continue
			}

			select {
			case <-doneCh:
				return
			case c <- bond:
			}
		}
	}()

	return c
}
