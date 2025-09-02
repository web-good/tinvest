package macd

import (
	"context"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/utils"
)

func (s service) CalculateMACD(context context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, dateTo *timestamppb.Timestamp, fast int32, slow int32, signal int32) ([]*domain.MACDItemTechAnalyse, error) {
	limit := slow * 2
	loc, _ := time.LoadLocation("Europe/Moscow")

	candles, _ := s.marketDataServiceClient.GetCandles(context, &instrumentUid, interval.ToNumberInvestApi(), utils.TimeStampPbGenerator(dateFrom.AsTime().In(loc), int64(-limit*2), interval), dateTo, &limit, true)

	if len(candles) < int(limit) {
		return nil, errors.New("prices len must be more or equal 2 periods")
	}

	candlesFloat := make([]float64, 0, len(candles))

	for i := 0; i < len(candles); i++ {
		candlesFloat = append(candlesFloat, utils.CombinePrice(candles[i].Close.Units, candles[i].Close.Nano))
	}

	shortEMA := calculateEMA(candlesFloat, int(fast))
	longEMA := calculateEMA(candlesFloat, int(slow))
	macdLine := make([]float64, 0, len(candlesFloat))
	for i := 0; i < len(longEMA); i++ {
		macdLine = append(macdLine, shortEMA[i]-longEMA[i])
	}

	signalLine := calculateEMA(macdLine, int(signal))
	macd := make([]*domain.MACDItemTechAnalyse, len(candlesFloat))
	start := 0
	end := 0
	for i := int(slow) - 1; i < len(candles); i++ {
		if candles[i].Time.In(loc).Before(dateTo.AsTime().In(loc)) {
			end = i
		}
		if candles[i].Time.In(loc).Before(dateFrom.AsTime().In(loc)) {
			start = i
		}
		k1, k2 := utils.SplitPrice(signalLine[i])
		g1, g2 := utils.SplitPrice(macdLine[i])
		diff := signalLine[i] - macdLine[i]
		if diff > 0 {
			diff = 1
		} else {
			diff = -1
		}

		underZero := true

		if signalLine[i] < 0 || macdLine[i] < 0 {
			underZero = false
		}

		isCross := false

		if i != int(slow)-1 {
			if macd[i-1].Diff == -1 && diff == 1 {
				isCross = true
			}

			if macd[i-1].Diff == 1 && diff == -1 {
				isCross = true
			}
		}

		macd[i] = &domain.MACDItemTechAnalyse{
			Date:    candles[i].Time.In(loc),
			Diff:    diff,
			IsCross: isCross,
			SignalLine: domain.Quotation{
				Units: k1,
				Nano:  k2,
			},
			UnderZero: underZero,
			MacDLine: domain.Quotation{
				Units: g1,
				Nano:  g2,
			},
		}
	}

	r := macd[start+1 : end+1]

	return r, nil
}

func calculateEMA(data []float64, period int) []float64 {
	emaValues := make([]float64, len(data))
	alpha := 2.0 / float64(period+1)
	emaValues[0] = data[0]
	for i := 1; i < len(data); i++ {
		emaValues[i] = (data[i]-emaValues[i-1])*alpha + emaValues[i-1]
	}
	return emaValues
}
