# TInvest Project

## Overview
This is a Go-based trading/investment application that implements various trading strategies (MACD-RSI, SuperTrend, EMA200, Bonds, Golden X) and interacts with Telegram bots for notifications.

## Tech Stack
- Go 1.25
- gRPC for internal service communication
- PostgreSQL (via sqlx) for data storage
- Telegram Bot API for notifications
- Docker for containerization (docker-compose.yaml present)

## Key Directories
- `internal/app`: Application initialization and lifecycle
- `internal/config`: Configuration loading
- `internal/domain`: Core domain models (shares, bonds, technical indicators)
- `internal/service`: Business logic for trading strategies and services
- `internal/service_provider`: Dependency injection / service provider
- `pkg`: Shared packages (logger, closer, gRPC clients)
- `cmd`: Entry point (main.go)

## How to Run
1. Copy `env/local.env.example` to `env/local.env` and fill in required values
2. Ensure PostgreSQL is running (see docker-compose.yaml)
3. Run: `go run ./cmd/main`

## Development Notes
- The application can run in `dev` or `prod` mode (set via APP_ENV)
- In dev mode, multiple strategy workers run via goroutines (see internal/app/app.go:runDev)
- In prod mode, specific workers are scheduled via cron-like schedulers
- Telegram bot integration is used for sending notifications

## Configuration
Configuration is loaded via heetch/confita from environment variables and/or config files.
See internal/config/config.go for structure.

## Database
Migrations are in the `migrations` directory.