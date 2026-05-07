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

			if dateTo.Before(bond.MaturityDate) || dateFrom.After(bond.MaturityDate) {
				continue
			}

			if bond.FloatingCouponFlag == true || bond.AmortizationFlag == true || bond.Nkd == 0 {
				continue
			}

			if isOfz == true && (reOfz.MatchString(bond.Name) == false && reRegion.MatchString(bond.Name) == false) {
				continue
			}

			if isOfz == false && (reOfz.MatchString(bond.Name) == true || reRegion.MatchString(bond.Name) == true) {
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
