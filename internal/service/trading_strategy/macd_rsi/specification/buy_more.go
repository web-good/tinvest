package specification

type BuyMore struct {
	Diff float64
}

func (s *BuyMore) IsSatisfiedBy(purchasePrice float64, price float64) bool {
	return purchasePrice-purchasePrice*s.Diff > price
}
