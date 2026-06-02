package scalping

import "tinvest/internal/service/trading_strategy/scalping/model"

// Candidate is the evaluated state of one instrument at decision time.
type Candidate struct {
	InstrumentID   string
	InstrumentName string
	Ticker         string
	Price          float64
	ATR            float64
	AboveEMA       bool
	RSIPrev        float64
	RSINow         float64
	HasPosition    bool
	PurchasePrice  float64
}

// Decide returns the trade signal for a candidate given the settings and the
// number of currently open positions. It is pure and side-effect free.
func Decide(c Candidate, s model.Settings, openCount int) model.Signal {
	base := model.Signal{
		InstrumentID:   c.InstrumentID,
		InstrumentName: c.InstrumentName,
		Ticker:         c.Ticker,
		Price:          c.Price,
		RSI:            c.RSINow,
	}

	if c.HasPosition {
		tp := c.PurchasePrice + s.AtrTakeProfitMult*c.ATR
		sl := c.PurchasePrice - s.AtrStopLossMult*c.ATR
		base.TakeProfit = tp
		base.StopLoss = sl
		switch {
		case c.Price >= tp:
			base.Kind = model.SignalSell
			base.Reason = "TP"
		case c.Price <= sl:
			base.Kind = model.SignalSell
			base.Reason = "SL"
		default:
			base.Kind = model.SignalNone
		}
		return base
	}

	if openCount >= s.MaxOpenPositions {
		base.Kind = model.SignalNone
		return base
	}

	if c.AboveEMA && c.RSIPrev < s.RsiReversalLevel && c.RSINow >= s.RsiReversalLevel {
		base.Kind = model.SignalBuy
		base.TakeProfit = c.Price + s.AtrTakeProfitMult*c.ATR
		base.StopLoss = c.Price - s.AtrStopLossMult*c.ATR
		return base
	}

	base.Kind = model.SignalNone
	return base
}
