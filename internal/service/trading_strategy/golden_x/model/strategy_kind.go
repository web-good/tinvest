package model

type StrategyKind int

const (
	StrategyKindUnknown StrategyKind = iota
	StrategyKindDividend
	StrategyKindGrowth
)

func (k StrategyKind) Medal() string {
	switch k {
	case StrategyKindDividend:
		return "🥇"
	case StrategyKindGrowth:
		return "🥈"
	default:
		return ""
	}
}
