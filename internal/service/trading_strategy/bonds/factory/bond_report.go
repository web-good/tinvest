package factory

import (
	"tinvest/internal/domain"
	"tinvest/internal/utils"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func CreateBondReport(bond *pkgmodel.Bond, finalSum, manyByYear, percentByYear, procentCouponYear float64) domain.BondReport {
	bondType := domain.CorpBondEnum
	if bond.Exchange == "moex_morning_evening_ofz" {
		bondType = domain.OfzBondEnum
	}

	return domain.BondReport{
		Name:                bond.Name,
		FinalSum:            utils.RoundFloat(finalSum, 1),
		ManyByYear:          utils.RoundFloat(manyByYear, 1),
		PercentByYear:       utils.RoundFloat(percentByYear, 1),
		CouponPercentByYear: utils.RoundFloat(procentCouponYear, 1),
		Nkd:                 bond.Nkd,
		ExecutionDate:       bond.MaturityDate,
		Type:                bondType,
	}
}
