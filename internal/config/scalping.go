package config

type ScalpingConfig struct {
	AccountID string `config:"SCALPING_ACCOUNT_ID,required,backend=env"`
}

func NewScalpingConfig() *ScalpingConfig {
	return &ScalpingConfig{}
}
