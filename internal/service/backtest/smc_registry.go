package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	smcafks "tinvest/internal/service/trading_strategy/smc/strategy/afks"
	"tinvest/internal/service/trading_strategy/smc/strategy/core"
	smcgazp "tinvest/internal/service/trading_strategy/smc/strategy/gazp"
	smcmdmg "tinvest/internal/service/trading_strategy/smc/strategy/mdmg"
	smcnvtk "tinvest/internal/service/trading_strategy/smc/strategy/nvtk"
	smcplzl "tinvest/internal/service/trading_strategy/smc/strategy/plzl"
	smcrusal "tinvest/internal/service/trading_strategy/smc/strategy/rusal"
	smcsber "tinvest/internal/service/trading_strategy/smc/strategy/sber"
	smcydex "tinvest/internal/service/trading_strategy/smc/strategy/ydex"
)

// smcBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All SMC tickers share the core engine; only ticker + defaults differ.
func smcBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	}
}

var smcRegistry = map[string]Binding{
	smcsber.Ticker:  smcBindingFor(smcsber.Ticker, smcsber.DefaultParams),
	smcgazp.Ticker:  smcBindingFor(smcgazp.Ticker, smcgazp.DefaultParams),
	smcnvtk.Ticker:  smcBindingFor(smcnvtk.Ticker, smcnvtk.DefaultParams),
	smcplzl.Ticker:  smcBindingFor(smcplzl.Ticker, smcplzl.DefaultParams),
	smcydex.Ticker:  smcBindingFor(smcydex.Ticker, smcydex.DefaultParams),
	smcafks.Ticker:  smcBindingFor(smcafks.Ticker, smcafks.DefaultParams),
	smcrusal.Ticker: smcBindingFor(smcrusal.Ticker, smcrusal.DefaultParams),
	smcmdmg.Ticker:  smcBindingFor(smcmdmg.Ticker, smcmdmg.DefaultParams),
}

// genericSMCDefaults are neutral baseline params for tickers without a dedicated
// SMC config. Intentionally independent of any per-ticker defaults so calibrating
// one ticker never drifts the generic baseline.
func genericSMCDefaults() core.Params {
	return core.Params{SwingK: 3, ReclaimBars: 4, Buffer: 0.5, TPR: 2, MaxHoldDays: 3}
}

// SMCLookupOrGeneric returns the registered SMC binding for a ticker, or a
// generic binding bound to that ticker (with genericSMCDefaults) when none is
// registered.
func SMCLookupOrGeneric(ticker string) Binding {
	if b, ok := smcRegistry[ticker]; ok {
		return b
	}
	return smcBindingFor(ticker, genericSMCDefaults)
}
