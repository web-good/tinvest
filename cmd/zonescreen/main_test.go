package main

import "testing"

func TestValidateRejectsHoldoutNotShorterThanWindow(t *testing.T) {
	if err := validate(runCfg{months: 6, holdoutMonths: 6, workers: 1}); err == nil {
		t.Fatal("validate accepted -holdout-months == -months, want an error: the train window would be empty")
	}
}

func TestValidateRejectsZeroWorkers(t *testing.T) {
	if err := validate(runCfg{months: 36, holdoutMonths: 6, workers: 0}); err == nil {
		t.Fatal("validate accepted -workers 0, want an error")
	}
}

func TestValidateRejectsNegativeTickGate(t *testing.T) {
	if err := validate(runCfg{months: 36, holdoutMonths: 6, workers: 1, maxTickPct: -0.1}); err == nil {
		t.Fatal("validate accepted -max-tick-pct < 0, want an error (0 is the off switch)")
	}
}

func TestValidateAcceptsDefaults(t *testing.T) {
	if err := validate(runCfg{months: 36, holdoutMonths: 6, workers: 8, maxTickPct: 0.3}); err != nil {
		t.Fatalf("validate rejected defaults: %v", err)
	}
}
