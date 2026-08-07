// Package reconstruct rebuilds rsi_pullback entry-state from the broker API when the local
// state file is missing but a position is open. EntryPrice uses the broker's average
// purchase price; EntryTime is the most recent BUY fill; EntryATR is the DAILY ATR over
// weekday dailies completed before the entry day's MSK midnight — the same unit the core
// used to size the stop and the target at entry; TakeProfit is rebuilt from it and the
// ticker's TPDailyATR; MaxFav is the highest 30-minute CLOSE since entry.
package reconstruct

import (
	"context"
	"fmt"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

// tradesLookbackDays bounds the fills request. rsi_pullback holds multi-day: a position
// entered before a long holiday stretch and carried on a wide trail can easily be a couple
// of months old, and a window that ends before its BUY fill would report "no BUY fill" for a
// position that is perfectly normal. Half a year costs one wider request and removes that
// class of false failure.
const tradesLookbackDays = 180

// dailyFetchDays sizes the daily request for the ATR, and it is deliberately the same 730
// days marketdata.Assemble uses — for the same reason. indicators.ATRSeries seeds its
// recursion with the average TR of the first `period` bars and decays that seed by
// (period-1)/period per step; a short window leaves the seed weighted enough to shift the
// daily ATR by a few percent. Here that error would be worse than in the assembler: the
// rebuilt EntryATR must match the ATR the core froze at entry, and the stop, the target and
// both day-gate thresholds are multiples of it. Two years decay the seed to ~1e-16, so the
// reconstruction and the entry converge on the same number.
const dailyFetchDays = 730

// m30ChunkDays bounds one 30-minute request the same way the assembler does — the API caps a
// 30-minute window at roughly three weeks.
const m30ChunkDays = 14

// maxM30Chunks bounds the walk back to the entry. 8 chunks cover ~16 weeks of holding, well
// past any position this strategy carries, and turn a data outage into an error rather than
// an endless loop.
const maxM30Chunks = 8

// mskLoc anchors the entry-day boundary to Moscow, the same clock the core's calendar rules
// use (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// TradesClient is the slice of the operations client reconstruct needs.
type TradesClient interface {
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Entry reconstructs the entry-state for an open position.
func Entry(ctx context.Context, tc TradesClient, cc candles.CandleClient,
	accountID, instrumentID, ticker string, purchasePrice float64,
	p core.Params, now time.Time) (statestore.Entry, error) {

	trades, err := tc.GetInstrumentTrades(ctx, accountID, instrumentID, now.AddDate(0, 0, -tradesLookbackDays), now)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: trades: %w", err)
	}
	var entryTime time.Time
	for _, tr := range trades {
		if tr.IsBuy && tr.Date.After(entryTime) {
			entryTime = tr.Date
		}
	}
	if entryTime.IsZero() {
		return statestore.Entry{}, fmt.Errorf("reconstruct: no BUY fill found for %s", ticker)
	}

	atr, err := dailyATRAtEntry(ctx, cc, instrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, err
	}
	if atr <= 0 {
		// Ядро мерит этим числом стоп, цель и обе границы гейта дня. Нулевой ATR
		// не «нейтрален» — он выдал бы стоп вплотную к цене входа; лучше отказаться
		// от реконструкции и оставить решение человеку.
		return statestore.Entry{}, fmt.Errorf("reconstruct: %s: cannot rebuild daily ATR as of %s", ticker, entryTime.Format(time.RFC3339))
	}

	maxFav, err := maxCloseSinceEntry(ctx, cc, instrumentID, entryTime, now, purchasePrice)
	if err != nil {
		return statestore.Entry{}, err
	}

	var takeProfit float64
	if p.TPDailyATR > 0 {
		takeProfit = purchasePrice + p.TPDailyATR*atr
	}

	return statestore.Entry{
		Ticker:     ticker,
		EntryTime:  entryTime,
		EntryPrice: purchasePrice,
		EntryATR:   atr,
		TakeProfit: takeProfit,
		MaxFav:     maxFav,
	}, nil
}

// dailyATRAtEntry recomputes the daily ATR exactly as the core did on the entry bar: only
// dailies that closed BEFORE the entry day's MSK midnight (the engine never shows a running
// day), and only weekday ones — MOEX weekend sessions are 3-4x narrower and leaving them in
// understates the ATR by 9-16%.
func dailyATRAtEntry(ctx context.Context, cc candles.CandleClient, instrumentID string,
	entryTime time.Time, period int) (float64, error) {

	if period <= 0 {
		return 0, nil
	}
	from := entryTime.AddDate(0, 0, -dailyFetchDays)
	// limit is nil, as in the assembler: the window already bounds the request, and an
	// explicit Limit is just another way to truncate differently from the backtest.
	raw, err := cc.GetCandles(ctx, &instrumentID, enum.Day1.ToNumberInvestAPI(),
		timestamppb.New(from), timestamppb.New(entryTime), nil, true)
	if err != nil {
		return 0, fmt.Errorf("reconstruct: daily candles: %w", err)
	}
	e := entryTime.In(mskLoc)
	bound := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, mskLoc)

	daily := candles.ToCandles(raw, true)
	sort.Slice(daily, func(a, b int) bool { return daily[a].Time.Before(daily[b].Time) })

	highs := make([]float64, 0, len(daily))
	lows := make([]float64, 0, len(daily))
	closes := make([]float64, 0, len(daily))
	for _, c := range daily {
		if !c.Time.Before(bound) {
			continue
		}
		if wd := c.Time.In(mskLoc).Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		highs = append(highs, c.High)
		lows = append(lows, c.Low)
		closes = append(closes, c.Close)
	}
	if len(closes) < period+1 {
		return 0, nil // caller turns this into a refusal
	}
	return indicators.ATR(highs, lows, closes, period), nil
}

// maxCloseSinceEntry walks the 30-minute series from the entry to now and returns the
// highest CLOSE, floored at the entry price. Closes, not highs: the engine marks the
// position to market on the bar close, so the trail must be measured against the same
// series — highs would place the trailing stop above where the core ever moved it.
func maxCloseSinceEntry(ctx context.Context, cc candles.CandleClient, instrumentID string,
	entryTime, now time.Time, purchasePrice float64) (float64, error) {

	maxFav := purchasePrice
	to := now
	for chunk := 0; chunk < maxM30Chunks && to.After(entryTime); chunk++ {
		from := to.AddDate(0, 0, -m30ChunkDays)
		if from.Before(entryTime) {
			from = entryTime
		}
		raw, err := cc.GetCandles(ctx, &instrumentID, enum.Minutes30.ToNumberInvestAPI(),
			timestamppb.New(from), timestamppb.New(to), nil, true)
		if err != nil {
			return 0, fmt.Errorf("reconstruct: 30m candles: %w", err)
		}
		for _, c := range candles.ToCandles(raw, true) {
			if !c.Time.Before(entryTime) && c.Close > maxFav {
				maxFav = c.Close
			}
		}
		to = from
	}
	return maxFav, nil
}
