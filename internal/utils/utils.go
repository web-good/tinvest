package utils

import "math"

func CombinePrice(intPart int64, frac int32) float64 {
	return float64(intPart) + float64(frac)/1e9
}

func SplitPrice(price float64) (int64, int32) {
	frac, intPart := math.Modf(price)

	return int64(frac), int32(math.Round(intPart * 1e9))
}
