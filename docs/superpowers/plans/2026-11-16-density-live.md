# Density live Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement task-by-task.

**Goal:** Отдельный live runner с покупкой одного лота, реальными брокерскими стопом и тейком и отменой входа при потере плотности.

**Architecture:** `density/live` содержит проверяемый последовательный исполнитель, постоянный журнал и независимый поток рынка. `pkg/client/grpc` содержит адаптер торгового API; `cmd/density-live` связывает их, по умолчанию выполняя только read-only проверку.

**Tech Stack:** Go 1.25, существующие protobuf/gRPC, стандартная библиотека для журнала и блокировок.

**Spec:** docs/superpowers/specs/2026-11-16-density-live-design.md (подтверждена пользователем 2026-11-16).

## Global Constraints

- Один лот, одна активная заявка/позиция; явно заданный счёт, маржинальная торговля отключена.
- Стоп и тейк — реальные условные поручения; исполнение только по подтверждению брокера.
- Проверки не отправляют реальные поручения. Боевой запуск выполняет пользователь.
- Сохранять текущий paper-режим и ранее сделанные незакоммиченные изменения.

## Review Focus

- Плотность пропадает во время медленного RPC и возвращается: отмена должна оставаться обязательной.
- Отмена входа одновременно с исполнением: купленное нельзя потерять в учёте.
- Активированный стоп без видимого дочернего ID: нельзя отправлять конкурирующую продажу.
- Перезапуск после записи намерения, но до сохранения ответа: нельзя создавать новый UUID и дубликат.
- Отключённая маржинальная торговля должна подтверждаться API, а не только флагом confirm_margin_trade.

### Task 1: Постоянное состояние и рынок

Files: `internal/service/screener/density/live/{config,model,store,feed}.go`, соответствующие тесты.
Interfaces: `OpenStore(root, account string)`, `Store.Save(State) error`; `Feed.Book`, `Feed.Watch`, `Feed.Snapshot`; broker-independent `Broker` contract.

- [x] Написать тесты одного лота/цен, восстановления журнала, account binding, эксклюзивного flock, защёлки потери плотности и свежести.
- [x] Выполнить тесты и увидеть ожидаемый RED; реализовать минимальные типы и методы; подтвердить GREEN.

### Task 2: Исполнитель жизненного цикла

Files: `internal/service/screener/density/live/{engine,orders}.go`, `engine_test.go`.
Interfaces: `NewEngine(Config, Broker, *Store, *Feed, []density.Instrument)`, `Step(ctx, now) error`, `Shutdown(ctx) error`.

- [x] RED: отмена pending, fill при cancel, защита, уход плотности после fill, завершение обоих стопов, ожидание child ID, отсутствие повторной продажи, неизвестный PostStopOrder, перезапуск и журнал.
- [x] GREEN: последовательные поручения, запись намерений до RPC, подтверждённые terminal-состояния, сверка собственного объёма, блокировка неопределённых эпизодов.

### Task 3: Реальный gRPC адаптер

Files: `pkg/client/grpc/density_broker.go`, тесты с bufconn/подменными generated client.
Interfaces: implements live.Broker; uses generated Orders, StopOrders, Users, Operations clients.

- [x] RED: полный доступ к явному счёту, только точная ошибка 30051 означает disabled margin, один лот, stop market / take limit, UUID, request-ID lookup, проверка денег и чужих заявок.
- [x] GREEN: авторизация, deadlines, точные цены, комиссии и статусы; никаких торговых RPC в preflight.

### Task 4: CLI и поток рынка

Files: `cmd/density-live/{main,main_test}.go`, `internal/service/screener/density/live/runner.go` и тесты.

- [x] RED: help/invalid config без соединений, default check-only, read-only проверки, independent feed vs slow broker, отсев старых книг/повторных событий, остановка и reconnect.
- [x] GREEN: поддержать dotenv, tickers, TLS classic, density overrides, явный account, duration; live только при явном `-trade`.
- [x] Создать `docs/orderbook_density/live.md` с точной командой, поведением остановки, восстановления и ограничениями двух независимых стопов.

