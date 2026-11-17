package accountcoord

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestConfirmClosedCandle(t *testing.T) {
	for _, name := range []string{"confirmed", "broker mismatch", "entry changed", "in-flight exit", "density reservation", "density inventory"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			store := statestore.New(filepath.Join(root, "candle.json"))
			coord := New(root, "test-account", store)
			entered := time.Now().UTC()
			entry := statestore.Entry{Ticker: "SVCB", Strategy: "rsi_zone", EntryTime: entered, Quantity: 300}
			sibling := statestore.Entry{Ticker: "SOFL", Quantity: 10}
			if name == "in-flight exit" {
				entry.ExitRequestID = "unknown"
			}
			if err := store.Save(map[string]statestore.Entry{"SVCB": entry, "SOFL": sibling}); err != nil {
				t.Fatal(err)
			}
			lease, err := coord.Lock(ctx)
			if err != nil {
				t.Fatal(err)
			}
			owner := "rsi_zone"
			if name == "density reservation" {
				owner = "density"
			}
			claim := Claim{Ticker: "SVCB", InstrumentID: "uid", Owner: owner, Key: "pending"}
			if err := coord.Reserve(lease, claim); err != nil {
				t.Fatal(err)
			}
			if err := lease.Close(); err != nil {
				t.Fatal(err)
			}
			if name == "density inventory" {
				coord.SetDensitySnapshot(func(context.Context) (map[string]DensityPosition, error) {
					return map[string]DensityPosition{"SVCB": {InstrumentID: "uid", Quantity: 10}}, nil
				})
			}
			if name == "entry changed" {
				entered = entered.Add(-time.Minute)
			}
			checked := false
			err = coord.ConfirmClosedCandle(ctx, "SVCB", "uid", entered, func(context.Context) error {
				checked = true
				if name == "broker mismatch" {
					return errors.New("broker position is open")
				}
				return nil
			})
			if success := name == "confirmed"; (err == nil) != success {
				t.Fatalf("unexpected result: %v", err)
			}
			if checked != (name == "confirmed" || name == "broker mismatch") {
				t.Fatalf("unexpected broker verification: %v", checked)
			}
			state, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			claims, err := coord.Claims(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, hasEntry := state["SVCB"]
			if hasEntry != (name != "confirmed") || (len(claims) == 1) != hasEntry || state["SOFL"] != sibling {
				t.Fatalf("unexpected state: %v, claims: %v", state, claims)
			}
		})
	}
}
