// Package reconstruct rebuilds rsi_pullback entry-state from the broker API when the local
// state file is missing but a position is open. EntryPrice uses the broker's average
// purchase price; EntryTime is the most recent BUY fill; EntryATR is the DAILY ATR over
// weekday dailies completed before the entry day's MSK midnight — the same unit the core
// used to size the stop and the target at entry; TakeProfit is rebuilt from it and the
// ticker's TPDailyATR; MaxFav is the highest 30-minute CLOSE since entry.
package reconstruct

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// m30ChunkDays bounds one 30-minute request the same way the assembler does — the API caps a
// 30-minute window at roughly three weeks.
const m30ChunkDays = 14

// maxM30Chunks bounds the walk back to the entry. 8 chunks cover ~16 weeks of holding, well
// past any position this strategy carries, and turn a data outage into an error rather than
// an endless loop.
const maxM30Chunks = 8

// TradesClient is the slice of the operations client reconstruct needs.
type TradesClient interface {
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Entry reconstructs the entry-state for an open position.
func Entry(ctx context.Context, tc TradesClient, cc candles.CandleClient,
	accountID, instrumentID, ticker string, purchasePrice float64,
	p core.Params, now time.Time) (statestore.Entry, error) {

	entryTime, err := rebuild.LastBuyTime(ctx, tc, accountID, instrumentID, now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("reconstruct: no BUY fill found for %s", ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("reconstruct: %w", err)
	}

	atr, err := rebuild.WeekdayDailyATRBefore(ctx, cc, instrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: %w", err)
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
