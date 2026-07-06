# Reversion Dedicated Trading Token — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the reversion strategy trade under its own Tinkoff API token (`REVERSION_TOKEN`) via a dedicated, isolated gRPC client, and fix the currently-unauthenticated order-placement path.

**Architecture:** Add a required `REVERSION_TOKEN` config field. Build a second `GrpcClient` from that token in the service provider and wire the reversion live service to it, so every Tinkoff call reversion makes runs under its own token. Separately fix `OrdersServiceClient` so `PostOrder` attaches the Bearer credential (today it attaches none, so trading fails under any token).

**Tech Stack:** Go 1.25, gRPC (Tinkoff Invest API), `heetch/confita` config, standard `testing`.

## Global Constraints

- `REVERSION_TOKEN` is **required** (`config:"REVERSION_TOKEN,required,backend=env"`) — no fallback to `T_BANK`.
- `REVERSION_TOKEN` is a **secret**: never commit its value; `env/prod.env` keeps only an empty placeholder / it is injected as a container env var.
- Reuse the existing gRPC address `AddressProd` and constructor `NewClientGrpc(address, token)` — do not add a new endpoint.
- Follow existing test style: gRPC tests are white-box (`package grpc`); config tests assert the constructor at struct level.
- Do not change behavior of other strategies beyond the incidental benefit that `T_BANK`-built orders now authenticate.

---

## File Structure

- `pkg/client/grpc/orders_service_client.go` — orders client gains a stored `*Auth` and attaches it in `PostOrder` (Task 1).
- `pkg/client/grpc/grpc.go` — pass `token` into `NewOrdersServiceClient` at the single call site (Task 1).
- `pkg/client/grpc/orders_auth_test.go` — new white-box tests for the auth wiring (Task 1).
- `internal/config/reversion.go` — add required `Token` field (Task 2).
- `internal/config/reversion_test.go` — assert `Token` has no default (Task 2).
- `internal/service_provider/client.go` — add cached `GetReversionGrpcClient()` (Task 3).
- `internal/service_provider/service.go` — `GetReversionLiveService()` pulls sub-clients from the reversion client (Task 3).
- `docs/reversion/live-runner.md`, `env/prod.env.example` — document `REVERSION_TOKEN` (Task 4).

---

## Task 1: Authenticate the orders path

**Files:**
- Modify: `pkg/client/grpc/orders_service_client.go`
- Modify: `pkg/client/grpc/grpc.go:78`
- Test: `pkg/client/grpc/orders_auth_test.go` (create)

**Interfaces:**
- Consumes: `Auth` struct and `NewAuth(token string) *Auth` (`pkg/client/grpc/auth.go`); `NewRPCCredential(auth *Auth) grpc.CallOption` (same file); `investapi.OrdersServiceClient` interface with `PostOrder(ctx, *PostOrderRequest, ...grpc.CallOption) (*PostOrderResponse, error)`.
- Produces: `NewOrdersServiceClient(conn grpc.ClientConnInterface, token string) OrdersServiceClient` — same signature Task 3 relies on indirectly via `NewClientGrpc`. The `ordersServiceClient` struct gains field `auth *Auth`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/client/grpc/orders_auth_test.go`:

```go
package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

// fakeOrdersAPI implements investapi.OrdersServiceClient by embedding the
// interface (so only PostOrder is defined) and records the call options it
// receives, to prove the auth credential was attached.
type fakeOrdersAPI struct {
	investapi.OrdersServiceClient
	gotOpts []grpc.CallOption
}

func (f *fakeOrdersAPI) PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	f.gotOpts = opts
	return &investapi.PostOrderResponse{}, nil
}

func TestNewOrdersServiceClient_StoresToken(t *testing.T) {
	c := NewOrdersServiceClient(nil, "tok-xyz").(*ordersServiceClient)
	if c.auth == nil || c.auth.token != "tok-xyz" {
		t.Fatalf("auth = %+v, want token tok-xyz", c.auth)
	}
}

