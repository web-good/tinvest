package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

func TestGAZPParams(t *testing.T) {
	if err := core.ValidateParams(Params()); err != nil {
		t.Fatalf("invalid GAZP params: %v", err)
	}
}
