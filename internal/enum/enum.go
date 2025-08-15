package enum

import (
	"fmt"
	"time"
)

type Interval int

const (
	Day1      Interval = 5
	Hour1     Interval = 4
	Hour4     Interval = 11
	Minutes15 Interval = 3
	Week1     Interval = 12
)

var intervalNames = map[Interval]string{
	5:  "Day1",
	4:  "Hour1",
	11: "Hour4",
	3:  "Minutes15",
	12: "Week1",
}

func (i Interval) String() string {
	if name, ok := intervalNames[i]; ok {
		return name
	}

	return fmt.Sprintf("Unknown(%d)", i)
}

func (i Interval) ToTimeDuration() time.Duration {
	interval := time.Hour
	switch i {
	case Day1:
		interval = time.Hour * 24
	case Hour1:
		interval = time.Hour
	case Minutes15:
		interval = time.Minute * 15
	case Week1:
		interval = time.Hour * 24 * 7

	}

	return interval
}

func (i Interval) ToNumberInvestApi() int32 {
	if i == Minutes15 {
		return 3
	}

	if i == Hour1 {
		return 4
	}

	if i == Hour4 {
		return 11
	}

	if i == Day1 {
		return 5
	}

	if i == Week1 {
		return 12
	}

	return 0
}
