package config

type GrpcClient struct {
	AddressProd string
	TokenProd   string `config:"T_BANK"`
}

func NewGrpcClientConfig() *GrpcClient {
	return &GrpcClient{
		AddressProd: "invest-public-api.tinkoff.ru:443",
	}
}
