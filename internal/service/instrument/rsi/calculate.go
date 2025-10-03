package rsi

import (
	"context"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/utils"
)

func (s service) CalculateRSI(context context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, dateTo *timestamppb.Timestamp, period int32) ([]*domain.RSIItemTechAnalyse, error) {
	//fmt.Println(dateFrom.AsTime(), dateTo.AsTime())
	limit := period*3 - 1
	candles, _ := s.marketDataServiceClient.GetCandles(context, &instrumentUid, interval.ToNumberInvestApi(), dateFrom, dateTo, &limit, true)

	if len(candles) < int(period+2)-1 {
		return nil, errors.New("prices len must be more or equal 2 periods +1 mor date from")
	}

	prices := make([]float64, 0, len(candles))
	date := make([]time.Time, 0, len(candles))

	for i := 0; i < len(candles); i++ {
		date = append(date, candles[i].Time)
		prices = append(prices, utils.CombinePrice(candles[i].Close.Units, candles[i].Close.Nano))
	}

	date = date[len(date)-int(period+1):]
	var gains, loses []float64
	var previousPrice float64
	p := float64(period)
	for _, currentPrice := range prices {
		if previousPrice == 0 {
			previousPrice = currentPrice
			continue
		}

		changePrice := currentPrice - previousPrice
		if changePrice > 0 {
			gains = append(gains, changePrice)
			loses = append(loses, 0)
		} else {
			loses = append(loses, math.Abs(changePrice))
			gains = append(gains, 0)
		}

		previousPrice = currentPrice
	}

	var avgGains, avgLoses []float64
	var previousAvgGain, previousAvgLose float64
	for _, g := range gains[:period] {
		previousAvgGain = previousAvgGain + g
	}

	previousAvgGain = previousAvgGain / p
	avgGains = append(avgGains, previousAvgGain)

	for _, l := range loses[:period] {
		previousAvgLose = previousAvgLose + l
	}

	previousAvgLose = previousAvgLose / p
	avgLoses = append(avgLoses, previousAvgLose)

	for _, g := range gains[period:] {
		avgGain := (previousAvgGain*(p-1) + g) / p
		avgGains = append(avgGains, avgGain)
		previousAvgGain = avgGain
	}

	for _, l := range loses[period:] {
		avgLose := (previousAvgLose*(p-1) + l) / p
		avgLoses = append(avgLoses, avgLose)
		previousAvgLose = avgLose
	}

	var rsiArr []float64
	for i, gain := range avgGains {
		var rs float64 = 1
		if avgLoses[i] != 0 {
			rs = gain / avgLoses[i]
		}

		rsi := 100 - 100/(1+rs)
		rsiArr = append(rsiArr, math.Round(rsi*100)/100)
	}

	rsi := make([]*domain.RSIItemTechAnalyse, 0, period)
	pe := period

	for i := len(rsiArr) - 1; i >= 1; i-- {
		r1, r2 := utils.SplitPrice(rsiArr[i])
		rsi = append(rsi, &domain.RSIItemTechAnalyse{
			Date:       date[pe],
			SignalLine: domain.Quotation{Units: r1, Nano: r2},
		})

		if pe != 0 {
			pe = pe - 1
		}
	}

	return rsi, nil
}
