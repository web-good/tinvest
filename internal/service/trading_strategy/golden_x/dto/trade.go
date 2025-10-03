package dto

import (
	"tinvest/internal/enum"
)

type Share struct {
	ID              string
	Name            string
	RSILength       int
	AverageDevident float64
}

type Trade struct {
	Interval  enum.Interval
	ShareList []Share
	Scheduler string
	HeartBeat <-chan struct{}
}
