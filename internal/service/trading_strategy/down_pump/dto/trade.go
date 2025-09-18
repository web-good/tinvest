package dto

import (
	"time"
	"tinvest/internal/enum"
)

type Trade struct {
	AtrInterval enum.Interval
	TimeFrame   time.Duration
	Interval    enum.Interval
	Scheduler   string
}
