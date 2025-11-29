package converter

import (
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
	"tinvest/pkg/client/grpc/model"
)

func ConvertBondsFromPb(in *investapi.BondsResponse) []*model.Bond {
	res := make([]*model.Bond, 0, len(in.Instruments))

	for _, bond := range in.Instruments {
		if bond.RiskLevel != investapi.RiskLevel_RISK_LEVEL_LOW {
			continue
		}

		if bond.Exchange != "moex_morning_evening_ofz" {
			continue
		}

		res = append(res, convertBondModelFromBondPb(bond))
	}

	return res
}

func convertBondModelFromBondPb(bond *investapi.Bond) *model.Bond {
	return &model.Bond{
		Id:                    bond.Uid,
		Name:                  bond.Name,
		CouponQuantityPerYear: bond.CouponQuantityPerYear,
		AciValue:              utils.CombinePrice(bond.AciValue.Units, bond.AciValue.Nano),
		Exchange:              bond.Exchange,
		MaturityDate:          bond.MaturityDate.AsTime(),
		FloatingCouponFlag:    bond.FloatingCouponFlag,
		Nominal:               utils.CombinePrice(bond.Nominal.Units, bond.Nominal.Nano),
		RiskLevel:             bond.RiskLevel.String(),
	}
}
