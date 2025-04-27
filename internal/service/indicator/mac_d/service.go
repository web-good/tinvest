package mac_d

import "context"

type Service interface {
	GetMacD(context.Context) error
}
