package utils

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"time"
	"tinvest/internal/enum"
)

func CombinePrice(intPart int64, frac int32) float64 {
	return float64(intPart) + float64(frac)/1e9
}

func SplitPrice(price float64) (int64, int32) {
	frac, intPart := math.Modf(price)

	return int64(frac), int32(math.Round(intPart * 1e9))
}

func TimeStampPbGenerator(date time.Time, countCandles int64, timeFrame enum.Interval) *timestamppb.Timestamp {
	loc, _ := time.LoadLocation("Europe/Moscow")

	return timestamppb.New(date.Add(time.Duration(countCandles * int64(timeFrame.ToTimeDuration()))).In(loc))
}

func TimeGenerator(date time.Time, count int64, timeFrame enum.Interval) time.Time {
	loc, _ := time.LoadLocation("Europe/Moscow")
	return date.Add(time.Duration(count * int64(timeFrame.ToTimeDuration()))).In(loc)
}

func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
