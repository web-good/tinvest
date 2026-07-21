package rank

import "strings"

// SectorKind — грубая классификация сектора инструмента для секторных поправок
// скоринга. v1 различает только финансовый сектор (банки/финансы), где часть
// фунд-метрик неприменима (нет EBITDA/FCF), от всех остальных.
type SectorKind int

const (
	SectorOther SectorKind = iota
	SectorFinancial
)

// financialSectorCodes — коды сектора Invest API, относимые к финансам.
// Набор откалиброван на живом API (см. cmd/divscreen). Сравнение
// регистронезависимое.
var financialSectorCodes = map[string]struct{}{
	"financial": {},
}

// ClassifySector относит строковый код сектора Invest API к SectorKind.
// Неизвестный или пустой код → SectorOther.
func ClassifySector(sector string) SectorKind {
	if _, ok := financialSectorCodes[strings.ToLower(strings.TrimSpace(sector))]; ok {
		return SectorFinancial
	}
	return SectorOther
}
