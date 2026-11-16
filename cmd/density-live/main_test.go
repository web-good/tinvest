package main

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"
)

func TestHelpNeedsNoTokenOrAccount(t *testing.T) {
	t.Setenv("T_BANK", "")
	t.Setenv("DENSITY_ACCOUNT_ID", "")
	var out bytes.Buffer
	if err := run(context.Background(), []string{"-help"}, &out); err != flag.ErrHelp {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "trade") || !strings.Contains(out.String(), "account") {
		t.Fatal("missing live flags")
	}
}

func TestStandaloneTradeCannotBypassSharedAccountCoordinator(t *testing.T) {
	var out bytes.Buffer
	err := run(context.Background(), []string{"-trade"}, &out)
	if err == nil || !strings.Contains(err.Error(), "shared") {
		t.Fatalf("standalone live still available: %v", err)
	}
}
func TestInvalidLiveOptionsFailBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{{"-rr", "NaN"}, {"-entry-ticks", "0"}, {"-duration", "-1s"}, {"-tickers", ""}} {
		var out bytes.Buffer
		err := run(context.Background(), args, &out)
		if err == nil || strings.Contains(err.Error(), "T_BANK") {
			t.Fatalf("%v: %v", args, err)
		}
	}
}
func TestMissingExplicitAccountNeverSelectsAnotherStrategyAccount(t *testing.T) {
	t.Setenv("T_BANK", "test")
	t.Setenv("DENSITY_ACCOUNT_ID", "")
	t.Setenv("RSI_PULLBACK_ACCOUNT_ID", "must-not-be-used")
	var out bytes.Buffer
	if err := run(context.Background(), nil, &out); err == nil || !strings.Contains(err.Error(), "DENSITY_ACCOUNT_ID") {
		t.Fatal(err)
	}
}
