package app

import (
	"context"
	"fmt"
	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/joho/godotenv"
	"os"
	"tinvest/internal/config"
	"tinvest/pkg/logger"
)

func (a *App) initConfig(ctx context.Context) error {
	logger.InfoContext(ctx, "Start initializing configuration")

	err := load("./env/local.env")

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
	}
	err = confita.NewLoader(
		env.NewBackend(),
	).Load(ctx, cfg)
	fmt.Println("))))))))))))))))))))))))))))))", os.Getenv("APP_ENV"), "***********************", cfg.TelegramClient.Token)
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
