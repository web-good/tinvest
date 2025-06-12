package model

type Share struct {
	ID                      string
	OriginalID              string
	Isin                    string
	Figi                    string
	Ticker                  string
	Type                    string
	Name                    string
	Currency                string
	Lot                     int32
	Country                 string
	Trading                 bool
	NKD                     float64
	NKDRub                  float64
	LastPrice               float64
	LastPriceRub            float64
	Nominal                 float64
	MinPriceIncrement       float64
	MinPriceIncrementAmount float64
}
