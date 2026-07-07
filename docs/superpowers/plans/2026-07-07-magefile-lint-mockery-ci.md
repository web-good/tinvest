# Magefile + golangci-lint + mockery + CI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `magefiles/` build package as the single entry point for `lint` (golangci-lint), `test`, and `mocks` (mockery), migrate hand-written test fakes to mockery mocks, and run all checks in CI as a gate before build/deploy.

**Architecture:** A `magefiles/` main package exposes mage targets (`Tools`, `Lint`, `Test`, `Mocks`, `MocksCheck`, `CI`). golangci-lint and mockery are pinned and installed into `./bin` (matching the existing `make install-deps` pattern). CI gets a `checks` job that runs `mage ci` on every PR/push; `image-build-and-push` and `deploy-image` are gated on it via `needs:` and only run on push to `main`.

**Tech Stack:** Go 1.25, [magefile/mage](https://magefile.org), golangci-lint v2, mockery v2, GitHub Actions.

## Global Constraints

- Go version: **1.25** (base toolchain locally is go1.23.3; go1.25 is pulled via `GOTOOLCHAIN=auto` — CI uses `actions/setup-go` with `go-version: '1.25'`).
- golangci-lint pinned to **v2.12.2** (must be built with go1.25; v2.1.6 built with go1.23 fails with "language version used to build golangci-lint is lower than targeted Go version").
- mockery pinned to **v2.53.4** (v2 line; `packages:` config schema). Verify this is the latest v2 tag at execution and bump if needed — do NOT jump to mockery v3 (different config schema, out of scope).
- All tools install into `./bin` via `GOBIN=$(CURDIR)/bin` — never into `$GOPATH/bin`.
- `mage test` runs `go test -race ./...` and must stay green at every task boundary.
- Generated code is excluded from lint: `internal/pb`, `internal/converter`, `pkg/client/grpc/converter`, and the mockery output dir.
- Commit after every task. Conventional-commit messages. End commit messages with:
  `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- Work happens on branch `feat/magefile-lint-mockery-ci` (already created).

---

## Phase 1 — Tooling foundation (mage + golangci-lint + CI gate)

Phase 1 is independently shippable: it delivers a working lint/test gate in CI without touching mockery.

### Task 1: `magefiles/` package with Tools, Lint, Test targets

**Files:**
- Create: `magefiles/mage.go`
- Modify: `Makefile` (add `install-deps` line for pinned lint/mock tools is optional; mage `Tools` target is authoritative)

**Interfaces:**
- Produces: mage targets invocable as `mage tools`, `mage lint`, `mage test` from repo root (mage lowercases the exported Go func name).

- [ ] **Step 1: Create the magefiles package**

Create `magefiles/mage.go`. In a `magefiles/` directory the package is a normal `package main` with **no** `//go:build mage` tag (that tag is only for the single-file root-magefile layout).

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

// pinned tool versions — keep in sync with the CI workflow and the plan's Global Constraints.
const (
	golangciLintVersion = "v2.12.2"
	mockeryVersion      = "v2.53.4"
)

// binDir is the repo-local tool directory (./bin), matching `make install-deps`.
func binDir() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "bin")
}

// goInstall runs `go install` with GOBIN pointed at ./bin.
func goInstall(pkg string) error {
	return sh.RunWith(map[string]string{"GOBIN": binDir()}, "go", "install", pkg)
}

// Tools installs golangci-lint and mockery into ./bin at pinned versions.
func Tools() error {
	if err := goInstall("github.com/golangci/golangci-lint/v2/cmd/golangci-lint@" + golangciLintVersion); err != nil {
		return fmt.Errorf("install golangci-lint: %w", err)
	}
	if err := goInstall("github.com/vektra/mockery/v2@" + mockeryVersion); err != nil {
		return fmt.Errorf("install mockery: %w", err)
	}
	return nil
}

// Lint runs golangci-lint over the whole module.
func Lint() error {
	return sh.RunV(filepath.Join(binDir(), "golangci-lint"), "run", "./...")
}

