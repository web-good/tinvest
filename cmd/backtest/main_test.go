package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	fvg "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestHTFCoverageLines_EmptySeriesErrors(t *testing.T) {
	leadInFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	lines, err := htfCoverageLines(nil, leadInFrom)

	if err == nil {
		t.Fatal("expected an error for an empty H1 series, got nil")
	}
	if lines != nil {
		t.Errorf("expected no lines alongside the error, got %v", lines)
	}
}

func TestHTFCoverageLines_HealthyHeadDoesNotWarn(t *testing.T) {
	leadInFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// The typical, healthy case: the first cached bar is the next session's open, a
	// few hours after leadInFrom — well inside htfLeadInGapTolerance.
	htf := []domain.Candle{
		{Time: leadInFrom.Add(12 * time.Hour), Close: 1},
		{Time: leadInFrom.Add(13 * time.Hour), Close: 2},
	}

	lines, err := htfCoverageLines(htf, leadInFrom)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected only the coverage line for a healthy head, got %v", lines)
	}
	for _, l := range lines {
		if strings.Contains(l, "⚠️") {
			t.Errorf("healthy head must not warn, got line: %q", l)
		}
	}
}

func TestHTFCoverageLines_ShortHeadWarns(t *testing.T) {
	leadInFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// A genuinely starved cache: the series starts months after the requested lead-in.
	htf := []domain.Candle{
		{Time: leadInFrom.AddDate(0, 3, 0), Close: 1},
		{Time: leadInFrom.AddDate(0, 3, 0).Add(time.Hour), Close: 2},
	}

	lines, err := htfCoverageLines(htf, leadInFrom)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected a coverage line and a warning line, got %v", lines)
	}
	if !strings.Contains(lines[1], "⚠️") {
		t.Errorf("expected the second line to be the warning, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "-refresh") {
		t.Errorf("expected the warning to mention -refresh, got %q", lines[1])
	}
}

func TestHTFCoverageLines_GapAtToleranceBoundaryWarns(t *testing.T) {
	leadInFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	htf := []domain.Candle{
		{Time: leadInFrom.Add(htfLeadInGapTolerance), Close: 1},
	}

	lines, err := htfCoverageLines(htf, leadInFrom)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected the boundary gap (== tolerance) to warn, got %v", lines)
	}
}

func TestHTFCoverageLines_GapJustUnderToleranceDoesNotWarn(t *testing.T) {
	leadInFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	htf := []domain.Candle{
		{Time: leadInFrom.Add(htfLeadInGapTolerance - time.Hour), Close: 1},
	}

	lines, err := htfCoverageLines(htf, leadInFrom)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected a gap just under tolerance to stay quiet, got %v", lines)
	}
}

func TestLoadParams_EmptyPathReturnsDefaults(t *testing.T) {
	type stubParams struct{ X int }
	b := svc.Binding{
		DefaultParams: func() any { return stubParams{X: 7} },
		ParseParams: func(raw []byte) (any, error) {
			t.Fatal("ParseParams must not be called when paramsPath is empty")
			return nil, nil
		},
	}

	params, err := loadParams(b, "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.(stubParams).X != 7 {
		t.Errorf("expected defaults to pass through unchanged, got %+v", params)
	}
}

func TestLoadParams_ReadsAndParsesFile(t *testing.T) {
	type stubParams struct{ X int }
	dir := t.TempDir()
	path := dir + "/params.json"
	if err := writeFile(path, `{"X":42}`); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	b := svc.Binding{
		DefaultParams: func() any { return stubParams{} },
		ParseParams: func(raw []byte) (any, error) {
			var p stubParams
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, err
			}
			return p, nil
		},
	}

	params, err := loadParams(b, path)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.(stubParams).X != 42 {
		t.Errorf("expected parsed params from file, got %+v", params)
	}
}

func TestLoadParams_MissingFileErrors(t *testing.T) {
	b := svc.Binding{
		DefaultParams: func() any { return struct{}{} },
		ParseParams: func(raw []byte) (any, error) {
			return raw, nil
		},
	}

	_, err := loadParams(b, "/no/such/file.json")

	if err == nil {
		t.Fatal("expected an error for a missing params file, got nil")
	}
}

// A missing/wrong CLI branch, interval bypass, or same-close fill breaks these routes.
func TestSelectStrategyFVGImbalance(t *testing.T) {
	for _, ticker := range []string{"GAZP", "UNKNOWN"} {
		t.Run(ticker, func(t *testing.T) {
			binding, nextOpen, err := selectStrategy(ticker, "fvg_imbalance", enum.Minutes30)
			if err != nil {
				t.Fatal(err)
			}
			params, ok := binding.DefaultParams().(fvg.Params)
			if !ok {
				t.Fatalf("expected FVG params, got %T", binding.DefaultParams())
			}
			strat := binding.Build(params)
			if _, ok := strat.(*fvg.Strategy); !ok {
				t.Fatalf("expected FVG core, got %T", strat)
			}
			if strat.Ticker() != ticker {
				t.Errorf("ticker = %q, want %q", strat.Ticker(), ticker)
			}
			if !nextOpen {
				t.Error("FVG retest must fill at the next open")
			}
		})
	}
}

func TestSelectStrategyFVGImbalanceRejectsNonM30(t *testing.T) {
	for _, interval := range []enum.Interval{enum.Minutes15, enum.Hour1, enum.Day1} {
		t.Run(interval.String(), func(t *testing.T) {
			_, _, err := selectStrategy("GAZP", "fvg_imbalance", interval)
			if err == nil || !strings.Contains(err.Error(), "FVG imbalance runs on Minutes30") {
				t.Fatalf("expected registry interval error, got %v", err)
			}
		})
	}
}
