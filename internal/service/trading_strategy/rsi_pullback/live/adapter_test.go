package live

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func closeStopParams() core.Params {
	p := core.DefaultParams()
	p.StopDailyATR = 0.7
	p.StopOnClose, p.CatStopMult = 1, 2
	return p
}

// Без режима биржевая заявка — прежний защитный уровень ядра.
func TestExchangeStopWithoutCloseModeIsTheProtectiveLevel(t *testing.T) {
	p := core.DefaultParams()
	p.StopDailyATR = 0.7
	e := statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100}
	gotLevel, gotReason := exchangeStop(p, e)
	wantLevel, wantReason := core.DesiredStop(p, 100, 10, 100)
	if gotLevel != wantLevel || gotReason != wantReason {
		t.Fatalf("exchangeStop = %v/%q, want %v/%q", gotLevel, gotReason, wantLevel, wantReason)
	}
}

// В режиме на бирже стоит только аварийный стоп: вход − CatStopMult×StopDailyATR×ATR.
func TestExchangeStopInCloseModeIsTheCatastropheStop(t *testing.T) {
	p := closeStopParams()
	level, reason := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100})
	if reason != "CATSL" || math.Abs(level-86) > 1e-9 {
		t.Fatalf("exchangeStop = %v/%q, want 86/CATSL", level, reason)
	}
}

// Трейл в режиме проверяется ядром по закрытию; биржевую заявку он не двигает.
func TestExchangeStopInCloseModeIgnoresTheTrail(t *testing.T) {
	p := closeStopParams()
	p.UseTrail, p.TrailDailyATR = 1, 0.5
	low, _ := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100})
	high, reason := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 130})
	if low != high || reason != "CATSL" {
		t.Fatalf("уровень при MaxFav 100 = %v, при 130 = %v/%q — трейл двигает аварийную заявку", low, high, reason)
	}
}

// Режим без аварийного множителя — заявки нет (такой литерал не пропустит сторожевой тест).
func TestExchangeStopInCloseModeWithoutCatHasNoOrder(t *testing.T) {
	p := closeStopParams()
	p.CatStopMult = 0
	if level, reason := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100}); level != 0 || reason != "" {
		t.Fatalf("exchangeStop = %v/%q, want 0/\"\"", level, reason)
	}
}