// Test runs the full test suite with the race detector.
func Test() error {
	return sh.RunV("go", "test", "-race", "./...")
}
```

- [ ] **Step 2: Ensure the mage dependency is available**

`github.com/magefile/mage` must be a module dependency so `magefiles/` compiles.

Run: `go get github.com/magefile/mage@latest && go mod tidy`
Expected: `go.mod` gains `github.com/magefile/mage`; no errors.

- [ ] **Step 3: Verify targets are discovered**

Run: `./bin/mage -l` (mage is already installed in `./bin` via the existing `make install-deps`)
Expected: lists `tools`, `lint`, `test`.

- [ ] **Step 4: Install the tools**

Run: `./bin/mage tools`
Expected: `./bin/golangci-lint` and `./bin/mockery` exist. Verify:
`./bin/golangci-lint version` → `2.12.2 ... built with go1.25.x`
`./bin/mockery --version` → `v2.53.4`

- [ ] **Step 5: Smoke-test Test target**

Run: `./bin/mage test`
Expected: PASS (all existing tests green). This is the baseline before any lint fixes.

- [ ] **Step 6: Commit**

```bash
git add magefiles/mage.go go.mod go.sum
git commit -m "build(mage): add magefiles package with tools/lint/test targets"
```

---

### Task 2: golangci-lint starter config

**Files:**
- Create: `.golangci.yml`

**Interfaces:**
- Produces: `./bin/golangci-lint run ./...` runs the curated starter set. `mage lint` (Task 1) consumes it.

- [ ] **Step 1: Write the config**

Create `.golangci.yml`. This is the v2 schema. The `revive` ruleset is restricted to `var-naming` only — the default revive "exported symbol should have a comment" / "package-comments" rules produce ~48 low-value doc-comment findings across this codebase and are deferred to a future iteration.

```yaml
version: "2"

run:
  timeout: 5m

linters:
  enable:
    - bodyclose
    - misspell
    - revive
    - unconvert
  settings:
    revive:
      rules:
        - name: var-naming
        - name: unreachable-code
        - name: error-return
        - name: context-as-argument
  exclusions:
    paths:
      - internal/pb
      - internal/converter
      - pkg/client/grpc/converter
    rules:
      # tests intentionally ignore some error returns
      - path: _test\.go
        linters:
          - errcheck

formatters:
  enable:
    - gofmt
    - goimports
  exclusions:
    paths:
      - internal/pb
      - internal/converter
      - pkg/client/grpc/converter
```

- [ ] **Step 2: Run lint to see the finding set**

Run: `./bin/mage lint`
Expected: FAIL, roughly 55–60 issues, dominated by `staticcheck` (~45), `unused` (5), `errcheck` (3), `gofmt` (3), `ineffassign` (1), `revive` var-naming (1–2). No `revive` "should have comment" findings (rule disabled).

- [ ] **Step 3: Commit the config (still red — fixed in Task 3)**

```bash
git add .golangci.yml
git commit -m "build(lint): add golangci-lint v2 starter config"
```

---

### Task 3: Fix all lint findings to green

Fix findings in small, reviewable commits by category. Run `./bin/mage lint` after each group. **Do not** suppress findings with `//nolint` unless a fix is genuinely unsafe — prefer real fixes.

**Files (modify):** the files reported by `mage lint`. Notable groups below.

- [ ] **Step 1: gofmt formatting**

Run: `gofmt -w internal/app/app.go internal/service/trading_strategy/reversion/strategy/afks/afks.go internal/service/trading_strategy/reversion/strategy/sfin/sfin.go`
(These are the 3 gofmt findings. Note `internal/app/app.go:245` is inside a commented-out block — gofmt still reformats it.)
Run: `./bin/golangci-lint run ./... 2>&1 | grep gofmt` → no output.
Commit: `git commit -am "style: gofmt formatting"`

- [ ] **Step 2: staticcheck S1002 (bool comparisons) and S1008 (if/return)**

These are mechanical simplifications across `internal/service/trading_strategy/**/specification/*.go`, `bonds/pipeline/*.go`, `super_trend/trade.go`, `scalping_rsi/*.go`, `ema200/*.go`, `pkg/client/grpc/*.go`. Apply each as suggested by staticcheck:
- `x == true` → `x`; `x == false` / `x != true` → `!x`.
- `if cond { return true }; return false` → `return cond`.

After editing, run: `./bin/golangci-lint run ./... 2>&1 | grep -E "S1002|S1008"` → no output.
Run: `./bin/mage test` → PASS (these are behavior-preserving).
Commit: `git commit -am "refactor: simplify boolean expressions and returns (staticcheck)"`

- [ ] **Step 3: staticcheck quick-fixes (QF1005, QF1012, S1021, S1024, S1039, QF1001)**

