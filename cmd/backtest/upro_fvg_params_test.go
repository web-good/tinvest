package main

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestUPROFVGDefaultParams(t *testing.T) {
	binding, _, err := selectStrategy("UPRO", "fvg_imbalance", enum.Minutes30)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadParams(binding, "")
	if err != nil {
		t.Fatal(err)
	}
	want := core.Params{ATRPeriod: 14, MinGapATR: 0.4, MaxWaitBars: 30, StochPeriod: 10, UseStochExit: 0, StopATR: 0.5, TakeProfitATR: 1.5, HTFTrendEMA: 0, RequireEntryAboveFVGLower: 1}
	if got != want {
		t.Fatalf("default UPRO params = %+v, want %+v", got, want)
	}
	if err := core.ValidateParams(want); err != nil {
		t.Fatal(err)
	}
}
