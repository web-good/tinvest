package bonds

import "time"

// Rung — одна ступень облигационной лесенки: окно погашения и тип бумаг.
type Rung struct {
	IsOfz bool
	From  time.Time
	To    time.Time
}

// DefaultLadder возвращает ступени лесенки для стратегии «надёжные облигации».
// Границы те же, что раньше были захардкожены в trade.go; вынесены сюда, чтобы
// их было видно, тестировать и легко менять.
func DefaultLadder(now time.Time) []Rung {
	return []Rung{
		{IsOfz: true, From: now.AddDate(0, 0, 180), To: now.AddDate(2, 0, 0)},
		{IsOfz: true, From: now.AddDate(2, 0, 0), To: now.AddDate(6, 0, 0)},
		{IsOfz: true, From: now.AddDate(6, 0, 0), To: now.AddDate(16, 0, 0)},
		{IsOfz: false, From: now.AddDate(0, 0, 180), To: now.AddDate(3, 0, 0)},
	}
}
