package volatility

import (
	"context"
	"math"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/utils"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) CalculateVolatility(context context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, dateTo *timestamppb.Timestamp, length int32) (*domain.VolatilityItemTechAnalyse, error) {
	limit := length * 3
	loc, _ := time.LoadLocation("Europe/Moscow")
	candles, _ := s.marketDataServiceClient.GetCandles(context, &instrumentUid, interval.ToNumberInvestApi(), utils.TimeStampPbGenerator(dateFrom.AsTime().In(loc), int64(-limit), interval), dateTo, &limit, true)
	prices := make([]float64, 0, len(candles))

	for i := 0; i < len(candles); i++ {
		prices = append(prices, utils.CombinePrice(candles[i].Close.Units, candles[i].Close.Nano))
	}
	if len(prices) < int(length+1) {
		return nil, errors.New("period can't be less than prices len")
	}
	/*var previousPrice, dispersionSum float64
	for _, p := range prices[len(prices)-int(length)-1:] {
		if previousPrice == 0 {
			previousPrice = p
			continue
		}

		diff := (p - previousPrice) / previousPrice * 100
		dispersionSum += math.Pow(diff, 2)

		previousPrice = p
	}

	v1, v2 := utils.SplitPrice(math.Round(math.Sqrt(dispersionSum/float64(length))*100) / 100)
	*/
	// Расчет среднего значения
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	average := sum / float64(len(prices))

	// Расчет стандартного отклонения
	variance := 0.0
	for _, price := range prices {
		variance += math.Pow(price-average, 2)
	}
	v1, v2 := utils.SplitPrice(math.Sqrt(variance / float64(len(prices)-1)))
	return &domain.VolatilityItemTechAnalyse{Value: domain.Quotation{
		Units: v1,
		Nano:  v2,
	}}, nil
}