Apply each mechanically:
- `math.Pow(x, 2)` → `x*x` (volatility/calculate.go:51).
- `WriteString(fmt.Sprintf(...))` → `fmt.Fprintf(builder, ...)` (bonds/notification/telegram.go).
- `var x T; x = y` → `x := y` (operations_service_client.go:39, users_service_client.go:33).
- `dateTo.Sub(time.Now())` → `time.Until(dateTo)` (bonds/pipeline/sender.go:27).
- `fmt.Sprintf("literal")` with no args → plain string (analyze.go:72,77).
- De Morgan simplifications in `golden_x/notification/notifications_test.go:42,426`.

Run: `./bin/golangci-lint run ./... 2>&1 | grep -E "QF|S1021|S1024|S1039"` → no output.
Run: `./bin/mage test` → PASS.
Commit: `git commit -am "refactor: apply staticcheck quick-fixes"`

- [ ] **Step 4: SA4009 — ctx overwritten before use (real bug)**

In `pkg/client/grpc/instruments_service_client.go`, three methods (`Bonds`, `Shares`, `ShareByID`) take a `ctx context.Context` parameter but immediately overwrite it with `context.WithTimeout(context.Background(), ...)`, discarding the caller's context (cancellation/deadline lost). Fix: derive the timeout from the passed-in ctx.

Change each occurrence from:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
```
to:
```go
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
```

Run: `./bin/golangci-lint run ./... 2>&1 | grep SA4009` → no output.
Run: `./bin/mage test` → PASS.
Commit: `git commit -am "fix(grpc): honor caller context in instruments client timeouts"`

- [ ] **Step 5: unused (5 findings)**

Remove the dead symbols reported: `internal/app/init_grpc_client.go:9` const `address`; `ema200/specification/macd_specification.go:8` field `differenceValue`; `super_trend/specification/green_mac_d.go:10` field `countGreenCandles`; `super_trend/specification/macd_specification.go:8` field `differenceValue`; `service_provider/service.go:24` field `macdRsiTradingService`.

For struct fields, remove the field and any now-unused struct literal keys. If removing `macdRsiTradingService` breaks a constructor/getter, trace and remove the dead wiring too (run `go build ./...` to confirm).

Run: `./bin/golangci-lint run ./... 2>&1 | grep unused` → no output.
Run: `go build ./... && ./bin/mage test` → PASS.
Commit: `git commit -am "refactor: remove unused symbols (staticcheck unused)"`

- [ ] **Step 6: errcheck (3) and ineffassign (1)**

errcheck in `reversion/live/statestore/statestore.go:73,74,78`: these are cleanup calls (`tmp.Close()`, `os.Remove(tmpName)`) on the temp-file error path. Handle explicitly — capture and log/join, or assign to `_ =` only where the error is genuinely irrelevant (temp cleanup on an already-failing path is a legitimate `_ =`). Prefer `_ = os.Remove(tmpName)` for best-effort cleanup.

ineffassign in `super_trend/take_profit.go:39`: `atr, err := ...` where `err` is never checked before being reassigned. Add the missing `if err != nil { ... }` check (this is a latent bug — an ATR fetch error is silently ignored).

Run: `./bin/golangci-lint run ./... 2>&1 | grep -E "errcheck|ineffassign"` → no output.
Run: `./bin/mage test` → PASS.
Commit: `git commit -am "fix: handle ignored errors in statestore and super_trend take-profit"`

- [ ] **Step 7: revive var-naming (Id → ID)**

`pkg/client/grpc/model/bond.go:6` field `Id` → `ID`. This is an exported field on a model struct; update all references (`grep -rn "\.Id\b" --include=*.go` scoped to bond usage) and any converter mappings. Rebuild to confirm.

Run: `go build ./... && ./bin/golangci-lint run ./... 2>&1 | grep var-naming` → no output.
Run: `./bin/mage test` → PASS.
Commit: `git commit -am "refactor(model): rename Bond.Id to Bond.ID (revive var-naming)"`

- [ ] **Step 8: Verify fully green**

Run: `./bin/mage lint`
Expected: exit 0, `0 issues`.

---

### Task 4: CI checks job gating build/deploy

**Files:**
- Modify: `.github/workflows/main.yaml`

**Interfaces:**
- Produces: a `checks` job; `image-build-and-push` gains `needs: checks` + a push-to-main condition.

- [ ] **Step 1: Add triggers and the checks job**

At the top of `.github/workflows/main.yaml`, widen triggers:

```yaml
on:
  push:
    branches: [ main, master, "**" ]
  pull_request:
```

Add a `checks` job (before `image-build-and-push`):

```yaml
jobs:
  checks:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true
      - name: Install tools
        run: go run github.com/magefile/mage tools
      - name: Lint + Test
        run: go run github.com/magefile/mage lint test
