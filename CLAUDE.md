# TInvest Project

## Overview
Go-based trading/investment application built around the Tinkoff Invest gRPC API. It implements several trading strategies (MACD-RSI, SuperTrend, EMA200, Golden X, Bonds, Scalping RSI), analyzes the portfolio, and sends notifications via Telegram bots.

## Tech Stack
- Go 1.25
- gRPC (Tinkoff Invest API) for market data and orders
- PostgreSQL (via sqlx) for data storage; migrations under `migrations/`
- Telegram Bot API for notifications
- Docker / docker-compose for local infra
- Code generation: `protoc-gen-go`, `protoc-gen-go-grpc`, `goverter` (see `Makefile`)

## Layout
- `cmd/main.go` — entry point; initializes and runs `internal/app`.
- `internal/app` — application initialization and lifecycle (`runDev` / `runProd`).
- `internal/config` — configuration loading via `heetch/confita`.
- `internal/service_provider` — dependency injection / wiring.
- `internal/domain` — core domain models and indicator math:
  - `share/`, `bond_report.go`, `portfolio.go`, `info.go`
  - indicators: `atr/`, `ema/`, `rsi.go`, `macd.go`, `volatility.go`
  - `notification/`, `backtest/`
- `internal/service` — business logic, grouped by concern:
  - `trading_strategy/` — `macd_rsi`, `super_trend`, `ema200`, `golden_x`, `bonds`, `scalping_rsi`
  - `instrument/` — indicator calculators: `atr`, `ema`, `macd`, `rsi`, `volatility`
  - `notification/purchase_shares`
  - `portfolio/analyze`
- `internal/converter` — DTO/model converters (goverter-generated).
- `internal/pb` — generated protobuf/gRPC stubs (`api/v1/*.proto`).
- `internal/enum`, `internal/model`, `internal/utils` — shared types/helpers.
- `pkg` — reusable packages: `client`, `closer`, `collection`, `heartbeat`, `logger`, `scheduler`, `semaphore`.
- `api/v1` — `.proto` definitions for Tinkoff Invest API.
- `migrations/` — SQL migrations (see `migration.sh` / `migration.Dockerfile`).

## How to Run
1. Copy `env/local.env.example` to `env/local.env` and fill in required values (tokens, DB DSN, etc.).
2. Start dependencies (PostgreSQL) via `docker-compose up -d`.
3. Run: `go run ./cmd/main`.

## Development Notes
- `APP_ENV` selects mode: `dev` runs strategy workers as goroutines (`internal/app/app.go:runDev`); `prod` schedules workers via the `pkg/scheduler` cron-like scheduler.
- Telegram bot integration is used for sending strategy/portfolio notifications.
- Code generation entry points live in `Makefile` (e.g. `make generate`, plus per-service `generate-*-api` targets); deps installed into `./bin` via `make install-deps`.

## Configuration
Loaded via `heetch/confita` from environment variables and/or files. See `internal/config/config.go` for the full schema.

## Database
PostgreSQL accessed through `sqlx`. Migrations live in `migrations/` and are applied via `migration.sh` (see `migration.Dockerfile` for the container variant).
