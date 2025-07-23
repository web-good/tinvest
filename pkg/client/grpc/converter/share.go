package converter

import (
	share "tinvest/internal/domain/share"
	investapi "tinvest/internal/pb/v1"
)

func ConvertShareFromPb(sh *investapi.Share) *share.Share {
	return &share.Share{
		Figi:     sh.Figi,
		Ticker:   sh.Ticker,
		Isin:     sh.Isin,
		Lot:      sh.Lot,
		Currency: sh.Currency,
		Name:     sh.Name,
		ID:       sh.Uid,
	}
}
