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

// m30Limit is the API's per-request cap on 30-minute candles.
const m30Limit int32 = 1200

// dailyFetchDays sizes the single daily request. DailyATRPeriod+1 = 15 completed WEEKDAY
// dailies is the most any registered ticker needs; 90 calendar days covers that with room
// for the January holidays, and one request is enough because the daily interval allows
// windows up to six years.
const dailyFetchDays = 90

// dailyLimit caps the daily request; 200 comfortably exceeds dailyFetchDays.
const dailyLimit int32 = 200

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
		limit := m30Limit
		raw, err := c.GetCandles(ctx, &instrumentID, enum.Minutes30.ToNumberInvestAPI(),
			timestamppb.New(from), timestamppb.New(to), &limit, true)
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
	limit := dailyLimit
	raw, err := c.GetCandles(ctx, &instrumentID, enum.Day1.ToNumberInvestAPI(),
		timestamppb.New(from), timestamppb.New(now), &limit, true)
	if err != nil {
		return nil, fmt.Errorf("rsi_pullback marketdata: daily candles: %w", err)
	}
	return candles.ToCandles(raw, true), nil
}
