package enum

import "fmt"

type Interval int

const (
	Day1  Interval = 5
	Hour1 Interval = 4
	Hour4 Interval = 11
)

var intervalNames = map[Interval]string{
	5:  "Day1",
	4:  "Hour1",
	11: "Hour4",
}

func (i Interval) String() string {
	if name, ok := intervalNames[i]; ok {
		return name
	}

	return fmt.Sprintf("Unknown(%d)", i)
}
