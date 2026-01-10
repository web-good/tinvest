package bonds

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
	"sync"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/bonds/notification"
	"tinvest/internal/utils"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/collection"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context) error {
	var wg sync.WaitGroup
	bonds, err := s.instrumentServiceGrpcClient.Bonds(ctx)
	var (
		selectionOfzBond  []domain.BondReport
		selectionCorpBond []domain.BondReport
	)
	if err != nil {
		return err
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		selectionOfzBond = s.findBonds(ctx, true, bonds)
		err := s.tgClient.SendMessage(notification.Send(selectionOfzBond))
		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", slog.String("error_msg", err.Error()))
		}
	}()
	go func() {
		defer wg.Done()
		selectionCorpBond = s.findBonds(ctx, false, bonds)
		err := s.tgClient.SendMessage(notification.Send(selectionCorpBond))
		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", slog.String("error_msg", err.Error()))
		}
	}()
	wg.Wait()

	return nil
}

func (s *service) findBonds(ctx context.Context, isOfz bool, bonds []*pkgmodel.Bond) []domain.BondReport {
	selectionOfzBond := make([]domain.BondReport, 15)
	collectionBond := collection.New[domain.BondReport]()

	for _, bond := range bonds {
		time.Sleep(100 * time.Millisecond)
		if bond.FloatingCouponFlag == true {
			continue
		}
		if isOfz == true && bond.Exchange != "moex_morning_evening_ofz" {
			continue
		}

		if isOfz == false && bond.Exchange == "moex_morning_evening_ofz" {
			continue
		}

		days := dayDiff(bond.MaturityDate, time.Now())
		if days < 180 {
			continue
		}

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

			continue
		}

		if len(candles) == 0 {
			continue
		}

		coupons, _ := s.instrumentServiceGrpcClient.GetBondCoupons(
			bond.Id,
			time.Now(),
			bond.MaturityDate,
		)
		collectionBond.Add(calculateProfit(bond, coupons, candles[len(candles)-1]))
	}

	selectionOfzBond = collectionBond.GetTopByCriteria(func(i, j domain.BondReport) bool {
		return i.PercentByYear > j.PercentByYear
	})

	return selectionOfzBond
}

func dayDiff(dateStart time.Time, dateNow time.Time) int {
	currentDate := time.Date(dateNow.Year(), dateNow.Month(), dateNow.Day(), 0, 0, 0, 0, dateNow.Location())
	diff := dateStart.Sub(currentDate)

	return int(diff.Hours() / 24)
}

func calculateProfit(bond *pkgmodel.Bond, coupons []*pkgmodel.BondCoupon, candles *model.CandleItemTechAnalyse) domain.BondReport {
	var (
		couponProfit  float64 = 0
		profitResult2 float64
	)

	for _, coupon := range coupons {
		couponProfit += utils.CombinePrice(coupon.PayOnBond.Units, coupon.PayOnBond.Nano)
	}

	bondPrice := (utils.CombinePrice(candles.Close.Units, candles.Close.Nano) * bond.Nominal) / 100
	couponProfitNalog := couponProfit - (couponProfit * 13 / 100)
	if bondPrice < bond.Nominal {
		profitResult2 = ((bond.Nominal - bondPrice - ((bond.Nominal - bondPrice) * 13 / 100)) + couponProfitNalog) - bond.Nkd
	} else {
		profitResult2 = ((bond.Nominal - bondPrice) + couponProfitNalog) - bond.Nkd
	}

	monthCount, _ := monthsBetweenDates(bond.MaturityDate.Format("2006-01-02"))
	price12 := (profitResult2 * 12) / float64(monthCount)
	ofzType := domain.CorpBondEnum
	percentByYear := (100 * price12) / bondPrice

	if bond.Exchange == "moex_morning_evening_ofz" {
		ofzType = domain.OfzBondEnum
	}
	//if bond.Name == "Яндекс Финтех выпуск 1" {
	//	fmt.Println(4234)
	//}
	return domain.BondReport{
		Name:          bond.Name,
		FinalSum:      utils.RoundFloat(profitResult2, 1),
		ManyByYear:    utils.RoundFloat(price12, 1),
		PercentByYear: utils.RoundFloat(percentByYear, 1),
		Nkd:           bond.Nkd,
		ExecutionDate: bond.MaturityDate,
		Type:          ofzType,
	}
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
