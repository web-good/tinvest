package converter

import (
	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/model"
)

func ConvertPortfolioFromBp(in *investapi.PortfolioResponse) []model.Position {
	res := make([]model.Position, 0, len(in.Positions))

	for _, item := range in.Positions {
		if item.InstrumentType != "share" {
			continue
		}

		res = append(res, convertPositionsFromBpToPosition(item))
	}

	return res
}

func convertPositionsFromBpToPosition(pos *investapi.PortfolioPosition) model.Position {
	return model.Position{
		Price:         model.Quotation{Nano: pos.CurrentPrice.Nano, Units: pos.CurrentPrice.Units},
		Quantity:      pos.Quantity.Units,
		PurchasePrice: model.Quotation{Nano: pos.AveragePositionPrice.Nano, Units: pos.AveragePositionPrice.Units},
		ShareID:       pos.InstrumentUid,
	}
}
