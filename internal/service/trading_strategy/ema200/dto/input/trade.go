package input

type Interval int

const (
	Day1 Interval = iota
	Hour1
	Hour4
)

type Trade struct {
	Interval Interval
}