```

(Phase 2 changes the last step to `... mage ci` once `MocksCheck` exists. Using `go run github.com/magefile/mage` avoids needing a pre-installed mage binary in CI.)

- [ ] **Step 2: Gate build and deploy on checks**

Add to the `image-build-and-push` job:

```yaml
  image-build-and-push:
    runs-on: ubuntu-latest
    needs: checks
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      # ... unchanged ...
```

Leave `deploy-image` as-is (`needs: image-build-and-push`) — it inherits the gate transitively.

- [ ] **Step 3: Validate workflow syntax**

Run: `go run github.com/magefile/mage lint test` locally (mirrors what CI runs) → PASS.
If `actionlint` is available: `actionlint .github/workflows/main.yaml` → no errors. Otherwise verify YAML parses: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/main.yaml'))"`.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/main.yaml
git commit -m "ci: run mage lint+test on PR/push and gate build+deploy"
```

Phase 1 is complete and shippable here: CI now blocks build/deploy on lint+test.

---

## Phase 2 — mockery + fake migration

Phase 2 introduces mockery, generates mocks, migrates hand-written fakes package-by-package (tests green at every step), and wires the drift check into CI.

**Migration inventory (in scope):**

| Package | Test file | Fake(s) | Interface to mock |
|---|---|---|---|
| `internal/service/portfolio/yield` | `yield_test.go` | `fakeOperationsClient`, `fakeUsersClient`, `fakeTelegramClient` | `grpc.OperationsServiceClient`, `grpc.UsersServiceClient`, `telegram.Client` |
| `internal/service/trading_strategy/reversion/live` | `service_test.go` | `fakeInstruments`, `fakeMarket`, `fakeOps`, `fakeTg` | the parameter interfaces of `live.NewService` (read `live.go`) |
| `.../reversion/live/executor` | `executor_test.go` | `fakeOrders` | `executor.OrdersClient` |
| `.../reversion/live/marketdata` | `marketdata_test.go` | `fakeCandleClient`, `fakeHTFCandleClient` | `marketdata.CandleClient` |
| `.../reversion/live/reconstruct` | `reconstruct_test.go` | `fakeTrades`, `fakeCandles` | `reconstruct.TradesClient` (+ its candle client iface) |
| `.../reversion/live/scheduler` | `scheduler_test.go` | `fakeSvc` | `live.Service` |
| `internal/service/trading_strategy/scalping` | `trade_test.go` | `fakeStrategy` | `strategy.Strategy` |
| `internal/service/backtest` | `candles_test.go` | `fakeFetcher` | the fetcher interface consumed by the code under test (read `candles.go`) |

**Explicitly OUT of scope (documented exceptions, keep hand-written):**
- `internal/service/backtest/walkforward_test.go` `fakeParams` — a generics test-data struct, not an interface.
- `pkg/client/grpc/orders_auth_test.go` `fakeOrdersAPI` — mocks the generated `investapi.OrdersServiceClient` (a large third-party generated interface) via a single method; mockery adds no value here.

### Task 5: mockery config + Mocks/MocksCheck targets + generate

**Files:**
- Create: `.mockery.yaml`
- Modify: `magefiles/mage.go`
- Create: generated mock files (output dir, e.g. `internal/mocks/...` — confirm layout in Step 1)

**Interfaces:**
- Produces: `mage mocks`, `mage mocksCheck`, `mage ci`; generated mock types (e.g. `MockOrdersClient` with `.EXPECT()` API) importable by tests.

- [ ] **Step 1: Write `.mockery.yaml`**

v2 `packages:` schema, expecter API on, mocks emitted into a per-consumer `mocks` subpackage next to each interface (keeps import paths local and avoids a giant top-level tree). List exactly the in-scope interfaces from the inventory.

```yaml
with-expecter: true
dir: "{{.InterfaceDir}}/mocks"
mockname: "Mock{{.InterfaceName}}"
outpkg: "mocks"
filename: "mock_{{.InterfaceName}}.go"
packages:
  tinvest/pkg/client/grpc:
    interfaces:
      OperationsServiceClient:
      UsersServiceClient:
  tinvest/pkg/client/telegram:
    interfaces:
      Client:
  tinvest/internal/service/trading_strategy/reversion/live:
    config:
      all: false
    interfaces:
      Service:
      # add the NewService parameter interfaces after reading live.go
  tinvest/internal/service/trading_strategy/reversion/live/executor:
    interfaces:
      OrdersClient:
  tinvest/internal/service/trading_strategy/reversion/live/marketdata:
    interfaces:
      CandleClient:
  tinvest/internal/service/trading_strategy/reversion/live/reconstruct:
    interfaces:
      TradesClient:
  tinvest/internal/service/trading_strategy/scalping/strategy:
    interfaces:
      Strategy:
