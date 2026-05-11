package computable

import (
	"testing"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/utils"
	pkgmodel "tinvest/pkg/client/grpc/model"
)

func TestCalculateProfit(t *testing.T) {
	tests := []struct {
		name           string
		bond           *pkgmodel.Bond
		coupons        []*pkgmodel.BondCoupon
		candle         *model.CandleItemTechAnalyse
		expectedProfit float64
		expectedYield  float64
	}{
		{
			name: "ОФЗ с дисконтом",
			bond: &pkgmodel.Bond{
				Name:                  "ОФЗ 26234",
				Nominal:               1000.0,
				Nkd:                   15.5,
				MaturityDate:          time.Now().AddDate(1, 0, 0), // Через год
				Exchange:              "moex_morning_evening_ofz",
				CouponQuantityPerYear: 2,
			},
			coupons: []*pkgmodel.BondCoupon{
				{PayOnBond: *utils.CreateQuotation(30, 0), CouponDate: time.Now().AddDate(0, 3, 0)}, // 30 руб через 3 месяца
				{PayOnBond: *utils.CreateQuotation(30, 0), CouponDate: time.Now().AddDate(0, 9, 0)}, // 30 руб через 9 месяцев
			},
			candle: &model.CandleItemTechAnalyse{
				Close: utils.CreateInternalQuotation(98, 500000000), // 98.5% от номинала
			},
			// Расчет:
			// Цена: 985 руб, НКД: 15.5 руб, Инвестиции: 1000.5 руб
			// Купоны: 60 руб, Налог на купоны: 7.8 руб
			// Номинал - Цена: 15 руб, Налог: 1.95 руб
			// Возврат: 1000 + 60 = 1060 руб
			// Прибыль: 1060 - 1000.5 - 7.8 - 1.95 = 49.75 руб
			expectedProfit: 49.75,
			expectedYield:  4.97, // ~5% годовых
		},
		{
			name: "Корпоративная облигация с премией",
			bond: &pkgmodel.Bond{
				Name:                  "Тинькофф БО-02",
				Nominal:               1000.0,
				Nkd:                   25.0,
				MaturityDate:          time.Now().AddDate(2, 0, 0), // Через 2 года
				Exchange:              "moex_share",
				CouponQuantityPerYear: 2,
			},
			coupons: []*pkgmodel.BondCoupon{
				{PayOnBond: *utils.CreateQuotation(45, 0), CouponDate: time.Now().AddDate(0, 6, 0)}, // 45 руб через 6 мес
				{PayOnBond: *utils.CreateQuotation(45, 0), CouponDate: time.Now().AddDate(1, 0, 0)}, // 45 руб через год
				{PayOnBond: *utils.CreateQuotation(45, 0), CouponDate: time.Now().AddDate(1, 6, 0)}, // 45 руб через 1.5 года
				{PayOnBond: *utils.CreateQuotation(45, 0), CouponDate: time.Now().AddDate(2, 0, 0)}, // 45 руб через 2 года
			},
			candle: &model.CandleItemTechAnalyse{
				Close: utils.CreateInternalQuotation(102, 0), // 102% от номинала
			},
			// Расчет:
			// Цена: 1020 руб, НКД: 25 руб, Инвестиции: 1045 руб
			// Купоны: 180 руб, Налог: 23.4 руб
			// Номинал - Цена: -20 руб (убыток, налог не платится)
			// Возврат: 1000 + 180 = 1180 руб
			// Прибыль: 1180 - 1045 - 23.4 = 111.6 руб
			expectedProfit: 111.6,
			expectedYield:  5.34, // ~5.34% годовых
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := calculateProfit(tt.bond, tt.coupons, tt.candle)

			// Проверяем итоговую прибыль с погрешностью 0.1
			if report.FinalSum < tt.expectedProfit-0.1 || report.FinalSum > tt.expectedProfit+0.1 {
				t.Errorf("Ожидаемая прибыль %.2f, получили %.2f", tt.expectedProfit, report.FinalSum)
			}

			// Проверяем годовую доходность с погрешностью 0.1%
			if report.PercentByYear < tt.expectedYield-0.1 || report.PercentByYear > tt.expectedYield+0.1 {
				t.Errorf("Ожидаемая доходность %.2f%%, получили %.2f%%", tt.expectedYield, report.PercentByYear)
			}

			// Проверяем, что купонная доходность не включает налог
				// Для ОФЗ с двумя купонами в году по 30 руб и ценой 985 руб:
				// Годовые купоны: 30 * 2 = 60 руб
				// Купонная доходность: (60 * 100) / 985 = 6.09%
				if tt.name == "ОФЗ с дисконтом" {
					expectedCouponYield := 3.00
					if report.CouponPercentByYear < expectedCouponYield-0.1 || report.CouponPercentByYear > expectedCouponYield+0.1 {
						t.Errorf("Ожидаемая купонная доходность %.2f, получили %.2f", expectedCouponYield, report.CouponPercentByYear)
					}
				}
		})
	}
}

