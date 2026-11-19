package density

import "time"

// Price retains the exact quotation used to identify a level.
type Price struct {
	Units int64
	Nano  int32
}
type Level struct {
	Price    Price
	Quantity int64
}
type Book struct {
	InstrumentID string
	Lot          int64
	Bids, Asks   []Level
	Time         time.Time
}
type Signal struct {
	InstrumentID               string
	Price                      Price
	Quantity                   int64
	Rubles, Multiple, Distance float64
	Since                      time.Time
	Ticker                     string
	Age                        time.Duration
	TradeLots                  int64
	TradeCoverage              time.Duration
	MinRubles, MinLots         float64
	HasAverageVolume           bool
	SupportPrice               Price
	SupportTouches             int
	ProfileUntil               time.Time
}
type Config struct {
	RequireAverageVolume       bool
	TurnoverFraction           float64
	Support                    SupportConfig
	MinRubles, MinMultiple     float64
	MinNeighbors, MaxNeighbors int
	Persistence, Freshness     time.Duration
	MinTurnover, MaxLotPrice   float64
	MaxBooks                   int
	MaxTrades                  int
	BookDepth                  int32
	TradeReadiness             time.Duration
}

func DefaultConfig() Config {
	return Config{MinRubles: 5_000_000, MinMultiple: 5, MinNeighbors: 5, MaxNeighbors: 10, Persistence: time.Minute, Freshness: 3 * time.Second, MinTurnover: 100_000_000, MaxLotPrice: 30_000, MaxBooks: 80, MaxTrades: 10, BookDepth: 50, TradeReadiness: 15 * time.Second}
}
