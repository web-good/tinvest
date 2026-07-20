package factory

import (
	"regexp"
	"tinvest/internal/domain"
	"tinvest/internal/utils"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func CreateBondReport(bond *pkgmodel.Bond, finalProfit, profitPerYear, percentByYear, couponPercentByYear float64) domain.BondReport {
	bondType := domain.CorpBondEnum
	reOfz := regexp.MustCompile(`ОФЗ`)
	reRegion := regexp.MustCompile(`Реги`)

	if reOfz.MatchString(bond.Name) || reRegion.MatchString(bond.Name) {
		bondType = domain.OfzBondEnum
	}

	return domain.BondReport{
		Name:                bond.Name,
		FinalSum:            utils.RoundFloat(finalProfit, 1),         // Итоговая прибыль после всех налогов
		ManyByYear:          utils.RoundFloat(profitPerYear, 1),       // Годовая прибыль в деньгах
		PercentByYear:       utils.RoundFloat(percentByYear, 1),       // Годовая доходность в %
		CouponPercentByYear: utils.RoundFloat(couponPercentByYear, 1), // Текущая купонная доходность
		Nkd:                 utils.RoundFloat(bond.Nkd, 2),            // НКД с точностью до копеек
		ExecutionDate:       bond.MaturityDate,
		Type:                bondType,
		Sector:              bond.Sector,
		RiskLevel:           bond.RiskLevel,
		Liquidity:           bond.LiquidityFlag,
	}
}
