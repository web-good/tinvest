package dto

import (
	"time"
	"tinvest/internal/enum"
)

type BackTest struct {
	AtrInterval  enum.Interval
	TimeFrame    time.Duration
	Interval     enum.Interval
	InstrumentID []string
	RSILength    int32
	MacDLength   int32
	DateFrom     time.Time
	DateTo       time.Time
}
