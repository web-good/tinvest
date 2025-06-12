package converter

import (
	"tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
)

func ConvertSharesFromPb(shares []*investapi.Share) []*model.Share {
	itemCount := 0
	for _, share := range shares {
		if share.Currency == "rub" {
			itemCount++
		}
	}

	res := make([]*model.Share, 0, itemCount)

	for _, share := range shares {
		if share.Currency != "rub" {
			continue
		}

		if share.Uid == "" {

		}
		res = append(res, ConvertShareFromPb(share))
	}

	return res
}

func ConvertShareFromPb(share *investapi.Share) *model.Share {
	return &model.Share{
		Figi:     share.Figi,
		Ticker:   share.Ticker,
		Isin:     share.Isin,
		Lot:      share.Lot,
		Currency: share.Currency,
		Name:     share.Name,
		ID:       share.Uid,
	}
}
