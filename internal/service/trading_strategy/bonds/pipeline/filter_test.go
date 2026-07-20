package pipeline

import (
	"testing"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func goodBond() *pkgmodel.Bond {
	return &pkgmodel.Bond{
		RiskLevel:           "RISK_LEVEL_LOW",
		LiquidityFlag:       true,
		SubordinatedFlag:    false,
		ForQualInvestorFlag: false,
		PerpetualFlag:       false,
	}
}

func TestPassesReliability(t *testing.T) {
	tests := []struct {
		name string
		mut  func(b *pkgmodel.Bond)
		want bool
	}{
		{"надёжная проходит", func(b *pkgmodel.Bond) {}, true},
		{"не LOW risk — отсев", func(b *pkgmodel.Bond) { b.RiskLevel = "RISK_LEVEL_MODERATE" }, false},
		{"неликвид — отсев", func(b *pkgmodel.Bond) { b.LiquidityFlag = false }, false},
		{"суборд — отсев", func(b *pkgmodel.Bond) { b.SubordinatedFlag = true }, false},
		{"только для квалов — отсев", func(b *pkgmodel.Bond) { b.ForQualInvestorFlag = true }, false},
		{"бессрочная — отсев", func(b *pkgmodel.Bond) { b.PerpetualFlag = true }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := goodBond()
			tc.mut(b)
			if got := PassesReliability(b); got != tc.want {
				t.Fatalf("PassesReliability = %v, want %v", got, tc.want)
			}
		})
	}
}
