// Package rebuild — общие части реконструкции входа по данным брокера, когда локального
// стейта нет, а позиция открыта: поиск последней BUY-сделки и дневной ATR на день входа.
// Их используют раннеры rsi_pullback и rsi_zone; остальное (цель, MaxFav) у каждой стратегии своё.
package rebuild

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/pkg/indicators"
)

// TradesLookbackDays bounds the fills request. rsi_pullback holds multi-day: a position
// entered before a long holiday stretch and carried on a wide trail can easily be a couple
// of months old, and a window that ends before its BUY fill would report "no BUY fill" for a
// position that is perfectly normal. Half a year costs one wider request and removes that
// class of false failure.
const TradesLookbackDays = 180

// DailyFetchDays sizes the daily request for the ATR, and it is deliberately the same 730
// days marketdata.Assemble uses — for the same reason. indicators.ATRSeries seeds its
// recursion with the average TR of the first `period` bars and decays that seed by
// (period-1)/period per step; a short window leaves the seed weighted enough to shift the
// daily ATR by a few percent. Here that error would be worse than in the assembler: the
// rebuilt EntryATR must match the ATR the core froze at entry, and the stop, the target and
// both day-gate thresholds are multiples of it. Two years decay the seed to ~1e-16, so the
// reconstruction and the entry converge on the same number.
const DailyFetchDays = 730

// ErrNoBuyFill — в окне нет ни одной BUY-сделки по инструменту.
var ErrNoBuyFill = errors.New("no BUY fill found")

// mskLoc anchors the entry-day boundary to Moscow, the same clock the core's calendar rules
// use (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// LastBuyTime возвращает время последней BUY-сделки по инструменту за TradesLookbackDays.
func LastBuyTime(ctx context.Context, tc adapter.TradesClient, accountID, instrumentID string,
	now time.Time) (time.Time, error) {

	trades, err := tc.GetInstrumentTrades(ctx, accountID, instrumentID, now.AddDate(0, 0, -TradesLookbackDays), now)
	if err != nil {
		return time.Time{}, fmt.Errorf("trades: %w", err)
	}
	var entryTime time.Time
	for _, tr := range trades {
		if tr.IsBuy && tr.Date.After(entryTime) {
			entryTime = tr.Date
		}
	}
	if entryTime.IsZero() {
		return time.Time{}, ErrNoBuyFill
	}
	return entryTime, nil
}

// WeekdayDailyATRBefore recomputes the daily ATR exactly as the core did on the entry bar: only
// dailies that closed BEFORE the entry day's MSK midnight (the engine never shows a running
// day), and only weekday ones — MOEX weekend sessions are 3-4x narrower and leaving them in
// understates the ATR by 9-16%. При period <= 0 или нехватке баров возвращает 0, nil.
func WeekdayDailyATRBefore(ctx context.Context, cc candles.CandleClient, instrumentID string,
	entryTime time.Time, period int) (float64, error) {

	if period <= 0 {
		return 0, nil
	}
	from := entryTime.AddDate(0, 0, -DailyFetchDays)
	// limit is nil, as in the assembler: the window already bounds the request, and an
	// explicit Limit is just another way to truncate differently from the backtest.
	raw, err := cc.GetCandles(ctx, &instrumentID, enum.Day1.ToNumberInvestAPI(),
		timestamppb.New(from), timestamppb.New(entryTime), nil, true)
	if err != nil {
		return 0, fmt.Errorf("daily candles: %w", err)
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
