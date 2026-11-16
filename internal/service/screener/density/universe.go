package density

import (
	"math"
	"sort"
	"strings"
	"time"
)

type Instrument struct {
	ID, Ticker string
	Lot        int64
	PriceStep  Price
	Turnover   float64
	ProfileAt  time.Time
	Supports   []Support
}
type DailyCandle struct {
	Time     time.Time
	Close    Price
	Volume   int64
	Complete bool
}
type UniverseInput struct {
	Instrument                Instrument
	Currency, Exchange        string
	BuyAvailable, TradingOpen bool
	Candles                   []DailyCandle
}

func eligible(in UniverseInput) bool {
	return in.Instrument.ID != "" && in.Instrument.Ticker != "" && in.Instrument.Lot > 0 && strings.EqualFold(in.Currency, "rub") && strings.EqualFold(in.Exchange, "moex") && in.BuyAvailable
}

// SelectUniverse uses only completed candles; the current trading status is handled by the stream.
func SelectUniverse(inputs []UniverseInput, cfg Config) []Instrument {
	var result []Instrument
	for _, in := range inputs {
		if !eligible(in) {
			continue
		}
		candles := append([]DailyCandle(nil), in.Candles...)
		sort.Slice(candles, func(i, j int) bool { return candles[i].Time.After(candles[j].Time) })
		count := 0
		total := 0.0
		lotPrice := 0.0
		for _, c := range candles {
			if !c.Complete || c.Time.IsZero() || !valid(c.Close) || c.Volume <= 0 {
				continue
			}
			p := value(c.Close) * float64(in.Instrument.Lot)
			turnover := p * float64(c.Volume)
			if math.IsInf(turnover, 0) || math.IsNaN(turnover) {
				continue
			}
			if count == 0 {
				lotPrice = p
			}
			total += turnover
			count++
			if count == 20 {
				break
			}
		}
		if count < 10 || lotPrice > cfg.MaxLotPrice || math.IsInf(total, 0) {
			continue
		}
		avg := total / float64(count)
		if avg < cfg.MinTurnover {
			continue
		}
		instrument := in.Instrument
		instrument.Turnover = avg
		result = append(result, instrument)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Turnover != result[j].Turnover {
			return result[i].Turnover > result[j].Turnover
		}
		return result[i].ID < result[j].ID
	})
	if len(result) > cfg.MaxBooks {
		result = result[:cfg.MaxBooks]
	}
	return result
}
