package main

import (
	"testing"

	svc "tinvest/internal/service/backtest"
)

func okCfg() runCfg {
	return runCfg{months: 12, workers: 8, opts: svc.DefaultGapScreenOptions()}
}

func TestValidate(t *testing.T) {
	if err := validate(okCfg()); err != nil {
		t.Fatalf("valid cfg rejected: %v", err)
	}
	tickOff := okCfg()
	tickOff.opts.MaxTickPct = 0
	if err := validate(tickOff); err != nil {
		t.Fatalf("-max-tick-pct 0 rejected: %v", err)
	}
	bad := map[string]func(*runCfg){
		"months":           func(c *runCfg) { c.months = 0 },
		"workers":          func(c *runCfg) { c.workers = 0 },
		"days below 0":     func(c *runCfg) { c.opts.MinMorningDays = -0.1 },
		"days above 1":     func(c *runCfg) { c.opts.MinMorningDays = 1.1 },
		"morning turnover": func(c *runCfg) { c.opts.MinMorningTurnoverM = -1 },
		"tick":             func(c *runCfg) { c.opts.MaxTickPct = -0.1 },
		"daily turnover":   func(c *runCfg) { c.opts.MinTurnoverM = -1 },
	}
	for name, mut := range bad {
		c := okCfg()
		mut(&c)
		if err := validate(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
