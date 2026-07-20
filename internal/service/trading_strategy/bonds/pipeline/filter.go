package pipeline

import pkgmodel "tinvest/pkg/client/grpc/model"

// riskLevelLow — строковое представление RiskLevel_RISK_LEVEL_LOW из proto
// (модель хранит RiskLevel как .String()).
const riskLevelLow = "RISK_LEVEL_LOW"

// PassesReliability — явная политика отбора надёжных облигаций для стратегии
// «надёжные облигации». Раньше фильтр по риску был захардкожен и спрятан в
// конвертере; теперь он видимый, тестируемый и настраиваемый здесь.
func PassesReliability(b *pkgmodel.Bond) bool {
	if b.RiskLevel != riskLevelLow {
		return false
	}
	if !b.LiquidityFlag {
		return false
	}
	if b.SubordinatedFlag || b.ForQualInvestorFlag || b.PerpetualFlag {
		return false
	}
	return true
}
