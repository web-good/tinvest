package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	domain "tinvest/internal/domain/backtest"
	svc "tinvest/internal/service/backtest"
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