```

Note: before running, read `internal/service/trading_strategy/reversion/live/live.go` (the `NewService` signature) and `internal/service/backtest/candles.go` to fill in the exact consumer-interface names for the `live` and `backtest` fetcher entries. Add a `tinvest/internal/service/backtest` entry for the fetcher interface.

- [ ] **Step 2: Add mage targets**

Append to `magefiles/mage.go`:

```go
// Mocks regenerates all mockery mocks from .mockery.yaml.
func Mocks() error {
	return sh.RunV(filepath.Join(binDir(), "mockery"))
}

// MocksCheck regenerates mocks and fails if the working tree changed (CI drift guard).
func MocksCheck() error {
	if err := Mocks(); err != nil {
		return err
	}
	return sh.RunV("git", "diff", "--exit-code", "--", "**/mocks/")
}

// CI runs the full check suite: lint, test, and mock-drift.
func CI() error {
	if err := Lint(); err != nil {
		return err
	}
	if err := Test(); err != nil {
		return err
	}
	return MocksCheck()
}
```

- [ ] **Step 3: Generate mocks and build**

Run: `./bin/mage mocks`
Expected: mock files created under each interface's `mocks/` subdir.
Run: `go build ./...`
Expected: PASS (generated mocks compile).

- [ ] **Step 4: Exclude generated mocks from lint**

Add `- "**/mocks"` (or the concrete mock dirs) to the `linters.exclusions.paths` and `formatters.exclusions.paths` in `.golangci.yml`.
Run: `./bin/mage lint` → 0 issues.

- [ ] **Step 5: Commit generated mocks + config**

```bash
git add .mockery.yaml magefiles/mage.go .golangci.yml **/mocks/
git commit -m "test(mocks): add mockery config, mage targets, and generated mocks"
```

### Tasks 6–13: Per-package fake → mock migration

**Shared recipe (apply per package, keeping tests green):**
1. Import the generated mock package.
2. Replace each `fakeX{...}` construction with `m := mocks.NewMockX(t)`.
3. Translate the fake's hardcoded return values into expectations: `m.EXPECT().Method(mock.Anything, ...).Return(val, err)` (use `.Maybe()` for calls that may not happen, `.Times(n)` where the old fake counted calls).
4. Pass the mock where the fake was injected (constructor/function under test).
5. Delete the now-unused fake type and its methods.
6. Run the package's tests; they must pass with identical assertions.

Each task below is one package. Do them in any order; each is independent.

- [ ] **Task 6 — `internal/service/trading_strategy/reversion/live/executor`**
  - Interface: `OrdersClient` (single method `PostOrder(ctx, *investapi.PostOrderRequest, ...grpc.CallOption) (*investapi.PostOrderResponse, error)`).
  - Replace `fakeOrders` in `executor_test.go` with `mocks.NewMockOrdersClient(t)`; set `.EXPECT().PostOrder(mock.Anything, mock.Anything).Return(resp, nil)` mirroring the fake's stored response, and assert the captured request via the mock's `RunAndReturn` or a `mock.MatchedBy` if the test checks request fields.
  - Run: `go test -race ./internal/service/trading_strategy/reversion/live/executor/...` → PASS.
  - Commit: `test(executor): migrate fakeOrders to mockery mock`

- [ ] **Task 7 — `.../reversion/live/marketdata`**
  - Interface: `CandleClient`. Replace `fakeCandleClient` and `fakeHTFCandleClient` with two `mocks.NewMockCandleClient(t)` instances (base + HTF), each returning the candle slices the fakes returned.
  - Run: `go test -race ./internal/service/trading_strategy/reversion/live/marketdata/...` → PASS.
  - Commit: `test(marketdata): migrate candle fakes to mockery mocks`

- [ ] **Task 8 — `.../reversion/live/reconstruct`**
  - Interface: `TradesClient` (+ candle client iface used here). Replace `fakeTrades`/`fakeCandles` with the generated mocks, wiring the trade/candle return values.
  - Run: `go test -race ./internal/service/trading_strategy/reversion/live/reconstruct/...` → PASS.
  - Commit: `test(reconstruct): migrate fakes to mockery mocks`

- [ ] **Task 9 — `.../reversion/live/scheduler`**
  - Interface: `live.Service`. `fakeSvc{calls int}` is a call-counter — replace with `m := mocks.NewMockService(t)` and `m.EXPECT().<Method>(...).Return(...).Times(n)` to assert invocation count instead of the manual counter.
  - Run: `go test -race ./internal/service/trading_strategy/reversion/live/scheduler/...` → PASS.
  - Commit: `test(scheduler): migrate fakeSvc to mockery mock`

- [ ] **Task 10 — `.../reversion/live` (`service_test.go`)**
  - Interfaces: the parameter interfaces of `live.NewService` (Instruments/Market/Ops/Tg — confirm exact names in `live.go`). Replace `fakeInstruments`, `fakeMarket`, `fakeOps`, `fakeTg` with the corresponding generated mocks, wiring each return.
  - Run: `go test -race ./internal/service/trading_strategy/reversion/live/...` → PASS.
  - Commit: `test(live): migrate service_test fakes to mockery mocks`

- [ ] **Task 11 — `internal/service/portfolio/yield`**
  - Interfaces: `grpc.OperationsServiceClient`, `grpc.UsersServiceClient`, `telegram.Client`. Replace `fakeOperationsClient`, `fakeUsersClient`, `fakeTelegramClient` with the generated mocks; the telegram mock captures sent messages via `.EXPECT().SendMessage(mock.MatchedBy(...))` or by asserting on recorded calls.
  - Run: `go test -race ./internal/service/portfolio/yield/...` → PASS.
  - Commit: `test(yield): migrate fakes to mockery mocks`

- [ ] **Task 12 — `internal/service/trading_strategy/scalping`**
  - Interface: `strategy.Strategy`. Replace `fakeStrategy` in `trade_test.go` with `mocks.NewMockStrategy(t)`, wiring the strategy method returns.
  - Run: `go test -race ./internal/service/trading_strategy/scalping/...` → PASS.
  - Commit: `test(scalping): migrate fakeStrategy to mockery mock`

- [ ] **Task 13 — `internal/service/backtest` (`candles_test.go`)**
  - Interface: the fetcher interface consumed by `candles.go` (confirm name). Replace `fakeFetcher` with the generated mock. Leave `walkforward_test.go` `fakeParams` untouched (out of scope).
  - Run: `go test -race ./internal/service/backtest/...` → PASS.
  - Commit: `test(backtest): migrate fakeFetcher to mockery mock`

### Task 14: Wire MocksCheck into CI and final verification

**Files:**
- Modify: `.github/workflows/main.yaml`

- [ ] **Step 1: Switch CI to `mage ci`**

In the `checks` job, change the last step to run the full suite including drift:
```yaml
      - name: Lint + Test + Mock drift
        run: go run github.com/magefile/mage ci
