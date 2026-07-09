package converter

import (
	"time"
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
		if bond.FloatingCouponFlag == true {
			continue
		}

		res = append(res, ConvertBondModelFromBondPb(bond))
	}

	return res
}

func ConvertCouponsFromPb(in []*investapi.Coupon) []*model.BondCoupon {
	res := make([]*model.BondCoupon, 0, len(in))

	for _, coupon := range in {
		res = append(res, convertBondCouponModelFromCouponPb(coupon))
	}

	return res
}

func ConvertBondModelFromBondPb(bond *investapi.Bond) *model.Bond {
	return &model.Bond{
		ID:                    bond.Uid,
		Name:                  bond.Name,
		CouponQuantityPerYear: bond.CouponQuantityPerYear,
		AciValue:              utils.CombinePrice(bond.AciValue.Units, bond.AciValue.Nano),
		Exchange:              bond.Exchange,
		MaturityDate:          bond.MaturityDate.AsTime(),
		FloatingCouponFlag:    bond.FloatingCouponFlag,
		Nominal:               utils.CombinePrice(bond.Nominal.Units, bond.Nominal.Nano),
		RiskLevel:             bond.RiskLevel.String(),
		AmortizationFlag:      bond.AmortizationFlag,
		Nkd:                   utils.CombinePrice(bond.AciValue.Units, bond.AciValue.Nano),
	}
}

func convertBondCouponModelFromCouponPb(coupon *investapi.Coupon) *model.BondCoupon {
	loc, _ := time.LoadLocation("Europe/Moscow")

	return &model.BondCoupon{
		CouponDate:      coupon.CouponDate.AsTime().In(loc),
		PayOnBond:       model.Quotation{Nano: coupon.PayOneBond.Nano, Units: coupon.PayOneBond.Units},
		CouponStartDate: coupon.CouponStartDate.AsTime().In(loc),
		CouponEndDate:   coupon.CouponEndDate.AsTime().In(loc),
	}
}
