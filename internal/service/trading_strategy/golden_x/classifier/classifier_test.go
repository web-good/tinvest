package classifier

import (
	"testing"

	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
)

func TestSignalScore_AddsFundamentalBonus(t *testing.T) {
	base := gxmodel.ShareResult{BuyTier: gxmodel.TierGreen, TrendStatus: gxmodel.TrendWith}
	withBonus := base
	withBonus.FundamentalBonus = 3

	if signalScore(withBonus)-signalScore(base) != 3 {
		t.Fatalf("bonus delta = %d, want 3", signalScore(withBonus)-signalScore(base))
	}
}