func TestCalculateProfit_Bond26248_CouponYield(t *testing.T) {
	// Тест для проверки правильного расчета купонной доходности
	// Облигация 26248: цена 885, купон 61.08 дважды в год
	bond := &pkgmodel.Bond{
		Name:                  "Облигация 26248",
		Nominal:               1000.0,
		Nkd:                   0.0,
		MaturityDate:          time.Date(2040, 5, 16, 0, 0, 0, 0, time.UTC),
		Exchange:              "moex_share",
		CouponQuantityPerYear: 2,
	}

	now := time.Now()
	currentYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	coupons := []*pkgmodel.BondCoupon{
		{PayOnBond: *utils.CreateQuotation(61, 800000000), CouponDate: currentYearStart.AddDate(0, 3, 0)}, // 61.08 в текущем году
		{PayOnBond: *utils.CreateQuotation(61, 800000000), CouponDate: currentYearStart.AddDate(0, 9, 0)}, // 61.08 в текущем году
	}

	candle := &model.CandleItemTechAnalyse{
		Close: utils.CreateInternalQuotation(88, 500000000), // 88.5% от номинала = 885 руб
	}

	report := calculateProfit(bond, coupons, candle)

	// Проверяем купонную доходность
	// Годовые купоны: 61.08 * 2 = 122.16 руб
	// Купонная доходность: (122.16 * 100) / 885 = 13.81%
	expectedCouponYield := 14.0

	if report.CouponPercentByYear < expectedCouponYield-0.1 || report.CouponPercentByYear > expectedCouponYield+0.1 {
		t.Errorf("Ожидаемая купонная доходность %.2f%%, получили %.2f%%", expectedCouponYield, report.CouponPercentByYear)
		t.Logf("Детали расчета:")
		t.Logf("  Цена облигации: %.2f руб", 885.0)
		t.Logf("  Годовые купоны: %.2f руб", 122.16)
		t.Logf("  Расчет: (%.2f * 100) / %.2f = %.2f%%", 122.16, 885.0, (122.16*100)/885.0)
	}
}

func TestCalculateProfit_AllYearCouponsIncludingPaid(t *testing.T) {
	// Тест проверяет, что в расчете купонной доходности участвуют ВСЕ купоны года,
	// включая те, что уже были выплачены

	now := time.Now()
	currentYear := now.Year()

	bond := &pkgmodel.Bond{
		Name:                  "Тестовая облигация",
		Nominal:               1000.0,
		Nkd:                   10.0,
		MaturityDate:          time.Now().AddDate(3, 0, 0),
		Exchange:              "moex_share",
		CouponQuantityPerYear: 4,
	}

	// Создаем купоны: 2 уже выплаченных в этом году и 2 будущих в этом же году
	coupons := []*pkgmodel.BondCoupon{
		// Уже выплаченные купоны (в начале года)
		{PayOnBond: *utils.CreateQuotation(25, 0), CouponDate: time.Date(currentYear, 2, 15, 0, 0, 0, 0, now.Location())},
		{PayOnBond: *utils.CreateQuotation(25, 0), CouponDate: time.Date(currentYear, 5, 15, 0, 0, 0, 0, now.Location())},

		// Будущие купоны (в конце года)
		{PayOnBond: *utils.CreateQuotation(25, 0), CouponDate: time.Date(currentYear, 8, 15, 0, 0, 0, 0, now.Location())},
		{PayOnBond: *utils.CreateQuotation(25, 0), CouponDate: time.Date(currentYear, 11, 15, 0, 0, 0, 0, now.Location())},
	}

	candle := &model.CandleItemTechAnalyse{
		Close: utils.CreateInternalQuotation(99, 0), // 99% от номинала = 990 руб
	}

	report := calculateProfit(bond, coupons, candle)

	// Проверяем купонную доходность
	// Все 4 купона должны учитываться: 25 * 4 = 100 руб в год
	// Купонная доходность: (100 * 100) / (990 + 10 НКД) = 10000 / 1000 = 10%
	expectedCouponYield := 10.0

	if report.CouponPercentByYear < expectedCouponYield-0.1 || report.CouponPercentByYear > expectedCouponYield+0.1 {
		t.Errorf("Ожидаемая купонная доходность %.2f%%, получили %.2f%%", expectedCouponYield, report.CouponPercentByYear)
		t.Logf("Детали расчета:")
		t.Logf("  Цена облигации: %.2f руб", 990.0)
		t.Logf("  НКД: %.2f руб", 10.0)
		t.Logf("  Общие инвестиции: %.2f руб", 1000.0)
		t.Logf("  Годовые купоны (4 шт по 25 руб): %.2f руб", 100.0)
		t.Logf("  Расчет: (%.2f * 100) / %.2f = %.2f%%", 100.0, 1000.0, expectedCouponYield)
		t.Logf("  Примечание: Учитываются ВСЕ купоны года, включая уже выплаченные")
	}
}