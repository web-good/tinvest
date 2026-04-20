package domain

type Bond struct {
	Name     string
	FinalSum float64
	Percent  float64
}

type PortfolioBond struct {
	Bonds    []*Bond
	TotalSum float64
	Percent  float64
}
