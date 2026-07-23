# TInvest Project

## Overview
Go-based trading/investment application built around the Tinkoff Invest gRPC API. It implements live trading strategies (Golden X, Reversion, Bonds screening), plus a shared scalping core (model/strategy packages) used by live Reversion and the backtest engine. Analyzes the portfolio and sends notifications via Telegram bots.

## Tech Stack
- Go 1.25
- gRPC (Tinkoff Invest API) for market data and orders
- Telegram Bot API for notifications
- Code generation: `protoc-gen-go`, `protoc-gen-go-grpc`, `goverter` (see `Makefile`); `mockery` (v2) for test mocks (see `.mockery.yaml`)
- Build/quality tooling: `mage` (`magefiles/`) drives lint/test/mocks; `golangci-lint` v2 (`.golangci.yml`). See `docs/tooling/mage.md`.

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
  - `trading_strategy/` — `golden_x`, `reversion`, `bonds` (live strategies); `scalping/model/` and `scalping/strategy/` (shared core used by reversion (live) and the backtest engine; scalping live layer removed); `scalping_rsimacd/` — backtest-only 5m long RSI+MACD scalper (`-strategy scalping_rsimacd`, docs: `docs/scalping_rsimacd/strategy.md`). `rsi_ema` — backtest-only 15m long RSI+EMA trend scalper (`-strategy rsi_ema`, docs: `docs/rsi_ema/strategy.md`). Note: backtest-only strategies `levels`, `momentum`, `smc` were removed after walk-forward validation failed (2026-07-20).
  - `instrument/` — indicator calculators: `atr`, `ema`, `macd`, `rsi`, `volatility`
  - `notification/purchase_shares`
  - `portfolio/analyze`
  - `screener/dividend/` — дивидендный фундаментальный скринер: чистое ядро ранжирования (`rank/`), кэш-провайдер, бот-команда `/dividend_screener` и фунд-бонус `+0..3` к buy-Score Golden X; см. `docs/dividend/screener.md`
  - `news/` — ежечасный дайджест новостей рынка (RSS smart-lab) в тему Telegram; `news/scheduler/` — cron-обёртка
- `internal/service/trading_strategy/golden_x/` — subpackages: `dto`, `factory`, `model`, `notification`, `percentile`, `scheduler`, `shares`.
- `internal/converter` — DTO/model converters (goverter-generated).
- `internal/pb` — generated protobuf/gRPC stubs (`api/v1/*.proto`).
- `internal/enum`, `internal/model`, `internal/utils` — shared types/helpers.
- `pkg` — reusable packages: `client` (grpc, telegram, rss), `closer`, `collection`, `heartbeat`, `indicators`, `logger`, `scheduler`, `semaphore`.
- `api/v1` — `.proto` definitions for Tinkoff Invest API.
- `magefiles/` — mage build package: `Tools`, `Lint`, `Test`, `Mocks`, `MocksCheck`, `CI` targets.
- `docs/golden_x/` — strategy documentation (strategy, settings, backtest); `docs/dividend/screener.md` — dividend screener reference; `docs/tooling/mage.md` — build/lint/mocks tooling.

## How to Run
1. Copy `env/local.env.example` to `env/local.env` and fill in required values (tokens, etc.).
2. Run: `go run ./cmd/main`.

## Development Notes
- `APP_ENV` selects mode: `dev` runs strategy workers as goroutines (`internal/app/app.go:runDev`); `prod` schedules workers via the `pkg/scheduler` cron-like scheduler.
- Telegram bot integration is used for sending strategy/portfolio notifications.
- Code generation entry points live in `Makefile` (e.g. `make generate`, plus per-service `generate-*-api` targets); deps installed into `./bin` via `make install-deps`.
- Quality checks run through `mage` (from repo root): `./bin/mage tools` installs pinned `golangci-lint`/`mockery` into `./bin`; `./bin/mage ci` runs lint + `go test -race ./...` + mock-drift check — the same gate CI enforces before build/deploy. Regenerate mocks with `./bin/mage mocks` after changing a mocked interface. Note: `go build ./...` fails on the `magefiles` package (no `main`); use `go build ./internal/... ./pkg/... ./cmd/...`. Details: `docs/tooling/mage.md`.
- Golden X strategy settings are centralized in `golden_x/model.Settings` with `DefaultSettings()` constructor. Algorithm knobs are exported fields; fetch-policy constants (`candleLookbackWeeks`, `divergenceFractalK`) remain in-package.
- Notifications go to forum topics of a Telegram supergroup (`TELEGRAM_GROUP_CHAT_ID`, `TELEGRAM_TOPIC_*`); portfolio reports are pulled on demand via bot commands (`/bonds_portfolio`, `/yield`, `/bonds_screener`, `/dividend_screener`); push-дайджест новостей рынка приходит в тему «Новости» раз в час (`TELEGRAM_TOPIC_NEWS`, источник `NEWS_FEED_URL`, дефолт — RSS smart-lab); см. `docs/superpowers/specs/2026-07-15-news-digest-design.md` — see `docs/superpowers/specs/2026-07-13-telegram-topic-routing-design.md`.

## Configuration
Loaded via `heetch/confita` from environment variables and/or files. See `internal/config/config.go` for the full schema.

go run ./cmd/backtest -ticker NVTK -strategy reversion -calibrate data/params/nvtk/reversion_grid.json -out ./reports/NVTK -months 24 -min-trades 20 -test-months 6 -metric profit_factor
