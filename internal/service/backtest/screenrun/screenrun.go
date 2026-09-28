// Package screenrun holds the I/O plumbing shared by the ticker screeners
// (cmd/pullscreen, cmd/zonescreen): the tradable-universe loader, the API token loader
// and the worker-pool pacing rules. The scoring itself lives in
// internal/service/backtest and stays pure.
package screenrun

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"tinvest/internal/model"
	grpcclient "tinvest/pkg/client/grpc"
)

const (
	// APIAddress is the Tinkoff Invest public API endpoint.
	APIAddress = "invest-public-api.tinkoff.ru:443"
	// CacheDir is the local candle cache every screener replays from.
	CacheDir = "data/candles"
)

// RefreshWorkerCap is the most concurrent tickers -refresh may run with. The candle
// provider (internal/service/backtest/candles.go) logs a failed chunk and keeps going on
// error, and Load(refresh=true) then writes whatever it managed to fetch back over the
// existing cache file — so a slow-down from rate limiting is not the risk, a partially
// overwritten 540MB local cache is. 8 workers sit around 26 req/s against market-data,
// well above what the API tolerates before chunks start failing.
const RefreshWorkerCap = 2

// EffectiveWorkers returns the worker count a screener should actually use, and whether
// it clamped the caller's request. Pure so the -refresh safety rule is unit-testable
// without standing up a gRPC client.
func EffectiveWorkers(requested int, refresh bool) (workers int, capped bool) {
	if refresh && requested > RefreshWorkerCap {
		return RefreshWorkerCap, true
	}
	return requested, false
}

// PauseAfterTicker idles a worker between tickers so a full-universe run does not
// pin every core for 20+ minutes. It reports whether the pause completed: a
// cancelled context abandons the sleep immediately, so Ctrl+C unwinds the pool at
// once instead of once per outstanding pause.
func PauseAfterTicker(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// ShareInfo is the per-ticker metadata a screener's worker pool needs.
type ShareInfo struct {
	Ticker            string
	Name              string
	ID                string
	Lot               int32
	MinPriceIncrement float64 // price tick in currency units; 0 when the API did not report one
}

// FilterShares keeps the tradable RUB shares, or — when only is non-empty — exactly the
// listed tickers (case-insensitive, regardless of currency or trading status).
func FilterShares(shares []*model.Share, only []string) []ShareInfo {
	want := make(map[string]bool, len(only))
	for _, t := range only {
		want[strings.ToUpper(t)] = true
	}
	var out []ShareInfo
	for _, s := range shares {
		if s == nil {
			continue
		}
		if len(want) > 0 {
			if !want[strings.ToUpper(s.Ticker)] {
				continue
			}
		} else if !strings.EqualFold(s.Currency, "rub") || !s.Trading {
			continue
		}
		out = append(out, ShareInfo{
			Ticker: s.Ticker, Name: s.Name, ID: s.ID, Lot: s.Lot, MinPriceIncrement: s.MinPriceIncrement,
		})
	}
	return out
}

// LoadUniverse returns the tradable RUB share universe, or just the requested tickers.
func LoadUniverse(ctx context.Context, client grpcclient.GrpcClient, only []string) ([]ShareInfo, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("load shares: %w", err)
	}
	universe := FilterShares(shares, only)
	if len(universe) == 0 {
		return nil, fmt.Errorf("no matching shares found")
	}
	return universe, nil
}

// SplitCSV splits a comma-separated flag value, dropping blanks.
func SplitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// LoadToken reads the T_BANK API token from the environment or the local env files.
func LoadToken() (string, error) {
	_ = godotenv.Load("./env/local.env")
	_ = godotenv.Load("./env/token.env")
	token := os.Getenv("T_BANK")
	if token == "" {
		return "", fmt.Errorf("T_BANK is not set (checked env + ./env/local.env, ./env/token.env)")
	}
	return token, nil
}
