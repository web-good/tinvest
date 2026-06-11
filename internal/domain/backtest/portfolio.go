package backtest

import (
	"math"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// portfolio is the long-only mock account the engine trades against.
type portfolio struct {
	cfg          Config
	cash         float64
	qty          int64
	entryPrice   float64
	entryTime    time.Time
	entryBar     int
	entryLevel   float64 // support level captured at entry
	entryTarget  float64 // resistance/target captured at entry
	entryATR     float64 // ATR captured at entry
	entryStop    float64 // hard stop frozen at entry
	entryReason  string  // human-readable entry rationale captured at entry
	maxFavorable float64 // highest close seen since entry (monotonic)
	bar          int     // current bar index, set by the engine each iteration
}

func newPortfolio(cfg Config) *portfolio {
	return &portfolio{cfg: cfg, cash: cfg.InitialCash}
}

// open deploys cfg.Fraction of cash into whole lots at price. No-op if already
// in a position or if there is not enough cash for a single lot.
func (p *portfolio) open(price float64, t time.Time, level, target, atr, stop float64, entryReason string) {
	if p.qty != 0 {
		return
	}
	lotCost := price * float64(p.cfg.Lot) * (1 + p.cfg.Commission)
	if lotCost <= 0 {
		return
	}
	budget := p.cfg.Fraction * p.cash
	lots := int64(math.Floor(budget / lotCost))
	if lots <= 0 {
		return
	}
	qty := lots * int64(p.cfg.Lot)
	cost := float64(qty) * price
	commission := cost * p.cfg.Commission
	p.cash -= cost + commission
	p.qty = qty
	p.entryPrice = price
	p.entryTime = t
	p.entryBar = p.bar
	p.entryLevel = level
	p.entryTarget = target
	p.entryATR = atr
	p.entryStop = stop
	p.entryReason = entryReason
	p.maxFavorable = price
}

// mark raises the running favourable-price maximum toward price. No-op when flat
// or when price is not a new high, which keeps maxFavorable monotonic.
func (p *portfolio) mark(price float64) {
	if p.qty == 0 {
		return
	}
	if price > p.maxFavorable {
		p.maxFavorable = price
	}
}

// close sells the whole position at price and returns the round-trip trade.
func (p *portfolio) close(price float64, t time.Time, reason, exitReason string) Trade {
	revenue := float64(p.qty) * price
	commission := revenue * p.cfg.Commission
	p.cash += revenue - commission
	entryCost := float64(p.qty) * p.entryPrice * (1 + p.cfg.Commission)
	pnl := (revenue - commission) - entryCost
	pnlPct := 0.0
	if entryCost > 0 {
		pnlPct = pnl / entryCost
	}
	tr := Trade{
		EntryTime:       p.entryTime,
		EntryPrice:      p.entryPrice,
		ExitTime:        t,
		ExitPrice:       price,
		Quantity:        p.qty,
		Reason:          reason,
		PnL:             pnl,
		PnLPct:          pnlPct,
		BarsHeld:        p.bar - p.entryBar,
		SupportLevel:    p.entryLevel,
		ResistanceLevel: p.entryTarget,
		ATR:             p.entryATR,
		EntryReason:     p.entryReason,
		ExitReason:      exitReason,
	}
	p.qty = 0
	p.entryPrice = 0
	p.entryLevel = 0
	p.entryTarget = 0
	p.entryATR = 0
	p.entryStop = 0
	p.entryReason = ""
	p.maxFavorable = 0
	return tr
}

// strategyPosition exposes the open position to a strategy (nil when flat).
func (p *portfolio) strategyPosition() *strategy.Position {
	if p.qty == 0 {
		return nil
	}
	return &strategy.Position{
		PurchasePrice:     p.entryPrice,
		Quantity:          p.qty,
		StopLoss:          p.entryStop,
		EntryATR:          p.entryATR,
		MaxFavorablePrice: p.maxFavorable,
	}
}

// equity is cash plus the position marked at price.
func (p *portfolio) equity(price float64) float64 {
	return p.cash + float64(p.qty)*price
}
