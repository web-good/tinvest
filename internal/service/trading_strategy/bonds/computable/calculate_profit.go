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
	"tinvest/pkg/indicators"
	"tinvest/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// maxCandleAgeDays — страховка от неликвида: цена не должна быть слишком старой.
const maxCandleAgeDays = 7

func (s *service) CalculateProfit(ctx context.Context, bond *pkgmodel.Bond) (domain.BondReport, error) {
	limit := int32(10)

	candles, errCandle := s.marketDataServiceGrpcClient.GetCandles(
		ctx,
		&bond.ID,
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
		bond.ID,
		yearStart,
		bond.MaturityDate,
	)

	last := candles[len(candles)-1]
	if time.Since(last.Time) > maxCandleAgeDays*24*time.Hour {
		return domain.BondReport{}, errors.New("stale candle: price too old")
	}

	return calculateProfit(bond, coupons, last)
}

func calculateProfit(bond *pkgmodel.Bond, coupons []*pkgmodel.BondCoupon, candles *model.CandleItemTechAnalyse) (domain.BondReport, error) {
	const taxRate = 0.13

	now := time.Now()

	var currentYearCoupons float64
	for _, coupon := range coupons {
		if coupon.CouponDate.Year() == now.Year() {
			currentYearCoupons += utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)
		}
	}

	closePrice := utils.CombinePrice(candles.Close.Units, candles.Close.Nano)
	bondPrice := (closePrice * bond.Nominal) / 100
	totalInvestment := bondPrice + bond.Nkd

	// Текущая купонная доходность (второй показатель, оставляем как было).
	var annualCouponIncome float64
	if currentYearCoupons > 0 {
		annualCouponIncome = currentYearCoupons
	} else if len(coupons) > 0 {
		annualCouponIncome = utils.CombinePrice(coupons[0].PayOnBond.Units, coupons[0].PayOnBond.Nano) * float64(bond.CouponQuantityPerYear)
	}
	couponPercentByYear := 0.0
	if totalInvestment > 0 {
		couponPercentByYear = (annualCouponIncome * 100) / totalInvestment
	}

	// Денежные потоки для XIRR.
	flows := []indicators.CashFlow{{Date: now, Amount: -totalInvestment}}

	var firstCouponIdx = -1
	for _, coupon := range coupons {
		if !coupon.CouponDate.After(now) {
			continue
		}
		amount := utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano) * (1 - taxRate)
		flows = append(flows, indicators.CashFlow{Date: coupon.CouponDate, Amount: amount})
		if firstCouponIdx == -1 {
			firstCouponIdx = len(flows) - 1
		}
	}
	// Налоговый щит НКД — к самому раннему будущему купону.
	if firstCouponIdx != -1 {
		flows[firstCouponIdx].Amount += bond.Nkd * taxRate
	}

	// Возврат номинала за вычетом налога на прирост (цена ниже номинала).
	var priceTax float64
	if diff := bond.Nominal - bondPrice; diff > 0 {
		priceTax = diff * taxRate
	}
	flows = append(flows, indicators.CashFlow{Date: bond.MaturityDate, Amount: bond.Nominal - priceTax})

	ytm, err := indicators.XIRR(flows)
	if err != nil {
		return domain.BondReport{}, err
	}

	// Совокупная прибыль и линейная годовая прибыль в деньгах (второстепенные показатели).
	var totalCouponsNet float64
	for _, f := range flows[1:] {
		totalCouponsNet += f.Amount
	}
	finalProfit := totalCouponsNet - totalInvestment
	daysToMaturity := int(bond.MaturityDate.Sub(now).Hours() / 24)
	if daysToMaturity < 1 {
		daysToMaturity = 1
	}
	profitPerYear := (finalProfit * 365) / float64(daysToMaturity)

	return factory.CreateBondReport(bond, finalProfit, profitPerYear, ytm*100, couponPercentByYear), nil
}
