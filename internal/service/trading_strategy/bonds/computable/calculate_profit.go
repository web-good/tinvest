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
		couponProfit  float64 = 0
		finalSum      float64
		couponYearSum float64 = 0
	)

	j := 0
	for _, coupon := range coupons {
		couponProfit += utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)
		if j < int(bond.CouponQuantityPerYear) {
			couponYearSum = couponYearSum + utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)
			j++
		}
	}

	couponYearSum = couponYearSum - couponYearSum*13/100
	bondPrice := (utils.CombinePrice(candles.Close.Units, candles.Close.Nano) * bond.Nominal) / 100
	procentCouponYear := (couponYearSum * 100) / bondPrice
	couponProfitNalog := couponProfit - (couponProfit * 13 / 100)
	if bondPrice < bond.Nominal {
		finalSum = ((bond.Nominal - bondPrice - ((bond.Nominal - bondPrice) * 13 / 100)) + couponProfitNalog) - bond.Nkd
	} else {
		finalSum = ((bond.Nominal - bondPrice) + couponProfitNalog) - bond.Nkd
	}

	monthCount, _ := monthsBetweenDates(bond.MaturityDate.Format("2006-01-02"))
	manyByYear := (finalSum * 12) / float64(monthCount)
	percentByYear := (100 * manyByYear) / bondPrice

	return factory.CreateBondReport(bond, finalSum, manyByYear, percentByYear, procentCouponYear)
}

// monthsBetweenDates возвращает количество месяцев между текущей датой и целевой датой
func monthsBetweenDates(targetDateStr string) (int, error) {
	// Парсим целевую дату из строки
	targetDate, err := time.Parse("2006-01-02", targetDateStr)
	if err != nil {
		return 0, err
	}

	// Получаем текущую дату
	currentDate := time.Now()
	// Вычисляем разницу в годах и месяцах
	years := targetDate.Year() - currentDate.Year()
	months := int(targetDate.Month()) - int(currentDate.Month())
	totalMonths := years*12 + months

	return totalMonths, nil
}
