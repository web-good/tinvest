package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"strings"
	"testing"
	"tinvest/internal/service/screener/density"
)

func TestHelpDoesNotNeedTokenOrConnect(t *testing.T) {
	t.Setenv("T_BANK", "")
	var out bytes.Buffer
	if err := run(context.Background(), []string{"-help"}, &out); err != flag.ErrHelp {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(out.String(), "commission") {
		t.Fatal("missing paper flags")
	}
}
func TestInvalidConfigurationFailsBeforeConnecting(t *testing.T) {
	t.Setenv("T_BANK", "")
	var out bytes.Buffer
	for _, args := range [][]string{{"-commission", "NaN"}, {"-instruments", "11"}, {"-cash", "-1"}} {
		if err := run(context.Background(), args, &out); err == nil || strings.Contains(err.Error(), "T_BANK") {
			t.Fatalf("%v: %v", args, err)
		}
	}
}
func TestMissingTokenFailsClearly(t *testing.T) {
	t.Setenv("T_BANK", "")
	var out bytes.Buffer
	if err := run(context.Background(), nil, &out); err == nil || !strings.Contains(err.Error(), "T_BANK") {
		t.Fatal(err)
	}
}

func TestCancellationDoesNotMaskReportingFailure(t *testing.T) {
	disk := errors.New("disk full")
	err := finishError(context.Canceled, disk, nil)
	if !errors.Is(err, disk) || errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	if err := finishError(context.Canceled, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestTLSCompatibilityIsExplicitAndVerifiesCertificates(t *testing.T) {
	normal := marketTLS(false)
	compat := marketTLS(true)
	if normal.InsecureSkipVerify || compat.InsecureSkipVerify || normal.MinVersion < tls.VersionTLS12 || compat.MinVersion < tls.VersionTLS12 {
		t.Fatal("TLS verification weakened")
	}
	if len(normal.CurvePreferences) != 0 {
		t.Fatal("default TLS unexpectedly changed")
	}
	if len(compat.CurvePreferences) != 2 || compat.CurvePreferences[0] != tls.X25519 || compat.CurvePreferences[1] != tls.CurveP256 {
		t.Fatal("missing compatible curves")
	}
}
func TestFilteredSourceUsesTargetedLookup(t *testing.T) {
	f := &targetedSource{}
	s := filteredSource{UniverseSource: f, tickers: map[string]bool{"T": true, "SBER": true}}
	got, err := s.Instruments(context.Background())
	if err != nil || f.bulkCalls != 0 || len(f.tickers) != 2 || f.tickers[0] != "SBER" || len(got) != 2 {
		t.Fatalf("%+v %+v %v", got, f, err)
	}
}

type targetedSource struct {
	bulkCalls int
	tickers   []string
}

func (f *targetedSource) Instruments(context.Context) ([]density.UniverseInput, error) {
	f.bulkCalls++
	return nil, nil
}
func (f *targetedSource) InstrumentsByTickers(_ context.Context, tickers []string) ([]density.UniverseInput, error) {
	f.tickers = tickers
	return make([]density.UniverseInput, len(tickers)), nil
}
func (f *targetedSource) Candles(context.Context, string) ([]density.DailyCandle, error) {
	return nil, nil
}
