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
- `cmd/` — command-line applications; `cmd/main.go` starts the service.
- `internal/app`, `internal/config`, `internal/service_provider` — lifecycle, configuration, and dependency wiring.
- `internal/domain` — domain models, indicators, notifications, and backtesting.
- `internal/service/trading_strategy` — live strategies and backtest-only strategy implementations.
- `docs/` — strategy mechanics, tooling, design specs, and operational guidance.
- `data/params/` — strategy parameter grids and ticker literals.
- Strategy catalog and detailed historical notes: `docs/strategy-catalog.md`. Read the relevant strategy section only when a task needs it; this file is reference material, not default instructions.

## Development Notes
- `APP_ENV` selects mode: `dev` runs strategy workers as goroutines (`internal/app/app.go:runDev`); `prod` schedules workers via the `pkg/scheduler` cron-like scheduler.
- Telegram bot integration is used for sending strategy/portfolio notifications.
- Code generation entry points live in `Makefile` (e.g. `make generate`, plus per-service `generate-*-api` targets); deps installed into `./bin` via `make install-deps`.
- Quality checks run through `mage` (from repo root): `./bin/mage tools` installs pinned `golangci-lint`/`mockery` into `./bin`; `./bin/mage ci` runs lint + `go test -race ./...` + mock-drift check — the same gate CI enforces before build/deploy. Regenerate mocks with `./bin/mage mocks` after changing a mocked interface. Note: `go build ./...` fails on the `magefiles` package (no `main`); use `go build ./internal/... ./pkg/... ./cmd/...`. Details: `docs/tooling/mage.md`.
- Golden X strategy settings are centralized in `golden_x/model.Settings` with `DefaultSettings()` constructor. Algorithm knobs are exported fields; fetch-policy constants (`candleLookbackWeeks`, `divergenceFractalK`) remain in-package.
- Notifications go to forum topics of a Telegram supergroup (`TELEGRAM_GROUP_CHAT_ID`, `TELEGRAM_TOPIC_*`); portfolio reports are pulled on demand via bot commands (`/bonds_portfolio`, `/yield`, `/bonds_screener`, `/dividend_screener`); push-дайджест новостей рынка приходит в тему «Новости» раз в час (`TELEGRAM_TOPIC_NEWS`, источник `NEWS_FEED_URL`, дефолт — RSS smart-lab); см. `docs/superpowers/specs/2026-07-15-news-digest-design.md` — see `docs/superpowers/specs/2026-07-13-telegram-topic-routing-design.md`.
- Все записи уровня ERROR дублируются сообщением в тему «General» Telegram-группы (`internal/service/notification/errorlog`, подключается шагом `initErrorLogSink` в `internal/app`). В stdout поток логов не меняется; в Telegram действуют дедуп по тексту сообщения (5 минут), лимит 10 сообщений в минуту и минутная сводка подавленного. Спека: `docs/superpowers/specs/2026-08-11-error-log-telegram-design.md`.
- Буст доли раннера счёта rsi_pullback (`LAG_BOOST_ENABLED/MULT/PCT/DAYS`, выключен по умолчанию): rsi_pullback и rsi_zone входят увеличенной долей, если тикер за N завершённых дней отстал от IMOEX на порог; gap_fade не затрагивается. Спека: `docs/superpowers/specs/2026-10-08-lag-boost-design.md`, механика — `docs/rsi_pullback/live.md` §8.

## Documentation rules
- `docs/rsi_pullback/` (`strategy.md`, `live.md`, `screener.md`) describes **mechanics only**: strategy logic, tunable parameters, the calibration procedure as a procedure, the live runner and its configuration, and how the screener works. It must NOT contain per-ticker calibration write-ups, run results, profit factors, onboarding dates, or per-ticker risk registers.
- Calibrating a new ticker does not mean rewriting these docs. The per-ticker write-up belongs in the doc comment of `internal/service/trading_strategy/rsi_pullback/strategy/<ticker>/` and in the `_comment` fields of `data/params/rsi_pullback/<ticker>/`.
- A measurement on a specific ticker is acceptable in these docs only as a one-line illustration of a mechanism, never as a record of a calibration run.

## Configuration
Loaded via `heetch/confita` from environment variables and/or files. See `internal/config/config.go` for the full schema.

go run ./cmd/backtest -ticker NVTK -strategy reversion -calibrate data/params/nvtk/reversion_grid.json -out ./reports/NVTK -months 24 -min-trades 20 -test-months 6 -metric profit_factor
