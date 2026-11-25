package feed

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

type klineResponse struct {
	marketResult[[]string]
	Symbol string `json:"symbol"`
}

// SpotCloses reads complete candles from [from, to), in chronological order.
func (c *Client) SpotCloses(ctx context.Context, key model.Key, interval time.Duration, from, to time.Time) ([]model.CandleClose, error) {
	return c.spotCandlePrices(ctx, key, interval, from, to, 4)
}

// DayHigh returns the highest traded price in the 24 closed hours starting at from.
func (c *Client) DayHigh(ctx context.Context, key model.Key, from time.Time) (float64, error) {
	hours, err := c.spotCandlePrices(ctx, key, time.Hour, from, from.Add(24*time.Hour), 2)
	if err != nil {
		return 0, err
	}
	var high float64
	for _, hour := range hours {
		high = max(high, hour.Price)
	}
	return high, nil
}

func (c *Client) spotCandlePrices(ctx context.Context, key model.Key, interval time.Duration, from, to time.Time, field int) ([]model.CandleClose, error) {
	var closes []model.CandleClose
	err := c.readSpotCandles(ctx, key, interval, from, to, true, func(end time.Time, row []string) bool {
		price, err := nonnegative(row[field])
		if err != nil || price == 0 {
			return false
		}
		closes = append(closes, model.CandleClose{End: end, Price: price})
		return true
	})
	if err != nil {
		return nil, err
	}
	return closes, nil
}

// SpotM5 reads available closed five-minute candles from [from, to) oldest first.
func (c *Client) SpotM5(ctx context.Context, key model.Key, from, to time.Time) ([]model.M5Candle, error) {
	var candles []model.M5Candle
	err := c.readSpotCandles(ctx, key, 5*time.Minute, from, to, false, func(end time.Time, row []string) bool {
		if len(row) < 7 {
			return false
		}
		open, openErr := nonnegative(row[1])
		high, highErr := nonnegative(row[2])
		low, lowErr := nonnegative(row[3])
		volume, volumeErr := nonnegative(row[5])
		turnover, turnoverErr := nonnegative(row[6])
		if openErr != nil || highErr != nil || lowErr != nil || volumeErr != nil || turnoverErr != nil || low <= 0 || open < low || open > high {
			return false
		}
		candles = append(candles, model.M5Candle{End: end, Open: open, High: high, Low: low, Volume: volume, Turnover: turnover})
		return true
	})
	if err != nil {
		return nil, err
	}
	return candles, nil
}

func (c *Client) readSpotCandles(ctx context.Context, key model.Key, interval time.Duration, from, to time.Time, requireComplete bool, visit func(time.Time, []string) bool) error {
	const pageSize = 1000
	if key.Market != model.Spot || interval < time.Minute || interval%time.Minute != 0 ||
		!from.Before(to) || !from.Equal(from.Truncate(interval)) || !to.Equal(to.Truncate(interval)) {
		return fmt.Errorf("invalid spot candle interval or range")
	}
	for start := from; start.Before(to); {
		end := start.Add(pageSize * interval)
		if end.After(to) {
			end = to
		}
		query := url.Values{
			"category": {string(key.Market)}, "symbol": {key.Symbol},
			"interval": {strconv.FormatInt(int64(interval/time.Minute), 10)}, "limit": {strconv.Itoa(pageSize)},
			"start": {strconv.FormatInt(start.UnixMilli(), 10)}, "end": {strconv.FormatInt(end.UnixMilli()-1, 10)},
		}
		var result klineResponse
		if err := c.get(ctx, "/v5/market/kline", query, &result); err != nil {
			return err
		}
		if result.Category != key.Market || result.Symbol != key.Symbol {
			return fmt.Errorf("mismatched kline response for %s", key.Symbol)
		}
		if requireComplete && len(result.List) != int(end.Sub(start)/interval) {
			return fmt.Errorf("incomplete candle history for %s from %s to %s", key.Symbol, start, end)
		}
		next := start
		for i := len(result.List) - 1; i >= 0; i-- {
			row := result.List[i]
			if len(row) < 5 {
				return fmt.Errorf("invalid kline for %s at %s", key.Symbol, start)
			}
			stamp, timeErr := strconv.ParseInt(row[0], 10, 64)
			at := time.UnixMilli(stamp).In(start.Location())
			if timeErr != nil || at.Before(next) || !at.Before(end) || !at.Equal(at.Truncate(interval)) ||
				requireComplete && !at.Equal(next) || !visit(at.Add(interval), row) {
				return fmt.Errorf("invalid kline for %s at %s", key.Symbol, at)
			}
			next = at.Add(interval)
		}
		start = end
	}
	return nil
}

// SpotBar contains the OHLC prices of a completed spot candle.
type SpotBar struct {
	End                    time.Time
	Open, High, Low, Close float64
}

// SpotBars requires a continuous series of spot candles in [from, to).
func (c *Client) SpotBars(ctx context.Context, key model.Key, interval time.Duration, from, to time.Time) ([]SpotBar, error) {
	var bars []SpotBar
	err := c.readSpotCandles(ctx, key, interval, from, to, true, func(end time.Time, row []string) bool {
		var prices [4]float64
		for i := range prices {
			price, err := nonnegative(row[i+1])
			if err != nil || price <= 0 {
				return false
			}
			prices[i] = price
		}
		o, h, l, cl := prices[0], prices[1], prices[2], prices[3]
		if l > min(o, cl) || h < max(o, cl) {
			return false
		}
		bars = append(bars, SpotBar{end, o, h, l, cl})
		return true
	})
	if err != nil {
		return nil, err
	}
	return bars, nil
}
