package converter

import (
	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/model"
)

func ConvertPortfolioFromBp(in *investapi.PortfolioResponse) []*model.Position {
	res := make([]*model.Position, 0, len(in.Positions))

	for _, item := range in.Positions {
		if item.InstrumentType != "share" && item.InstrumentType != "bond" {
			continue
		}

		res = append(res, convertPositionsFromBpToPosition(item))
	}

	return res
}

func convertPositionsFromBpToPosition(pos *investapi.PortfolioPosition) *model.Position {
	return &model.Position{
		InstrumentType: pos.InstrumentType,
		Figi:           pos.Figi,
		Price:          model.Quotation{Nano: pos.CurrentPrice.Nano, Units: pos.CurrentPrice.Units},
		Quantity:       pos.Quantity.Units,
		PurchasePrice:  model.Quotation{Nano: pos.AveragePositionPrice.Nano, Units: pos.AveragePositionPrice.Units},
		ShareID:        pos.InstrumentUid,
	}
}

func ConvertOperationFromBp(in []*investapi.Operation) []*model.Operation {
	res := make([]*model.Operation, 0, len(in))

	for _, item := range in {
		if item.InstrumentType != "share" {
			continue
		}

		res = append(res, convertPositionFromBpToOperation(item))
	}

	return res
}

func convertPositionFromBpToOperation(op *investapi.Operation) *model.Operation {
	return &model.Operation{
		Date:         op.Date.AsTime(),
		InstrumentID: op.InstrumentUid,
		Type:         op.OperationType.String(),
	}
}
