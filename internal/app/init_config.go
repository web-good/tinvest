package app

import (
	"context"
	"fmt"
	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/joho/godotenv"
	"tinvest/internal/config"
)

func (a *App) initConfig(ctx context.Context) error {
	err := load("./env/local.env")

	if err != nil {
		return fmt.Errorf("failed to load env_file: %w", err)
	}

	cfg := &config.Config{}
	err = confita.NewLoader(
		env.NewBackend(),
	).Load(ctx, cfg)

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	a.config = cfg

	return nil
}

func load(path string) error {
	err := godotenv.Load(path)

	if err != nil {
		return err
	}

	return nil
}