func TestOrdersServiceClient_PostOrderAttachesAuth(t *testing.T) {
	fake := &fakeOrdersAPI{}
	c := &ordersServiceClient{orderApi: fake, auth: NewAuth("tok-123")}

	if _, err := c.PostOrder(context.Background(), &investapi.PostOrderRequest{}); err != nil {
		t.Fatalf("PostOrder returned error: %v", err)
	}
	if len(fake.gotOpts) != 1 {
		t.Fatalf("PostOrder passed %d call options, want 1 (auth credential)", len(fake.gotOpts))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/client/grpc/ -run 'Orders' -v`
Expected: compile failure — `ordersServiceClient` has no field `auth`, and `NewOrdersServiceClient` takes 1 arg not 2.

- [ ] **Step 3: Implement token + auth in the orders client**

Edit `pkg/client/grpc/orders_service_client.go` — add the `auth` field, accept a `token`, and attach the credential in `PostOrder`:

```go
type ordersServiceClient struct {
	orderApi investapi.OrdersServiceClient
	auth     *Auth
}

func NewOrdersServiceClient(conn grpc.ClientConnInterface, token string) OrdersServiceClient {
	return &ordersServiceClient{
		orderApi: investapi.NewOrdersServiceClient(conn),
		auth:     NewAuth(token),
	}
}

func (c *ordersServiceClient) PostOrder(ctx context.Context, in *investapi.PostOrderRequest, opts ...grpc.CallOption) (*investapi.PostOrderResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts = append(opts, NewRPCCredential(c.auth))
	return c.orderApi.PostOrder(ctx, in, opts...)
}
```

- [ ] **Step 4: Update the single call site**

Edit `pkg/client/grpc/grpc.go:78`, passing the `token` already in scope in `NewClientGrpc`:

```go
		ordersServiceClient:      NewOrdersServiceClient(conn, token),
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/client/grpc/ -run 'Orders' -v`
Expected: PASS (both tests).

- [ ] **Step 6: Verify the whole package + build still compile**

Run: `go build ./... && go test ./pkg/client/grpc/`
Expected: no build errors; package tests PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/client/grpc/orders_service_client.go pkg/client/grpc/grpc.go pkg/client/grpc/orders_auth_test.go
git commit -m "fix(grpc): attach auth credential to PostOrder

NewOrdersServiceClient now takes a token and PostOrder attaches the Bearer
credential, matching the other sub-clients. Previously the orders path was
unauthenticated, so real order placement would fail under any token."
```

---

## Task 2: Add required REVERSION_TOKEN config field

**Files:**
- Modify: `internal/config/reversion.go`
- Test: `internal/config/reversion_test.go`

**Interfaces:**
- Consumes: existing `ReversionConfig` struct and `NewReversionConfig() *ReversionConfig`.
- Produces: `ReversionConfig.Token string` (env `REVERSION_TOKEN`, required) — Task 3 reads `cfg.Reversion.Token`.

- [ ] **Step 1: Write the failing test**

Append to `internal/config/reversion_test.go`:

```go
func TestNewReversionConfig_TokenHasNoDefault(t *testing.T) {
	c := NewReversionConfig()
	if c.Token != "" {
		t.Fatalf("default Token = %q, want empty (must come from REVERSION_TOKEN env)", c.Token)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TokenHasNoDefault -v`
Expected: compile failure — `c.Token` undefined.

- [ ] **Step 3: Add the field**

Edit `internal/config/reversion.go` — add `Token` as the first field (keep `AccountID` required-first grouping) with the required env tag; do not set a default in `NewReversionConfig()`:

```go
type ReversionConfig struct {
	AccountID     string   `config:"REVERSION_ACCOUNT_ID,required,backend=env"`
	Token         string   `config:"REVERSION_TOKEN,required,backend=env"`
	Tickers       []string `config:"REVERSION_TICKERS,backend=env"`
	BuyPct        float64  `config:"REVERSION_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"REVERSION_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"REVERSION_NOTIFY_ENABLED,backend=env"`
}
```

Leave `NewReversionConfig()` unchanged (it must NOT seed `Token`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS (all config tests including the new one).

- [ ] **Step 5: Commit**

```bash
git add internal/config/reversion.go internal/config/reversion_test.go
git commit -m "feat(config): add required REVERSION_TOKEN for reversion strategy"
```

---

## Task 3: Wire a dedicated reversion gRPC client

**Files:**
- Modify: `internal/service_provider/client.go`
- Modify: `internal/service_provider/service.go:229-244`

**Interfaces:**
- Consumes: `internalgrpc.NewClientGrpc(address, token string) (internalgrpc.GrpcClient, error)`; `s.appConfig.GrpcClient.AddressProd`; `s.appConfig.Reversion.Token` (from Task 2); `live.NewService(instruments, marketData, operations, orders, tg, cfg)` (existing signature).
- Produces: `GetReversionGrpcClient() (internalgrpc.GrpcClient, error)` on `*ServiceProvider`, cached in the `client` struct.

- [ ] **Step 1: Add the cache field and getter**

Edit `internal/service_provider/client.go` — add a `reversionGrpcClient` field to the `client` struct and a lazy getter alongside `GetGrpcClient`:

```go
type client struct {
	grpcClient          internalgrpc.GrpcClient
	reversionGrpcClient internalgrpc.GrpcClient
	telegramBot         telegram.Client
}
```

```go
// GetReversionGrpcClient returns a gRPC client authenticated with the reversion
// strategy's dedicated token (REVERSION_TOKEN), separate from the shared T_BANK
// client. It dials the same AddressProd; the second connection is negligible for
// an hourly cron strategy and keeps the reversion account fully isolated.
func (s *ServiceProvider) GetReversionGrpcClient() (internalgrpc.GrpcClient, error) {
	if serviceProvider.client.reversionGrpcClient == nil {
		var err error
		serviceProvider.client.reversionGrpcClient, err = internalgrpc.NewClientGrpc(
			s.appConfig.GrpcClient.AddressProd,
			s.appConfig.Reversion.Token,
		)
		if err != nil {
			return nil, err
		}
	}

	return serviceProvider.client.reversionGrpcClient, nil
}
```

- [ ] **Step 2: Point the reversion live service at the reversion client**

Edit `internal/service_provider/service.go` in `GetReversionLiveService()` — replace `GetGrpcClient()` with `GetReversionGrpcClient()`:

```go
func (*ServiceProvider) GetReversionLiveService() live.Service {
	if serviceProvider.service.reversionLiveService == nil {
		grpcClient, _ := serviceProvider.GetReversionGrpcClient()
		tgClient, _ := serviceProvider.GetTelegramBotClient()
		serviceProvider.service.reversionLiveService = live.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			grpcClient.OrdersServiceClient(),
			tgClient,
			serviceProvider.appConfig.Reversion,
		)
	}

	return serviceProvider.service.reversionLiveService
}
```

- [ ] **Step 3: Build to verify it compiles**

Run: `go build ./... && gofmt -l internal/service_provider/`
Expected: no build errors; `gofmt -l` prints nothing.

- [ ] **Step 4: Manual verification (paper mode)**

Set `REVERSION_TOKEN=<a valid read-capable token>`, `REVERSION_ACCOUNT_ID=<real account>`, `REVERSION_TRADE_ENABLED=false`, `APP_ENV=prod`, plus `T_BANK`/`TELEGRAM`. Run `go run ./cmd/main`.
Expected: app starts; the reversion buy/manage passes fetch portfolio/candles without `UNAUTHENTICATED` errors in the logs (read calls now run under `REVERSION_TOKEN`). No orders placed (paper mode).

> If you cannot supply a second token, verify at minimum that startup config load succeeds with `REVERSION_TOKEN` set and fails when it is unset (required).

- [ ] **Step 5: Commit**

```bash
git add internal/service_provider/client.go internal/service_provider/service.go
git commit -m "feat(reversion): trade via a dedicated REVERSION_TOKEN gRPC client"
```

---

## Task 4: Document REVERSION_TOKEN

**Files:**
- Modify: `docs/reversion/live-runner.md`
- Modify: `env/prod.env.example`

**Interfaces:**
- Consumes: nothing (docs only).
- Produces: nothing.

- [ ] **Step 1: Add REVERSION_TOKEN to the operator env table**

Edit `docs/reversion/live-runner.md` — add a row to the env-vars table (just below the `REVERSION_ACCOUNT_ID` row):

```markdown
| `REVERSION_TOKEN` | *обязательная* | API-токен Tinkoff Invest для стратегии reversion. Аутентифицирует **отдельный** gRPC-клиент reversion (не общий `T_BANK`). Секрет — задавайте в рантайме, не коммитьте. Нужен, когда счёт reversion под отдельным логином Tinkoff. |
```

- [ ] **Step 2: Add REVERSION_TOKEN to the prod env example**

Edit `env/prod.env.example` — add under the reversion block, above `REVERSION_ACCOUNT_ID`:

```bash
# Secret — inject at runtime, do NOT commit the value
REVERSION_TOKEN=
```

- [ ] **Step 3: Commit**

```bash
git add docs/reversion/live-runner.md env/prod.env.example
git commit -m "docs(reversion): document required REVERSION_TOKEN env var"
```

---

## Self-Review

**Spec coverage:**
- Config field `REVERSION_TOKEN` required → Task 2. ✓
- Dedicated reversion gRPC client + rewire live service → Task 3. ✓
- Orders auth fix (token + `PostOrder` credential + call site) → Task 1. ✓
- Tests: orders Bearer attach + config required-no-default → Tasks 1 & 2; manual paper-mode verify → Task 3. ✓
- Docs + prod.env.example → Task 4. ✓
- YAGNI items (no fallback, no conn reuse, no new endpoint) honored across tasks. ✓

**Placeholder scan:** No TBD/TODO; every code step shows full code; commands have expected output. ✓

**Type consistency:** `NewOrdersServiceClient(conn, token)`, `ordersServiceClient.auth *Auth`, `NewAuth`/`NewRPCCredential`, `GetReversionGrpcClient() (internalgrpc.GrpcClient, error)`, `ReversionConfig.Token`, `cfg.Reversion.Token` used consistently across Tasks 1–3. ✓