```

- [ ] **Step 2: Verify drift guard locally**

Run: `./bin/mage mocksCheck`
Expected: exit 0 (mocks committed, no drift).
Sanity-check the guard trips: edit an interface method signature, run `./bin/mage mocksCheck`, confirm it FAILS with a non-empty `git diff`, then revert.

- [ ] **Step 3: Full green run**

Run: `./bin/mage ci`
Expected: lint 0 issues, all tests PASS, no mock drift.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/main.yaml
git commit -m "ci: include mockery drift check in the gate (mage ci)"
```

---

## Self-Review Notes

- **Spec coverage:** magefiles package (Task 1) ✓; golangci-lint config + fix-to-green whole repo (Tasks 2–3) ✓; mockery config + targets + generate (Task 5) ✓; fake→mock migration package-by-package (Tasks 6–13) ✓; CI checks job on PR/push gating build+deploy (Task 4) ✓; mock drift in CI (Task 14) ✓; pinned versions in Global Constraints ✓; generated-code lint exclusions ✓.
- **Deviations from spec, flagged:** (1) The spec's starter linter set is realized with `revive` restricted to `var-naming` (the exported-comment rules were deferred — ~48 doc-comment findings of low value; noted as future work). (2) Mock output layout chosen as per-package `mocks/` subpackages (spec left this "confirmed during planning"). (3) Two hand-written fakes documented as out-of-scope exceptions (`fakeParams`, `fakeOrdersAPI`) with rationale.
- **Bonus real bugs surfaced by lint (fixed in Task 3):** SA4009 (caller context dropped in instruments client) and ineffassign (ignored ATR-fetch error in super_trend take-profit).
