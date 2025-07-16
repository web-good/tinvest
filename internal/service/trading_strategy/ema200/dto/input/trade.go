package input

type Interval int

const (
	Day1  Interval = 5
	Hour1 Interval = 4
	Hour4 Interval = 11
)

type Trade struct {
	Interval Interval
}
