package account

import "errors"

var ErrQuantity = errors.New("quantity must be greater than zero")
var ErrNoMany = errors.New("no many on account")
