package tbank

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestLiteral5mIsPinned(t *testing.T) {
	want := core.Params{
		MinDropPct:     0.9,
		MaxDropPct:     0,
		TargetBody:     0.5,
		MinTargetPct:   0,
		StopBody:       2.0,
		StartHour:      7,
		EODHour:        23,
		EODMinute:      0,
		TimeExit:       1,
		TimeExitBars:   4,
		DropSeries:     0,
		DropSeriesBars: 3,
		ConfirmBar:     0,
	}
	if got := DefaultParams5m(); got != want {
		t.Fatalf("литерал T 5m изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestLiteral15mIsPinned(t *testing.T) {
	want := core.Params{
		MinDropPct:     1.5,
		MaxDropPct:     0,
		TargetBody:     0.5,
		MinTargetPct:   0,
		StopBody:       0.75,
		StartHour:      7,
		EODHour:        23,
		EODMinute:      0,
		TimeExit:       1,
		TimeExitBars:   1,
		DropSeries:     0,
		DropSeriesBars: 3,
		ConfirmBar:     0,
	}
	if got := DefaultParams15m(); got != want {
		t.Fatalf("литерал T 15m изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams5m() == core.DefaultParams() || DefaultParams15m() == core.DefaultParams() {
		t.Fatal("T вернул core.DefaultParams(): тикер не должен отслеживать baseline")
	}
}

func TestLiteralsDifferPerTimeframe(t *testing.T) {
	if DefaultParams15m() == DefaultParams5m() {
		t.Fatal("литерал 15m совпал с 5m: у таймфрейма должны быть свои настройки")
	}
}

func TestLiteral1hIsPinned(t *testing.T) {
	want := core.Params{
		MinDropPct:     2.0,
		MaxDropPct:     0,
		TargetBody:     0.5,
		MinTargetPct:   0,
		StopBody:       1.5,
		StartHour:      7,
		EODHour:        23,
		EODMinute:      0,
		TimeExit:       1,
		TimeExitBars:   2,
		DropSeries:     0,
		DropSeriesBars: 3,
		ConfirmBar:     0,
	}
	if got := DefaultParams1h(); got != want {
		t.Fatalf("литерал T 1h изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestLiteral1hDiffersFromOtherTimeframes(t *testing.T) {
	if DefaultParams1h() == DefaultParams5m() || DefaultParams1h() == DefaultParams15m() {
		t.Fatal("литерал 1h совпал с литералом другого таймфрейма")
	}
}
