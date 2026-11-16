package paper

import (
	"sort"
	"time"
)

type Summary struct {
	At              time.Time `json:"at"`
	Mode            string    `json:"mode"`
	InitialCash     float64   `json:"initial_cash"`
	Cash            float64   `json:"cash"`
	ReservedCash    float64   `json:"reserved_cash"`
	RealizedPnL     float64   `json:"realized_pnl"`
	UnrealizedPnL   *float64  `json:"estimated_unrealized_pnl"`
	Equity          *float64  `json:"estimated_equity"`
	NetPnL          *float64  `json:"estimated_net_pnl"`
	Fees            float64   `json:"paid_fees"`
	MaxDrawdown     float64   `json:"max_drawdown_rubles"`
	ClosedTrades    int       `json:"closed_trades"`
	WinningTrades   int       `json:"winning_trades"`
	WinRate         float64   `json:"win_rate"`
	CanceledEntries int       `json:"canceled_entries"`
	PendingOrders   int       `json:"pending_orders"`
	OpenPositions   int       `json:"open_positions"`
	DataGaps        int       `json:"data_gaps"`
	DataComplete    bool      `json:"data_complete"`
	ValuationFresh  bool      `json:"valuation_fresh"`
	Orders          []Order   `json:"orders"`
}

func (e *Engine) Summary(now time.Time) Summary {
	s := Summary{At: now, Mode: "paper", InitialCash: e.cfg.InitialCash, Cash: e.cash, ReservedCash: e.reserved(), RealizedPnL: e.realized, Fees: e.fees, MaxDrawdown: e.maxDrawdown, ClosedTrades: e.closed, WinningTrades: e.wins, CanceledEntries: e.canceled, DataGaps: e.gaps, DataComplete: e.complete, ValuationFresh: true, Orders: []Order{}}
	equity, unrealized := e.cash, 0.0
	ids := make([]string, 0, len(e.orders))
	for id := range e.orders {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		o := e.orders[id]
		s.Orders = append(s.Orders, *o)
		if o.State == "pending" {
			s.PendingOrders++
		}
		if o.OpenLots <= 0 {
			continue
		}
		s.OpenPositions++
		b, ok := e.books[id]
		remaining := o.OpenLots
		value := 0.
		if !ok || !e.fresh(id, now) {
			s.ValuationFresh = false
		}
		if ok {
			for _, l := range b.book.Bids {
				lots := min(remaining, l.Quantity)
				value += rub(price(l.Price)) * float64(lots) * float64(o.Lot)
				remaining -= lots
				if remaining == 0 {
					break
				}
			}
		}
		// Do not invent a bid for inventory beyond visible depth; valuation is flagged unavailable.
		if remaining > 0 {
			s.ValuationFresh = false
		}
		netValue := value * (1 - e.cfg.Commission)
		equity += netValue
		unrealized += netValue - rub(o.Entry)*float64(o.OpenLots)*float64(o.Lot) - o.EntryFees*float64(o.OpenLots)/float64(o.FilledLots)
	}
	if s.ValuationFresh {
		net := equity - s.InitialCash
		s.Equity = &equity
		s.UnrealizedPnL = &unrealized
		s.NetPnL = &net
	}
	if e.closed > 0 {
		s.WinRate = float64(e.wins) / float64(e.closed)
	}
	return s
}
func (e *Engine) mark(now time.Time) {
	s := e.Summary(now)
	if !s.ValuationFresh {
		return
	}
	if *s.Equity > e.peak {
		e.peak = *s.Equity
	}
	if loss := e.peak - *s.Equity; loss > e.maxDrawdown {
		e.maxDrawdown = loss
	}
}
