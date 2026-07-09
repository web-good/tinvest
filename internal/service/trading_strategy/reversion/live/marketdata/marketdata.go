// Package marketdata assembles the live reversion MarketData snapshot from Tinkoff
// candles, reusing the backtest's AssembleMarketData so live and backtest build
// identical inputs. Only hourly + 4H are fetched: the reversion core reads neither
// daily series nor TodayHigh/Low, and computes ATR on the hourly window.
package marketdata

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
)

// CandleClient is the slice of the market-data client the assembler needs.
type CandleClient interface {
	GetCandles(ctx context.Context, instrumentUID *string, interval int32,
		from, to *timestamppb.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error)
}

// Conservative lower bounds on completed trading bars per calendar day on MOEX, used
// to size the fetch window so it always contains enough warm-up bars after weekends
// and holidays are excluded. Intentionally pessimistic: over-fetching is cheap, the
// snapshot trims to the exact lookback.
const (
	barsPerCalendarDayHourly = 6
	barsPerCalendarDayHTF    = 2
	warmupBufferFactor       = 3
)

// htfWarmupBars is the minimum number of completed 4H bars fetched for the HTF EMA.
// The backtest computes that EMA over full history, while live seeds it with an SMA
// over the fetched window, so a short window skews the gate: at period+20 bars the
// SMA seed still carries ~77% weight for EMA150 and the live gate decision diverged
// from the backtest on ~3.9% of NVTK bars (measured on 2y of cached 4H candles);
// at 280 bars that drops to ~1.3%. 280 always fits the 3-month API window — every
// 89-day span in the same data holds ≥300 4H bars — and its clamped Limit stays
// under the 700 cap.
const htfWarmupBars = 280

// candleRequestCap bounds a single GetCandles request. Tinkoff rejects (InvalidArgument
// 30014) any request whose window (from..to) or Limit exceeds the per-interval maximum;
// see the CandleInterval doc comments in internal/pb/v1/marketdata.pb.go. The pessimistic
// warm-up sizing below can blow these caps (e.g. NVTK's 150-period 4H EMA needs bars=170,
// producing a ~258-day window and limit 1020), so we clamp both to the interval's cap.
type candleRequestCap struct {
	spanMonths int   // maximum from..to span the API accepts
	maxLimit   int32 // maximum Limit the API accepts
}

var candleRequestCaps = map[enum.Interval]candleRequestCap{
	enum.Hour1: {spanMonths: 3, maxLimit: 2400},
	enum.Hour4: {spanMonths: 3, maxLimit: 700},
}

// ToCandles converts oldest-first API candles to domain candles. When completedOnly is
// true the still-forming trailing bar (IsComplete=false) is dropped.
func ToCandles(in []*imodel.CandleItemTechAnalyse, completedOnly bool) []backtest.Candle {
	out := make([]backtest.Candle, 0, len(in))
	for _, c := range in {
		if completedOnly && !c.IsComplete {
			continue
		}
		out = append(out, backtest.Candle{
			Time:   c.Time,
			Open:   utils.CombinePrice(c.Open.Units, c.Open.Nano),
			High:   utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:    utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close:  utils.CombinePrice(c.Close.Units, c.Close.Nano),
			Volume: c.Volume,
		})
	}
	return out
}

// fetchCompleted pulls `bars` completed candles of one interval ending at `now`,
// returning the last `bars` (oldest-first). It requests a calendar window generous
// enough to survive non-trading hours.
func fetchCompleted(ctx context.Context, c CandleClient, instrumentID string,
	interval enum.Interval, bars, barsPerDay int, now time.Time) ([]backtest.Candle, error) {
	if bars <= 0 {
		return nil, nil
	}
	calendarDays := bars/barsPerDay + 1
	calendarDays *= warmupBufferFactor
	from := now.AddDate(0, 0, -calendarDays)
	limit := int32(bars * warmupBufferFactor * 2)
	// Clamp the window and limit to the interval's API cap. The +1-day nudge past the
	// exact 3-month calendar boundary keeps us safely inside regardless of month lengths.
	if rc, ok := candleRequestCaps[interval]; ok {
		if earliest := now.AddDate(0, -rc.spanMonths, 1); from.Before(earliest) {
			from = earliest
		}
		if limit > rc.maxLimit {
			limit = rc.maxLimit
		}
	}
	raw, err := c.GetCandles(ctx, &instrumentID, interval.ToNumberInvestAPI(),
		timestamppb.New(from), timestamppb.New(now), &limit, true)
	if err != nil {
		return nil, err
	}
	completed := ToCandles(raw, true)
	if len(completed) > bars {
		completed = completed[len(completed)-bars:]
	}
	return completed, nil
}

// Assemble builds the MarketData snapshot. lookbackBars is the hourly window size
// (Strategy.Lookback()); htfEMAPeriod>0 triggers a 4H fetch warmed to that period.
// Position is left nil for the caller to set.
func Assemble(ctx context.Context, c CandleClient, instrumentID string,
	lookbackBars, htfEMAPeriod int, now time.Time) (strategy.MarketData, error) {
	window, err := fetchCompleted(ctx, c, instrumentID, enum.Hour1, lookbackBars, barsPerCalendarDayHourly, now)
	if err != nil {
		return strategy.MarketData{}, fmt.Errorf("reversion marketdata: hourly candles: %w", err)
	}
	if len(window) < lookbackBars {
		return strategy.MarketData{}, fmt.Errorf("reversion marketdata: %d completed hourly candles < lookback %d", len(window), lookbackBars)
	}

	var htf []backtest.Candle
	if htfEMAPeriod > 0 {
		// Warm the 4H EMA to htfWarmupBars (see its doc comment) so the live EMA tracks
		// the backtest's full-history EMA; period+20 covers hypothetical larger periods.
		htf, err = fetchCompleted(ctx, c, instrumentID, enum.Hour4,
			max(htfEMAPeriod+20, htfWarmupBars), barsPerCalendarDayHTF, now)
		if err != nil {
			return strategy.MarketData{}, fmt.Errorf("reversion marketdata: 4H candles: %w", err)
		}
	}

	cur := window[len(window)-1].Time
	return backtest.AssembleMarketData(window, nil, htf, cur), nil
}
