package adapter

import (
	"testing"

	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestOwnerOfLegacyEntryIsPullback(t *testing.T) {
	if got := Owner(statestore.Entry{}); got != LegacyOwner {
		t.Fatalf("Owner(пусто) = %q, want %q", got, LegacyOwner)
	}
}

func TestOwnerKeepsExplicitStrategy(t *testing.T) {
	if got := Owner(statestore.Entry{Strategy: "rsi_zone"}); got != "rsi_zone" {
		t.Fatalf("Owner = %q, want rsi_zone", got)
	}
}
