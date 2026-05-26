package model

type TrendStatus int

const (
	TrendUnknown TrendStatus = iota
	TrendWith
	TrendAgainst
)

func (t TrendStatus) Mark() string {
	switch t {
	case TrendWith:
		return "✅"
	case TrendAgainst:
		return "🚫"
	default:
		return ""
	}
}
