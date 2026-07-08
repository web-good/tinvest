package computable

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/bonds/factory"
	"tinvest/internal/utils"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) CalculateProfit(ctx context.Context, bond *pkgmodel.Bond) (domain.BondReport, error) {
	limit := int32(10)

	candles, errCandle := s.marketDataServiceGrpcClient.GetCandles(
		ctx,
		&bond.Id,
		int32(enum.Day1),
		utils.TimeStampPbGenerator(time.Now(), -20, enum.Day1),
		timestamppb.New(time.Now()),
		&limit,
		true,
	)

	if errCandle != nil {
		logger.ErrorContext(ctx, "error in GetCandles", slog.String("error_msg", errCandle.Error()))

		return domain.BondReport{}, errCandle
	}

	if len(candles) == 0 {
		return domain.BondReport{}, errors.New("few parameters returned candles")
	}

	yearStart := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	coupons, _ := s.instrumentServiceGrpcClient.GetBondCoupons(
		bond.Id,
		yearStart,
		bond.MaturityDate,
	)

	return calculateProfit(bond, coupons, candles[len(candles)-1]), nil
}

func calculateProfit(bond *pkgmodel.Bond, coupons []*pkgmodel.BondCoupon, candles *model.CandleItemTechAnalyse) domain.BondReport {
	var (
		totalFutureCoupons float64
		currentYearCoupons float64
	)

	now := time.Now()

	for _, coupon := range coupons {
		couponAmount := utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)

		if coupon.CouponDate.Year() == now.Year() {
			currentYearCoupons += couponAmount
		}

		if coupon.CouponDate.After(now) {
			totalFutureCoupons += couponAmount
		}
	}

	closePrice := utils.CombinePrice(candles.Close.Units, candles.Close.Nano)
	bondPrice := (closePrice * bond.Nominal) / 100
	totalInvestment := bondPrice + bond.Nkd

	var annualCouponIncome float64
	if currentYearCoupons > 0 {
		annualCouponIncome = currentYearCoupons
	} else {
		if len(coupons) > 0 {
			annualCouponIncome = utils.CombinePrice(coupons[0].PayOnBond.Units, coupons[0].PayOnBond.Nano) * float64(bond.CouponQuantityPerYear)
		}
	}
	couponPercentByYear := (annualCouponIncome * 100) / totalInvestment

	couponTax := (totalFutureCoupons - bond.Nkd) * 0.13
	if couponTax < 0 {
		couponTax = 0
	}

	var nominalPriceTax float64
	if diff := bond.Nominal - bondPrice; diff > 0 {
		nominalPriceTax = diff * 0.13
	}

	totalReturn := bond.Nominal + totalFutureCoupons
	finalProfit := totalReturn - totalInvestment - couponTax - nominalPriceTax

	daysToMaturity := int(bond.MaturityDate.Sub(now).Hours() / 24)
	if daysToMaturity < 1 {
		daysToMaturity = 1
	}

	profitPerYear := (finalProfit * 365) / float64(daysToMaturity)
	percentByYear := (100 * profitPerYear) / totalInvestment

	return factory.CreateBondReport(bond, finalProfit, profitPerYear, percentByYear, couponPercentByYear)
}
