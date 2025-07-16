package atr

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"time"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
)

func (s *service) TechAnalyse(context context.Context, instrumentUid *string) (atr.ItemTechAnalyse, error) {
	p := int32(25)
	candles, err := s.marketDataServiceClient.GetCandles(context, instrumentUid, 5, timestamppb.New(time.Now().AddDate(0, 0, -50)), timestamppb.New(time.Now()), &p, false)

	if err != nil {
		return atr.ItemTechAnalyse{}, fmt.Errorf("failed to get candles from MarketDataService: %w", err)
	}

	if len(candles) < int(s.period) {
		return atr.ItemTechAnalyse{}, fmt.Errorf("candles from MarketDataService is too short")
	}

	filteredCandles := s.filterCandles(candles, 2.0)
	atrV := atr.ItemTechAnalyse{
		Value: s.calculateATR(filteredCandles),
	}

	if atrV.Value == 0 {
		return atr.ItemTechAnalyse{}, fmt.Errorf("cannot calculate ATR")
	}

	atrV.Passed = s.calculatePassedValue(candles[len(candles)-1], atrV.Value)

	return atrV, nil
}

// Функция для вычисления среднего значения
func (s *service) mean(data []float64) float64 {
	sum := 0.0

	for _, v := range data {
		sum += v
	}

	return sum / float64(len(data))
}

// Функция для вычисления стандартного отклонения
func (s *service) stdDev(data []float64) float64 {
	m := s.mean(data)
	varianceSum := 0.0

	for _, v := range data {
		varianceSum += (v - m) * (v - m)
	}

	variance := varianceSum / float64(len(data))

	return math.Sqrt(variance)
}

// Функция для фильтрации свечей по стандартному отклонению
func (s *service) filterCandles(candles []*model.CandleItemTechAnalyse, multiplier float64) []*model.CandleItemTechAnalyse {
	// Собираем диапазоны
	ranges := make([]float64, len(candles))

	for i := len(candles) - 1; i >= 0; i-- {
		ranges[i] = float64(candles[i].High.Units - candles[i].Low.Units)
	}

	// Вычисляем среднее и стандартное отклонение
	meanRange := s.mean(ranges)
	sd := s.stdDev(ranges)

	// Определяем порог
	lowerThreshold := meanRange - multiplier*sd
	upperThreshold := meanRange + multiplier*sd

	if lowerThreshold < 0 {
		lowerThreshold = 0
	}

	// Отбираем свечи, попадающие в диапазон
	var filtered []*model.CandleItemTechAnalyse
	for _, c := range candles {
		r := float64(c.High.Units - c.Low.Units)

		if r >= lowerThreshold && r <= upperThreshold {
			filtered = append(filtered, c)
		}

		if len(filtered) == int(s.period) {
			break
		}
	}

	return filtered
}

// Функция для вычисления ATR по массиву свечей
func (s *service) calculateATR(candles []*model.CandleItemTechAnalyse) float64 {
	if len(candles) < int(s.period) {
		return 0.0
	}

	sumTR := 0.0

	for i := 1; i < int(s.period); i++ {
		current := candles[i]
		prev := candles[i-1]
		high := current.High.Units
		low := current.Low.Units
		prevClose := prev.Close.Units

		highLow := float64(high - low)
		highPrevClose := math.Abs(float64(high - prevClose))
		lowPrevClose := math.Abs(float64(low - prevClose))

		tr := math.Max(highLow, math.Max(highPrevClose, lowPrevClose))
		sumTR += tr
	}

	return sumTR / float64(s.period)
}

func (s *service) calculatePassedValue(candle *model.CandleItemTechAnalyse, atr float64) int64 {
	var (
		high model.Quotation
		low  model.Quotation
	)
	if candle.Open.Units > candle.Close.Units {
		high = candle.Open
		low = candle.Close
	}

	if candle.Open.Units < candle.Close.Units {
		low = candle.Open
		high = candle.Close
	}

	if candle.Open.Units == candle.Close.Units {
		if candle.Open.Nano > candle.Close.Nano {
			high = candle.Open
			low = candle.Close
		} else {
			high = candle.Close
			low = candle.Open
		}
	}

	return int64(float64(combinePrice(high.Units, high.Nano)-combinePrice(low.Units, low.Nano)) * 100 / atr)
}

func combinePrice(intPart int64, frac int32) float64 {
	return float64(intPart) + float64(frac)/1e9
}
