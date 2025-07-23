package converter

import (
	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/model"
)

func ConvertAccountsFromBp(in *investapi.GetAccountsResponse) []model.Account {
	res := make([]model.Account, 0, len(in.Accounts))

	for _, item := range in.Accounts {
		res = append(res, convertAccountFromBpToAccount(item))
	}

	return res
}

func convertAccountFromBpToAccount(ac *investapi.Account) model.Account {
	return model.Account{
		ID:   ac.Id,
		Name: ac.Name,
	}
}
