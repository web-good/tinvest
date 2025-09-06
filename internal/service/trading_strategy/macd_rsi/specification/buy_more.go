package specification

type BuyMore struct {
	Diff float64
}

func (s *BuyMore) IsSatisfiedBy(purchasePrice float64, price float64) bool {
	if purchasePrice-purchasePrice*s.Diff > price {
		return true
	}

	return false
}
