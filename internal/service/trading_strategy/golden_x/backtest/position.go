// Package backtest provides the per-share replay engine for the Golden X
// strategy. It is read-only against historical candles and does not touch
// the live trading services.
package backtest

import (
	"time"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

// ExitReason enumerates the cause of a closed trade (or partial exit).
type ExitReason string

const (
	ExitReasonSellP80 ExitReason = "sell_p80"
	ExitReasonSellP90 ExitReason = "sell_p90"
	ExitReasonSellP95 ExitReason = "sell_p95"
	ExitReasonStop    ExitReason = "stop"
	ExitReasonTimeout ExitReason = "timeout"
	ExitReasonOpen    ExitReason = "open" // marked-to-market at end of run
)

// Trade is a closed (or marked-to-market) backtest row. Each Gold partial
// exit is one Trade with Units ≈ 1/3; Growth exits produce a single Trade
// with Units = 1.0. ReturnPct uses the shared entry price.
type Trade struct {
	ShareID    string
	EntryDate  time.Time
	EntryPrice float64
	ExitDate   time.Time
	ExitPrice  float64
	Units      float64
	ExitReason ExitReason
	ReturnPct  float64
	WeeksHeld  int
}

// Position is an open backtest position. Gold positions track three sell
// flags (P80/P90/P95) that each consume 1/3 units. Growth positions track
// only P90 and consume the full 1.0 unit at the first trigger.
type Position struct {
	shareID    string
	entryDate  time.Time
	entryPrice float64
	stopPrice  float64 // 0 means no stop
	kind       dto.StrategyKind
	units      float64
	soldP80    bool
	soldP90    bool
	soldP95    bool
}

func OpenPosition(shareID string, entryDate time.Time, entryPrice float64, stop dto.Stop, kind dto.StrategyKind) *Position {
	return &Position{
		shareID:    shareID,
		entryDate:  entryDate,
		entryPrice: entryPrice,
		stopPrice:  stop.Price,
		kind:       kind,
		units:      1.0,
	}
}

func (p *Position) ShareID() string         { return p.shareID }
func (p *Position) EntryDate() time.Time    { return p.entryDate }
func (p *Position) EntryPrice() float64     { return p.entryPrice }
func (p *Position) StopPrice() float64      { return p.stopPrice }
func (p *Position) UnitsRemaining() float64 { return p.units }
func (p *Position) FullyClosed() bool       { return p.units <= 1e-9 }
func (p *Position) WeeksHeldAt(t time.Time) int {
	return int(t.Sub(p.entryDate).Hours() / (24 * 7))
}

// CloseAll emits a single Trade covering all remaining units. Used for
// stop hits, 52-week timeouts, and end-of-history mark-to-market.
func (p *Position) CloseAll(exitDate time.Time, exitPrice float64, reason ExitReason) Trade {
	units := p.units
	p.units = 0
	return Trade{
		ShareID:    p.shareID,
		EntryDate:  p.entryDate,
		EntryPrice: p.entryPrice,
		ExitDate:   exitDate,
		ExitPrice:  exitPrice,
		Units:      units,
		ExitReason: reason,
		ReturnPct:  (exitPrice - p.entryPrice) / p.entryPrice * 100,
		WeeksHeld:  p.WeeksHeldAt(exitDate),
	}
}

// EvaluateSellExits returns any partials triggered by the current week's
// close price + RSI vs the share's sell thresholds. For Gold: independent
// P80/P90/P95 checks — one partial per tier per call (so a single-week
// spike past P95 with no prior partials produces all three trades). For
// Growth: one full exit at the first P90 trigger.
func (p *Position) EvaluateSellExits(weekEnd time.Time, weekClose float64, st dto.SellThresholds, rsi float64) []Trade {
	if p.FullyClosed() {
		return nil
	}
	var out []Trade
	switch p.kind {
	case dto.StrategyKindGrowth:
		if !p.soldP90 && rsi > st.P90 {
			p.soldP90 = true
			units := p.units
			p.units = 0
			out = append(out, Trade{
				ShareID:    p.shareID,
				EntryDate:  p.entryDate,
				EntryPrice: p.entryPrice,
				ExitDate:   weekEnd,
				ExitPrice:  weekClose,
				Units:      units,
				ExitReason: ExitReasonSellP90,
				ReturnPct:  (weekClose - p.entryPrice) / p.entryPrice * 100,
				WeeksHeld:  p.WeeksHeldAt(weekEnd),
			})
		}
	case dto.StrategyKindDividend:
		third := 1.0 / 3.0
		if !p.soldP80 && rsi > st.P80 {
			p.soldP80 = true
			p.units -= third
			out = append(out, p.partialTrade(weekEnd, weekClose, ExitReasonSellP80, third))
		}
		if !p.soldP90 && rsi > st.P90 {
			p.soldP90 = true
			p.units -= third
			out = append(out, p.partialTrade(weekEnd, weekClose, ExitReasonSellP90, third))
		}
		if !p.soldP95 && rsi > st.P95 {
			p.soldP95 = true
			p.units = 0 // zero out to absorb 1/3 + 1/3 + 1/3 ≠ 1.0 rounding residual
			out = append(out, p.partialTrade(weekEnd, weekClose, ExitReasonSellP95, third))
		}
	}
	return out
}

func (p *Position) partialTrade(weekEnd time.Time, weekClose float64, reason ExitReason, units float64) Trade {
	return Trade{
		ShareID:    p.shareID,
		EntryDate:  p.entryDate,
		EntryPrice: p.entryPrice,
		ExitDate:   weekEnd,
		ExitPrice:  weekClose,
		Units:      units,
		ExitReason: reason,
		ReturnPct:  (weekClose - p.entryPrice) / p.entryPrice * 100,
		WeeksHeld:  p.WeeksHeldAt(weekEnd),
	}
}
