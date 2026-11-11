package t

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestParams(t *testing.T) {
	if got, want := Params(), core.DefaultParams(); got != want {
		t.Fatalf("Params() = %+v, want %+v", got, want)
	}
}
