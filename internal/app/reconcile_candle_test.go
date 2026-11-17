package app

import (
	"testing"

	densitylive "tinvest/internal/service/screener/density/live"
)

func TestCheckClosedInstrument(t *testing.T) {
	for _, tc := range []struct {
		name    string
		account densitylive.Account
		wantErr bool
	}{
		{name: "flat"},
		{name: "inventory", account: densitylive.Account{Holdings: map[string]densitylive.Holding{"uid": {Units: 1}}}, wantErr: true},
		{name: "active order", account: densitylive.Account{Orders: []densitylive.ActiveOrder{{InstrumentID: "uid"}}}, wantErr: true},
		{name: "active stop", account: densitylive.Account{Stops: map[string]string{"stop": "uid"}}, wantErr: true},
		{name: "unrelated order", account: densitylive.Account{Orders: []densitylive.ActiveOrder{{InstrumentID: "other"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkClosedInstrument(tc.account, "uid"); (err != nil) != tc.wantErr {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}
