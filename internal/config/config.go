package config

var ConfigPath string

type Config struct {
	AppEnv         string `config:"APP_ENV,backend=env"`
	AppName        string `config:"APP_NAME,required,backend=env"`
	GrpcClient     *GrpcClient
	TelegramClient *TelegramClient
	PortfolioYield *PortfolioYieldConfig
	Reversion      *ReversionConfig
	News           *NewsConfig
}
