package config

var ConfigPath string

type Config struct {
	MovementEnabled    bool    `config:"MOVEMENT_ENABLED"`
	MovementMOEXPct    float64 `config:"MOVEMENT_MOEX_MIN_PCT"`
	MovementBinancePct float64 `config:"MOVEMENT_BINANCE_MIN_PCT"`
	AppEnv             string  `config:"APP_ENV,backend=env"`
	AppName            string  `config:"APP_NAME,required,backend=env"`
	GrpcClient         *GrpcClient
	TelegramClient     *TelegramClient
	PortfolioYield     *PortfolioYieldConfig
	Reversion          *ReversionConfig
	RSIPullback        *RSIPullbackConfig
	RSIZone            *RSIZoneConfig
	GapFade            *GapFadeConfig
	PDLRSI             *PDLRSIConfig
	FVGImbalance       *FVGImbalanceConfig
	DropBounce         *DropBounceConfig
	PDLReclaim         *PDLReclaimConfig
	News               *NewsConfig
	Density            *DensityConfig
	DensityLive        *DensityLiveConfig `config:"-"`
	DensityLiveError   error              `config:"-"`
}
