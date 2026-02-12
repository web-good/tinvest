package pipeline

import (
	"time"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func Finder(doneCh chan struct{}, bonds []*pkgmodel.Bond, isOfz bool, dateFrom, dateTo time.Time) <-chan *pkgmodel.Bond {
	c := make(chan *pkgmodel.Bond)

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

			if isOfz == true && bond.Exchange != "moex_morning_evening_ofz" {
				continue
			}

			if isOfz == false && bond.Exchange == "moex_morning_evening_ofz" {
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
