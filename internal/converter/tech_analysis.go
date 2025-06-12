package converter

import (
	"fmt"
	"time"
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

func ConvertCandlesTechAnalysisFromPb(techAnalysisItems []*investapi.HistoricCandle) []*model.CandleItemTechAnalyse {
	res := make([]*model.CandleItemTechAnalyse, 0, len(techAnalysisItems))

	for _, item := range techAnalysisItems {
		res = append(res, convertCandlesTechAnalyseFromBp(item))
	}

	return res
}

func convertCandlesTechAnalyseFromBp(item *investapi.HistoricCandle) *model.CandleItemTechAnalyse {
	loc, _ := time.LoadLocation("Europe/Moscow")
	fmt.Println(item)
	return &model.CandleItemTechAnalyse{
		Time:       item.Time.AsTime().In(loc),
		Open:       model.Quotation{},
		Close:      model.Quotation{},
		High:       model.Quotation{},
		Low:        model.Quotation{},
		IsComplete: item.IsComplete,
	}
}

func ConvertMacDTechAnalysisFromPb(techAnalysisItems []*investapi.GetTechAnalysisResponse_TechAnalysisItem) []*model.MacDItemTechAnalyse {
	res := make([]*model.MacDItemTechAnalyse, 0, len(techAnalysisItems))

	for _, item := range techAnalysisItems {
		res = append(res, convertMacDTechAnalyseFromBp(item))
	}

	return res
}

func convertMacDTechAnalyseFromBp(item *investapi.GetTechAnalysisResponse_TechAnalysisItem) *model.MacDItemTechAnalyse {
	loc, _ := time.LoadLocation("Europe/Moscow")

	return &model.MacDItemTechAnalyse{
		Date: item.GetTimestamp().AsTime().In(loc),
		SignalLine: model.Quotation{
			Units: item.GetSignal().GetUnits(),
			Nano:  item.GetSignal().GetNano(),
		},
		MacDLine: model.Quotation{
			Units: item.GetMacd().GetUnits(),
			Nano:  item.GetMacd().GetNano(),
		},
	}
}

func ConvertRsiTechAnalysisFromPb(techAnalysisItems []*investapi.GetTechAnalysisResponse_TechAnalysisItem) []*model.RsiItemTechAnalyse {
	res := make([]*model.RsiItemTechAnalyse, 0, len(techAnalysisItems))

	for _, item := range techAnalysisItems {
		res = append(res, convertRsiTechAnalyseFromBp(item))
	}

	return res
}

func convertRsiTechAnalyseFromBp(item *investapi.GetTechAnalysisResponse_TechAnalysisItem) *model.RsiItemTechAnalyse {
	loc, _ := time.LoadLocation("Europe/Moscow")
	
	return &model.RsiItemTechAnalyse{
		Date: item.GetTimestamp().AsTime().In(loc),
		SignalLine: model.Quotation{
			Units: item.GetSignal().GetUnits(),
			Nano:  item.GetSignal().GetNano(),
		},
	}
}

func ConvertEmaTechAnalysisFromPb(techAnalysisItems []*investapi.GetTechAnalysisResponse_TechAnalysisItem) []*model.EmaItemTechAnalyse {
	res := make([]*model.EmaItemTechAnalyse, 0, len(techAnalysisItems))

	for _, item := range techAnalysisItems {
		res = append(res, convertEmaTechAnalyseFromBp(item))
	}

	return res
}

func convertEmaTechAnalyseFromBp(item *investapi.GetTechAnalysisResponse_TechAnalysisItem) *model.EmaItemTechAnalyse {
	return &model.EmaItemTechAnalyse{
		Date: item.GetTimestamp().AsTime(),
		SignalLine: model.Quotation{
			Units: item.GetSignal().GetUnits(),
			Nano:  item.GetSignal().GetNano(),
		},
	}
}

func ConvertBbTechAnalysisFromPb(techAnalysisItems []*investapi.GetTechAnalysisResponse_TechAnalysisItem) []*model.BbItemTechAnalyse {
	res := make([]*model.BbItemTechAnalyse, 0, len(techAnalysisItems))

	for _, item := range techAnalysisItems {
		res = append(res, convertBbTechAnalyseFromBp(item))
	}

	return res
}

func convertBbTechAnalyseFromBp(item *investapi.GetTechAnalysisResponse_TechAnalysisItem) *model.BbItemTechAnalyse {
	return &model.BbItemTechAnalyse{
		Date: item.GetTimestamp().AsTime(),
		MiddleBand: model.Quotation{
			Units: item.GetMiddleBand().GetUnits(),
			Nano:  item.GetMiddleBand().GetNano(),
		},
		LowerBand: model.Quotation{
			Units: item.GetLowerBand().GetUnits(),
			Nano:  item.GetLowerBand().GetNano(),
		},
		UpperBand: model.Quotation{
			Units: item.GetUpperBand().GetUnits(),
			Nano:  item.GetUpperBand().GetNano(),
		},
	}
}
