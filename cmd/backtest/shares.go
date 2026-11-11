package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	grpcclient "tinvest/pkg/client/grpc"
)

// Cache only instrument identifiers and lot sizes, not prices or trading status.
// Refresh daily so changes to instrument metadata do not persist indefinitely.
// Removing data/candles/shares.json forces a refresh on the next run.
const shareCacheTTL = 24 * time.Hour

func resolveShare(ctx context.Context, client grpcclient.GrpcClient, ticker string) (shareInfo, error) {
	path := filepath.Join(cacheDir, "shares.json")
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) >= 0 && time.Since(info.ModTime()) < shareCacheTTL {
		if raw, err := os.ReadFile(path); err == nil {
			var cached map[string]shareInfo
			if json.Unmarshal(raw, &cached) == nil {
				if s := cached[ticker]; s.ID != "" && s.Lot > 0 {
					return s, nil
				}
			}
		}
	}

	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return shareInfo{}, fmt.Errorf("load shares: %w", err)
	}
	cached := make(map[string]shareInfo, len(shares))
	for _, s := range shares {
		if s == nil || s.Ticker == "" || s.ID == "" || s.Lot <= 0 {
			continue
		}
		// Preserve the original resolver's first-match policy for duplicate tickers.
		if _, exists := cached[s.Ticker]; !exists {
			cached[s.Ticker] = shareInfo{ID: s.ID, Lot: s.Lot}
		}
	}
	s, ok := cached[ticker]
	if !ok {
		return shareInfo{}, fmt.Errorf("ticker %q not found in Shares()", ticker)
	}
	if err := writeShareCache(path, cached); err != nil {
		log.Printf("backtest: cannot cache shares: %v", err)
	}
	return s, nil
}

// Rename a complete temporary file so concurrent CLI runs cannot read partial JSON.
func writeShareCache(path string, shares map[string]shareInfo) error {
	raw, err := json.Marshal(shares)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".shares-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
