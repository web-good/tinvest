package computable

import (
	"context"
	"errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/bonds/factory"
	"tinvest/internal/utils"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"
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

	coupons, _ := s.instrumentServiceGrpcClient.GetBondCoupons(
		bond.Id,
		time.Now(),
		bond.MaturityDate,
	)

	return calculateProfit(bond, coupons, candles[len(candles)-1]), nil
}

func calculateProfit(bond *pkgmodel.Bond, coupons []*pkgmodel.BondCoupon, candles *model.CandleItemTechAnalyse) domain.BondReport {
	var (
		totalCoupons       float64 = 0
		currentYearCoupons float64 = 0
	)

	// Расчет общей суммы всех купонов до погашения
	now := time.Now()

	for _, coupon := range coupons {
		couponAmount := utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)
		totalCoupons += couponAmount

		// Суммируем только купоны текущего года
		// Учитываем все купоны текущего года (выплаченные и будущие)
		couponYear := coupon.CouponDate.Year()
		if couponYear == now.Year() {
			currentYearCoupons += couponAmount
		}
	}

	// Расчет цены облигации
	closePrice := utils.CombinePrice(candles.Close.Units, candles.Close.Nano)
	bondPrice := (closePrice * bond.Nominal) / 100

	// Текущая купонная доходность (годовой показатель)
	var annualCouponIncome float64
	if currentYearCoupons > 0 {
		annualCouponIncome = currentYearCoupons
	} else {
		// Если купонов в текущем году нет, берем ближайший будущий купон
		if len(coupons) > 0 {
			annualCouponIncome = utils.CombinePrice(coupons[0].PayOnBond.Units, coupons[0].PayOnBond.Nano) * float64(bond.CouponQuantityPerYear)
		}
	}
	couponPercentByYear := (annualCouponIncome * 100) / bondPrice

	// Расчет налогов
	couponTax := totalCoupons * 0.13 // Налог на все купоны

	// Налог на разницу между номиналом и ценой покупки (только если есть прибыль)
	var nominalPriceTax float64 = 0
	nominalPriceDiff := bond.Nominal - bondPrice
	if nominalPriceDiff > 0 {
		nominalPriceTax = nominalPriceDiff * 0.13
	}

	// Итого инвестиции: цена облигации + НКД
	totalInvestment := bondPrice + bond.Nkd

	// Итого возврат: номинал + все купоны
	totalReturn := bond.Nominal + totalCoupons

	// Итоговая прибыль после вычета всех налогов
	finalProfit := totalReturn - totalInvestment - couponTax - nominalPriceTax

	// Точный расчет периода до погашения в днях
	daysToMaturity := int(bond.MaturityDate.Sub(now).Hours() / 24)
	if daysToMaturity < 1 {
		daysToMaturity = 1
	}

	// Годовая доходность в деньгах
	profitPerYear := (finalProfit * 365) / float64(daysToMaturity)

	// Годовая доходность в процентах от реальных инвестиций
	percentByYear := (100 * profitPerYear) / totalInvestment

	return factory.CreateBondReport(bond, finalProfit, profitPerYear, percentByYear, couponPercentByYear)
}
