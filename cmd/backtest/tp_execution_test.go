package main

import (
	"strings"
	"testing"
	"time"
	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
)

func TestTPExecutionReport(t *testing.T) {
	cfg := domain.Config{TPExecution: domain.TPExecution{Mode: "bar-close", SlippageBPS: 10}}
	meta := metaCommon("GAZP", enum.Minutes30, time.Now(), time.Now(), cfg)
	report := domain.RenderMarkdown(meta, domain.Metrics{}, nil, nil)
	if !strings.Contains(report, "bar-close") || !strings.Contains(report, "10 bps") {
		t.Fatalf("missing execution assumptions: %s", report)
	}
}

func TestTPExecutionRejectedBeforeIO(t *testing.T) {
	for _, tc := range []struct {
		name, strategy, screen string
		execution              domain.TPExecution
		want                   string
	}{
		{name: "mode", strategy: "rsi_pullback", execution: domain.TPExecution{Mode: "invalid"}, want: "-tp-fill"},
		{name: "foreign", strategy: "rsi_zone", execution: domain.TPExecution{Mode: "bar-close"}, want: "rsi_pullback"},
		{name: "screen", strategy: "rsi_pullback", screen: "GAZP", execution: domain.TPExecution{SlippageBPS: 5}, want: "screening"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := run("GAZP", tc.strategy, enum.Minutes30, 24, 100000, 1, 0.0005,
				"", "", "", 0, 0, 0, 0, 0, t.TempDir(), false, "", 0, tc.screen,
				false, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, tc.execution)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v want %s", err, tc.want)
			}
		})
	}
}
