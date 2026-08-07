// Package marketdata assembles the live rsi_pullback MarketData snapshot from Tinkoff
// candles, reusing the backtest's AssembleMarketData and TodayExtent so live and backtest
// build identical inputs. Two series are fetched: a 30-minute window (the strategy's
// working timeframe, sized by Strategy.Lookback) and daily candles (the unit of risk —
// the stop, the target and both thresholds of the day gate are multiples of the daily ATR).
// No 4H series: the core never reads it.
package marketdata

import (
	"context"
	"fmt"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// chunkDays bounds one 30-minute request. The API caps a 30-minute request window at
// roughly three weeks (see the CandleInterval doc comments in internal/pb/v1/marketdata.pb.go);
// 14 days keeps us clear of the cap the way the backtest's chunkDaysFor does.
const chunkDays = 14

// maxChunks bounds the walk backwards. UGLD's 403-bar window is about two calendar weeks
// of MOEX 30-minute bars, so two chunks normally suffice; eight leaves room for the New
// Year holidays and a halted instrument without turning a data outage into an endless loop.
const maxChunks = 8

// dailyFetchDays sizes the single daily request. This is NOT about how many dailies the
// gate/ATR window needs (DailyATRPeriod+1 = 15 weekday bars would do) — it is about
// matching the backtest's Wilder-seed convergence. indicators.ATRSeries seeds its
// recursion with the average TR of the FIRST period bars of whatever series it is given,
// then decays that seed by a (period-1)/period factor per step. The backtest feeds it
// dailies from a year before the run window (cmd/backtest/main.go's ~250 trading days of
// lead-in), so by any bar in the run the seed's weight is negligible. A 90-day live fetch
// gives ATR(14) only ~46 completed-weekday smoothing steps, leaving the seed at
// (13/14)^46 ≈ 3.3% weight — enough to shift the daily ATR by 1-3%, which the stop, the
// target and BOTH day-gate thresholds are multiples of, and the gate is a threshold
// comparison: any day whose range sits within that margin of the threshold flips relative
// to the backtest. 730 days (two years) decays the seed to ~1e-16: both builds converge on
// the same ATR. One request is enough because the daily interval allows windows up to six
// years.
const dailyFetchDays = 730

// Assemble builds the MarketData snapshot as of `now`. lookbackBars is the 30-minute window
// size (Strategy.Lookback()). Position is left nil for the caller to set.
func Assemble(ctx context.Context, c candles.CandleClient, instrumentID string,
	lookbackBars int, now time.Time) (strategy.MarketData, error) {

	window, err := fetch30m(ctx, c, instrumentID, lookbackBars, now)
	if err != nil {
		return strategy.MarketData{}, err
	}
	if len(window) < lookbackBars {
		// Короткое окно опаснее ошибки: ema.Compute на окне короче периода возвращает
		// нулевую серию, трендовый гейт молча закрывается на весь прогон, и раннер
		// выглядит работающим, ничего не торгуя.
		return strategy.MarketData{}, fmt.Errorf(
			"rsi_pullback marketdata: %d completed 30m candles < lookback %d", len(window), lookbackBars)
	}

	daily, err := fetchDaily(ctx, c, instrumentID, now)
	if err != nil {
		return strategy.MarketData{}, err
	}

	i := len(window) - 1
	md := backtest.AssembleMarketData(window, daily, nil, window[i].Time)
	md.TodayHigh, md.TodayLow = backtest.TodayExtent(window, i)
	return md, nil
}

// fetch30m walks backwards in chunkDays windows until it holds lookbackBars completed bars,
// then returns the last lookbackBars (oldest-first). Chunk boundaries may overlap, so bars
// are de-duplicated by open-time.
func fetch30m(ctx context.Context, c candles.CandleClient, instrumentID string,
	lookbackBars int, now time.Time) ([]backtest.Candle, error) {

	seen := make(map[int64]struct{})
	var all []backtest.Candle
	to := now
	for chunk := 0; chunk < maxChunks && len(all) < lookbackBars; chunk++ {
		from := to.AddDate(0, 0, -chunkDays)
		// limit is nil, exactly like the backtest's candle provider (internal/service/backtest/candles.go):
		// the request window is already bounded by chunkDays/the interval's API cap, and an
		// explicit Limit is just another way to silently truncate differently from the engine.
		raw, err := c.GetCandles(ctx, &instrumentID, enum.Minutes30.ToNumberInvestAPI(),
			timestamppb.New(from), timestamppb.New(to), nil, true)
		if err != nil {
			return nil, fmt.Errorf("rsi_pullback marketdata: 30m candles: %w", err)
		}
		for _, cd := range candles.ToCandles(raw, true) {
			key := cd.Time.UnixNano()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, cd)
		}
		to = from
	}
	sort.Slice(all, func(a, b int) bool { return all[a].Time.Before(all[b].Time) })
	if len(all) > lookbackBars {
		all = all[len(all)-lookbackBars:]
	}
	return all, nil
}

// fetchDaily pulls the completed daily series in one request. visibleDaily inside
// AssembleMarketData then drops anything not closed before the current bar's MSK midnight,
// and the core drops weekend sessions itself, so nothing is filtered here.
func fetchDaily(ctx context.Context, c candles.CandleClient, instrumentID string,
	now time.Time) ([]backtest.Candle, error) {

	from := now.AddDate(0, 0, -dailyFetchDays)
	// limit is nil for the same reason as fetch30m: the window already bounds the request.
	raw, err := c.GetCandles(ctx, &instrumentID, enum.Day1.ToNumberInvestAPI(),
		timestamppb.New(from), timestamppb.New(now), nil, true)
	if err != nil {
		return nil, fmt.Errorf("rsi_pullback marketdata: daily candles: %w", err)
	}
	out := candles.ToCandles(raw, true)
	// Порядок здесь — не косметика: visibleDaily режет серию ПРЕФИКСОМ до полуночи
	// текущего дня, а ATR Уайлдера — рекурсия по порядку баров. Единственный бар не на
	// своём месте либо обрежет хвост серии, либо перемешает шаги сглаживания, и дневной
	// ATR — множитель стопа, цели и обеих границ гейта дня — уедет молча. API отдаёт
	// oldest-first, поэтому обычно это no-op; сортировка делает инвариант явным, а не
	// подразумеваемым.
	sort.Slice(out, func(a, b int) bool { return out[a].Time.Before(out[b].Time) })
	return out, nil
}
