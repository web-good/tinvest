package macd

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
	"strings"
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

	ema26, _ := calculateEMA(candlesFloat, slow)
	ema12, _ := calculateEMA(candlesFloat, fast)
	roundPrecision := detectPrecision(ema12[0])

	//roundPrecision := detectPrecision(utils.CombinePrice(ema12[0].SignalLine.Units, ema12[0].SignalLine.Nano))
	ema12 = ema12[len(ema12)-len(ema26):]

	macdq := make([]float64, 0)
	for i := 0; i < len(ema26); i++ {
		macdq = append(macdq, utils.RoundFloat(ema12[i]-ema26[i], roundPrecision))
	}
	/*macdL := make([]*model.CandleItemTechAnalyse, 0, len(ema26))

	for i := 0; i < len(ema26); i++ {
		v1, v2 := utils.SplitPrice(
			utils.RoundFloat(utils.CombinePrice(ema12[i].SignalLine.Units, ema12[i].SignalLine.Nano)-utils.CombinePrice(ema26[i].SignalLine.Units, ema26[i].SignalLine.Nano), roundPrecision),
		)

		macdL = append(macdL, &model.CandleItemTechAnalyse{
			Time:  ema26[i].Date,
			Close: model.Quotation{Units: v1, Nano: v2},
		})
	}

	signalL, err := calculateEMA(macdL, signal)

	if err != nil {
		return nil, fmt.Errorf("can't calculate signal ema: %w", err)
	}*/
	signalL, _ := calculateEMA(macdq, signal)
	macd := make([]*domain.MACDItemTechAnalyse, len(macdq))
	date := dateFrom.AsTime()
	for i := 0; i < len(macdq); i++ {
		if len(signalL)-1 < i {
			continue
		}

		date = date.Add(interval.ToTimeDuration())
		k1, k2 := utils.SplitPrice(signalL[i])
		g1, g2 := utils.SplitPrice(macdq[i])
		macd[i] = &domain.MACDItemTechAnalyse{
			Date: date.In(loc),
			SignalLine: domain.Quotation{
				Units: k1,
				Nano:  k2,
			},
			MacDLine: domain.Quotation{
				Units: g1,
				Nano:  g2,
			},
		}
	}

	return macd, nil
}

func calculateEMA(prices []float64, period int32) ([]float64, error) {
	if len(prices) < 2*int(period) {
		return nil, errors.New("prices len must be more or equal 2 periods")
	}

	var emas []float64

	roundPrecision := detectPrecision(prices[0])
	// first ema value = sma value
	sma, err := calculateSMA(prices[:period], int(period))
	if err != nil {
		return nil, fmt.Errorf("can't calculate sma: %w", err)
	}

	previousEma := sma[0]

	emas = append(emas, utils.RoundFloat(previousEma, roundPrecision))

	k := 2 / float64(1+period)
	for _, p := range prices[period:] {
		previousEma = emas[len(emas)-1]
		ema := (p * k) + (previousEma * (1 - k))
		emas = append(emas, utils.RoundFloat(ema, roundPrecision))
	}

	return emas, nil
}

func detectPrecision(val float64) uint {
	// WARNING: default precision needs for int prices, because division takes place in formulas
	var defaultPrecision uint = 2

	strFloat := strconv.FormatFloat(val, 'f', -1, 64)
	posDecimal := strings.Index(strFloat, ".")

	if posDecimal == -1 {
		return defaultPrecision
	}

	decimalSignsCnt := len(strFloat[posDecimal+1:])
	if decimalSignsCnt > int(defaultPrecision) {
		// WARNING: added 1 to better precision for small values like 0.0000078, because calculate functions could divide small price
		return uint(decimalSignsCnt) + 1
	}

	return defaultPrecision
}

func calculateSMA(prices []float64, period int) ([]float64, error) {
	if len(prices) < period {
		return nil, errors.New("prices len must be equal to period")
	}

	roundPrecision := detectPrecision(prices[0])

	var sma []float64
	for i := 0; i+period <= len(prices); i++ {
		var sum float64
		for _, p := range prices[i : i+period] {
			sum += p
		}

		sma = append(sma, utils.RoundFloat(sum/float64(period), roundPrecision))
	}

	return sma, nil
}

/*
func calculateEMA(candles []*model.CandleItemTechAnalyse, period int32) ([]*ema.ItemTechAnalyse, error) {
	if len(candles) < 2*int(period) {
		return nil, errors.New("prices len must be more or equal 2 periods")
	}

	var emas []*ema.ItemTechAnalyse

	prices := make([]float64, 0, len(candles))

	for i := 0; i < len(candles); i++ {
		prices = append(prices, utils.CombinePrice(candles[i].Close.Units, candles[i].Close.Nano))
	}

	roundPrecision := detectPrecision(prices[0])
	// first ema value = sma value
	sma, err := calculateSMA(candles[:period], period)
	if err != nil {
		return nil, fmt.Errorf("can't calculate sma: %w", err)
	}

	previousEma := sma[0]
	emas = append(emas, &ema.ItemTechAnalyse{
		Date:       previousEma.Date,
		SignalLine: previousEma.SignalLine,
	})

	k := 2 / float64(1+period)
	for _, p := range candles[period:] {
		previousEma = emas[len(emas)-1]
		v1, v2 := utils.SplitPrice(utils.RoundFloat(utils.CombinePrice(p.Close.Units, p.Close.Nano)*k, roundPrecision) + (utils.CombinePrice(previousEma.SignalLine.Units, previousEma.SignalLine.Nano) * (1 - k)))
		emas = append(emas, &ema.ItemTechAnalyse{
			Date: p.Time,
			SignalLine: ema.Quotation{
				Units: v1,
				Nano:  v2,
			},
		})
	}

	return emas, nil
}

func detectPrecision(val float64) uint {
	// WARNING: default precision needs for int prices, because division takes place in formulas
	var defaultPrecision uint = 2

	strFloat := strconv.FormatFloat(val, 'f', -1, 64)
	posDecimal := strings.Index(strFloat, ".")

	if posDecimal == -1 {
		return defaultPrecision
	}

	decimalSignsCnt := len(strFloat[posDecimal+1:])
	if decimalSignsCnt > int(defaultPrecision) {
		// WARNING: added 1 to better precision for small values like 0.0000078, because calculate functions could divide small price
		return uint(decimalSignsCnt) + 1
	}

	return defaultPrecision
}

func calculateSMA(prices []*model.CandleItemTechAnalyse, period int32) ([]*ema.ItemTechAnalyse, error) {
	if len(prices) < int(period) {
		return nil, errors.New("prices len must be equal to period")
	}

	roundPrecision := detectPrecision(utils.CombinePrice(prices[0].Close.Units, prices[0].Close.Nano))

	var sma []*ema.ItemTechAnalyse
	for i := 0; i+int(period) <= len(prices); i++ {
		var sum float64
		for _, p := range prices[i : i+int(period)] {
			sum += utils.CombinePrice(p.Close.Units, p.Close.Nano)
		}

		v1, v2 := utils.SplitPrice(utils.RoundFloat(sum/float64(period), roundPrecision))
		sma = append(sma, &ema.ItemTechAnalyse{
			Date: prices[i].Time,
			SignalLine: ema.Quotation{
				Units: v1,
				Nano:  v2,
			},
		})
	}

	return sma, nil
}
*/
