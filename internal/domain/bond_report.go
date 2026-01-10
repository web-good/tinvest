package domain

import "time"

type TypeEnumBond string

const (
	OfzBondEnum  TypeEnumBond = "ОФЗ"
	CorpBondEnum TypeEnumBond = "Корпоративная"
)

type BondReport struct {
	Name          string
	FinalSum      float64
	PercentByYear float64
	ManyByYear    float64
	Nkd           float64
	ExecutionDate time.Time
	Type          TypeEnumBond
}
