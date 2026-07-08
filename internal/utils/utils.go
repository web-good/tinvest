package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	pkgmodel "tinvest/pkg/client/grpc/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func CombinePrice(intPart int64, frac int32) float64 {
	return float64(intPart) + float64(frac)/1e9
}

func SplitPrice(price float64) (int64, int32) {
	frac, intPart := math.Modf(price)

	return int64(frac), int32(math.Round(intPart * 1e9))
}

func CreateQuotation(units int64, nano int64) *pkgmodel.Quotation {
	return &pkgmodel.Quotation{
		Units: units,
		Nano:  int32(nano),
	}
}

func CreateInternalQuotation(units int64, nano int64) model.Quotation {
	return model.Quotation{
		Units: units,
		Nano:  int32(nano),
	}
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

// Вспомогательная функция для экранирования специальных символов MarkdownV2
func EscapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// CalculateWeightedAverage принимает слайс чисел и возвращает взвешенное среднее,
// где последние элементы имеют больший вес.
func CalculateWeightedAverage(numbers []float64) float64 {
	var weightedSum float64
	var totalWeight float64

	// создаем веса так, чтобы последние имели больший вес
	for i, num := range numbers {
		weight := float64(i + 1) // веса: 1, 2, 3, ...
		weightedSum += num * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0 // на случай пустого слайса
	}

	num, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", weightedSum/totalWeight), 64)

	return num
}
