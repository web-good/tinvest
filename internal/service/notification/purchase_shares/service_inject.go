//go:build wireinject
// +build wireinject

package purchase_shares

func Inject() (*service, error) {
	wire.Build(NewService())

	return new(service), nil
}
