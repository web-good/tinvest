package main

import (
	"testing"

	"tinvest/internal/enum"
)

func validCfg() runCfg {
	return runCfg{interval: enum.Hour1, months: 36, holdoutMonths: 6, workers: 1, maxTickPct: 0.3, minTrades: 15, shortlist: 10}
}

func TestValidateAcceptsDefaults(t *testing.T) {
	if err := validate(validCfg()); err != nil {
		t.Fatalf("validate rejected defaults: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*runCfg){
		"holdout == months":     func(c *runCfg) { c.holdoutMonths = c.months },
		"zero workers":          func(c *runCfg) { c.workers = 0 },
		"negative tick gate":    func(c *runCfg) { c.maxTickPct = -0.1 },
		"negative min trades":   func(c *runCfg) { c.minTrades = -1 },
		"negative shortlist":    func(c *runCfg) { c.shortlist = -1 },
		"Minutes15 not screened": func(c *runCfg) { c.interval = enum.Minutes15 },
		"Day1 not screened":     func(c *runCfg) { c.interval = enum.Day1 },
	}
	for name, mutate := range cases {
		c := validCfg()
		mutate(&c)
		if err := validate(c); err == nil {
			t.Fatalf("%s: validate accepted it, want an error", name)
		}
	}
}

func TestParseInterval(t *testing.T) {
	for s, want := range map[string]enum.Interval{"Hour1": enum.Hour1} {
		got, err := parseInterval(s)
		if err != nil || got != want {
			t.Fatalf("parseInterval(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	for _, s := range []string{"Minutes5", "Minutes15", "Minutes30", "Day1", "1h", ""} {
		if _, err := parseInterval(s); err == nil {
			t.Fatalf("parseInterval(%q) accepted, want an error", s)
		}
	}
}

func TestPregated(t *testing.T) {
	cases := []struct {
		turnover, gate float64
		want           bool
	}{
		{49.9, 50, true},
		{50, 50, false},
		{50.1, 50, false},
		{0, 0, false},
	}
	for _, c := range cases {
		if got := pregated(c.turnover, c.gate); got != c.want {
			t.Errorf("pregated(%v, %v) = %v, want %v", c.turnover, c.gate, got, c.want)
		}
	}
}
