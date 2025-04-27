package purchase_shares

import (
	"context"
)

var _ Service = (*service)(nil)

type service struct{}

type Service interface {
	PurchaseByStrategy(context.Context) error
}

func NewService() *service {
	return &service{}
}
