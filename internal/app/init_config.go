package app

import (
	"context"
	"fmt"
	"os"
	"tinvest/internal/config"
	"tinvest/pkg/logger"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/joho/godotenv"
)

func (a *App) initConfig(ctx context.Context) error {
	logger.InfoContext(ctx, "Start initializing configuration")

	// APP_ENV is injected as a real environment variable in prod (see Dockerfile),
	// so it is already set before any file is loaded and selects the env file. In
	// dev it is unset here and defined inside local.env. godotenv.Load never
	// overrides an existing variable, so the prod value always wins.
	envFile := "./env/local.env"
	if os.Getenv("APP_ENV") == "prod" {
		envFile = "./env/prod.env"
	}

	err := load(envFile)

	if err != nil {
		return fmt.Errorf("failed to load env_file: %w", err)
	}

	if os.Getenv("APP_ENV") == "dev" {
		err = load("./env/token.env")

		if err != nil {
			return fmt.Errorf("failed to load token_env_file: %w", err)
		}
	}

	cfg := &config.Config{
		AppName:        "T-invest",
		GrpcClient:     config.NewGrpcClientConfig(),
		TelegramClient: config.NewTelegramClientConfig(),
		PortfolioYield: config.NewPortfolioYieldConfig(),
		Reversion:      config.NewReversionConfig(),
		RSIPullback:    config.NewRSIPullbackConfig(),
		News:           config.NewNewsConfig(),
	}
	err = confita.NewLoader(
		env.NewBackend(),
	).Load(ctx, cfg)

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	a.config = cfg
	logger.InfoContext(ctx, "Finish initializing configuration")

	return nil
}

func load(path string) error {
	err := godotenv.Load(path)

	if err != nil {
		return err
	}

	return nil
}
