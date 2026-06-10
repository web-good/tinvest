package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/momentum/strategy/core"
	momentumgazp "tinvest/internal/service/trading_strategy/momentum/strategy/gazp"
	momentummdmg "tinvest/internal/service/trading_strategy/momentum/strategy/mdmg"
	momentumnvtk "tinvest/internal/service/trading_strategy/momentum/strategy/nvtk"
	momentumsber "tinvest/internal/service/trading_strategy/momentum/strategy/sber"
)

func TestMomentumLookupRegisteredRUAL(t *testing.T) {
	b := MomentumLookupOrGeneric("RUAL")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	want := core.Params{
		EMAPeriod: 100, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.0, DailyATRPeriod: 14, MaxDailyATRUsed: 0.7,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 1.0, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 20,
	}
	if got != want {
		t.Fatalf("RUAL defaults = %+v\nwant %+v", got, want)
	}
	if s := b.Build(got); s.Ticker() != "RUAL" {
		t.Fatalf("ticker=%q want RUAL", s.Ticker())
	}
}

func TestMomentumLookupRegisteredAFKS(t *testing.T) {
	b := MomentumLookupOrGeneric("AFKS")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	want := core.Params{
		EMAPeriod: 200, MACDFast: 10, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.0, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 1.0, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
	if got != want {
		t.Fatalf("AFKS defaults = %+v\nwant %+v", got, want)
	}
	if s := b.Build(got); s.Ticker() != "AFKS" {
		t.Fatalf("ticker=%q want AFKS", s.Ticker())
	}
}

func TestMomentumLookupGenericFallback(t *testing.T) {
	b := MomentumLookupOrGeneric("UNKNOWN")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "UNKNOWN" {
		t.Fatalf("ticker=%q want UNKNOWN", s.Ticker())
	}
	// ParseParams must layer the override on top of genericMomentumDefaults, not a zero struct.
	got, err := b.ParseParams([]byte(`{"SLMult": 1.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.SLMult != 1.5 {
		t.Fatalf("SLMult=%f want 1.5 (override)", p.SLMult)
	}
	if p.EMAPeriod != 200 || p.MACDSlow != 26 {
		t.Fatalf("generic defaults not preserved: EMAPeriod=%d MACDSlow=%d want 200/26", p.EMAPeriod, p.MACDSlow)
	}
}

func TestGenericMomentumDefaultsAreFrozenBaseline(t *testing.T) {
	want := core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
	if got := genericMomentumDefaults(); got != want {
		t.Fatalf("genericMomentumDefaults drifted = %+v\nwant %+v", got, want)
	}
}

func TestMomentumParseParamsPartialOverride(t *testing.T) {
	b := MomentumLookupOrGeneric("RUAL")
	got, err := b.ParseParams([]byte(`{"TakeProfitRR": 3.0}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.TakeProfitRR != 3.0 {
		t.Fatalf("TakeProfitRR=%f want 3.0 (override)", p.TakeProfitRR)
	}
	if p.MACDSlow != 26 {
		t.Fatalf("MACDSlow=%d want 26 (default kept)", p.MACDSlow)
	}
}

func TestMomentumLookupRegisteredYDEX(t *testing.T) {
	if _, ok := momentumRegistry["YDEX"]; !ok {
		t.Fatal("YDEX not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric("YDEX")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	want := core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
	if got != want {
		t.Fatalf("YDEX defaults = %+v\nwant %+v", got, want)
	}
	if s := b.Build(got); s.Ticker() != "YDEX" {
		t.Fatalf("ticker=%q want YDEX", s.Ticker())
	}
}

func TestMomentumLookupRegisteredPLZL(t *testing.T) {
	if _, ok := momentumRegistry["PLZL"]; !ok {
		t.Fatal("PLZL not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric("PLZL")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	want := core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
	if got != want {
		t.Fatalf("PLZL defaults = %+v\nwant %+v", got, want)
	}
	if s := b.Build(got); s.Ticker() != "PLZL" {
		t.Fatalf("ticker=%q want PLZL", s.Ticker())
	}
}

func TestMomentumLookupRegisteredSBER(t *testing.T) {
	if _, ok := momentumRegistry[momentumsber.Ticker]; !ok {
		t.Fatalf("SBER not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric(momentumsber.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentumsber.DefaultParams() {
		t.Fatalf("SBER defaults = %+v\nwant %+v", got, momentumsber.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "SBER" {
		t.Fatalf("ticker=%q want SBER", s.Ticker())
	}
}

func TestMomentumLookupRegisteredGAZP(t *testing.T) {
	if _, ok := momentumRegistry[momentumgazp.Ticker]; !ok {
		t.Fatalf("GAZP not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric(momentumgazp.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentumgazp.DefaultParams() {
		t.Fatalf("GAZP defaults = %+v\nwant %+v", got, momentumgazp.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "GAZP" {
		t.Fatalf("ticker=%q want GAZP", s.Ticker())
	}
}

func TestMomentumLookupRegisteredNVTK(t *testing.T) {
	if _, ok := momentumRegistry[momentumnvtk.Ticker]; !ok {
		t.Fatalf("NVTK not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric(momentumnvtk.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentumnvtk.DefaultParams() {
		t.Fatalf("NVTK defaults = %+v\nwant %+v", got, momentumnvtk.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "NVTK" {
		t.Fatalf("ticker=%q want NVTK", s.Ticker())
	}
}

func TestMomentumLookupRegisteredMDMG(t *testing.T) {
	if _, ok := momentumRegistry[momentummdmg.Ticker]; !ok {
		t.Fatalf("MDMG not registered in momentumRegistry")
	}
	b := MomentumLookupOrGeneric(momentummdmg.Ticker)
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != momentummdmg.DefaultParams() {
		t.Fatalf("MDMG defaults = %+v\nwant %+v", got, momentummdmg.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "MDMG" {
		t.Fatalf("ticker=%q want MDMG", s.Ticker())
	}
}