### Task 5: Проверка

- [x] `go test -race ./internal/service/screener/density/... ./pkg/client/grpc ./cmd/density-paper ./cmd/density-live`.
- [x] `go vet` затронутых пакетов, `go build ./internal/... ./pkg/... ./cmd/...`, relevant golangci-lint, `git diff --check`.
- [x] Независимая проверка кода по Review Focus, исправление важных замечаний через RED→GREEN.

## Execution ledger

Ruling: implement inline in the existing density worktree without another approval round — user explicitly approved the proposed implementation; no real financial orders are authorized for agent execution.
Ruling: preserve work uncommitted — existing paper implementation and unrelated changes are already uncommitted, and no commit/merge was requested.
Ruling: default command is read-only preflight; explicit `-trade` is the user's activation switch, not a request for another chat approval.


Task 1: complete — initial undefined-symbol RED then GREEN for plan/one lot, sticky density loss, journal replay/corruption and account lock. Persistent state defaults outside checkout in XDG_STATE_HOME or ~/.local/state/tinvest/density-live.
Task 2: complete — cancel/fill race, confirmed protection, stop/take child execution, pending activated take, rejected protection, unknown stop, uncertain market exit, shutdown, accounting and journal failure tests pass.
Task 3: complete — exact margin-disabled API code, full account access, mutation guards, UUID lookup, commission-aware own-cash allocation, one-lot requests, explicit regular take-limit subtype, and recovery metadata by UID tested without network trading calls.
Task 4: complete — CLI help/options, explicit account, default state path, ack/readiness gates, stale trade filtering and market updates during blocked broker RPC tested. Russian operator guide: docs/orderbook_density/live.md.
Task 5: complete — go test -race -count=1 ./internal/service/screener/density/... ./pkg/client/grpc ./cmd/density-paper ./cmd/density-live passed; all six packages passed (density ~21s). go vet affected packages, go build ./internal/... ./pkg/... ./cmd/..., git diff --check passed; relevant golangci-lint reports 0 issues. CLI -help ran without credentials or network calls.

Final independent review: density_live_code_review, read-only fresh context. Three Important findings; no deferred minors.
Final: fixed eligibility-dependent recovery — TestRecoveryUniverseDoesNotApplyEntryFilters RED→GREEN; original metadata resolved by UID before fresh entry filters and recovery performed first.
Final: fixed disappearing stale interval during a slow RPC — TestFeedLatchesStaleIntervalBeforeReplacingBook RED→GREEN; previous book expiry latched before replacement.
Final: fixed initial directory durability — TestDirectorySyncFailsForNonDirectory RED→GREEN; both files and all parent directory chains synced before OpenStore permits trading.
Final: additional verified corrections — one durable accounting/episode-close transition, polling rate on unresolved exits, stable state-root binding, instrument/side/UUID validation, final execution commission refresh, unknown commission flags, protection despite malformed accounting, and no suppression of joined operational errors. Each regression observed RED then GREEN.
Final: fixed stale confirmation during allocation RPC — TestExpiredConfirmationDuringAllocationCheckCannotEnter RED→GREEN; atomically revalidate the same detector observation and install its loss watcher after preflight calls.
Final: fixed integer tick-offset overflow — TestPlanRejectsOverflowingTickOffset RED→GREEN.
Final: explicit regular take subtype — existing request test extended RED→GREEN; affected adapter/CLI race tests rerun.

Ruling: default state outside checkout — preserve recovery when later moving code into the main TInvest worktree; explicit custom state roots remain bound to their absolute path.
Ruling: unknown PostStopOrder is persisted and blocks automation instead of being retried — no verified universal lookup by its request UUID is exposed by the local contract; this can require manual broker reconciliation, but avoids a duplicate protective sell.
Ruling: real API/account behavior remains unverified — no actual order, stop, cancel or account-setting mutation was executed; passing local tests is not a claim of a successful real trade.
