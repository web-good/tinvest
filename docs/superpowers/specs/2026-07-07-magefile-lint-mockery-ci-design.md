# Magefile + golangci-lint + mockery + CI — Design Spec

**Date:** 2026-07-07
**Status:** Approved (design), pending implementation plan

## Goal

Introduce a `magefiles/` build package as the single entry point for developer
and CI tooling, wire in `golangci-lint` (lint) and `mockery` (mock generation +
test migration), and run all of these steps in CI as a gate before build/deploy.

## Decisions (locked)

- **Mage package layout:** dedicated `magefiles/` directory (idiomatic mage main
  package; clean build tags; its own deps don't leak into the main module).
  Rejected: root `magefile.go` with `//go:build mage` (worse isolation).
- **Tool installation:** pinned versions of `golangci-lint` and `mockery`
  installed into `./bin` via a `mage tools` target — matches the existing
  `make install-deps` pattern (`GOBIN=./bin go install …@version`). Mage itself
  is still bootstrapped by the existing `make install-deps`.
  Rejected: go.mod `tool` directives (breaks uniformity with current `bin/`).
- **Tool versions:** `golangci-lint` v2 (config `version: "2"`), `mockery` v2
  (stable `packages:` config). Exact patch versions pinned during implementation.
- **golangci-lint scope:** curated starter linter set applied to the **whole**
  repo; fix all findings until CI is green.
- **mockery scope:** generate mocks **and migrate** the existing hand-written
  fakes to mockery mocks, one package at a time, keeping tests green throughout.
- **CI:** a `checks` job runs on `pull_request` + `push` (all branches); build
  and deploy are gated on it (`needs: checks`, deploy only on push to `main`).

## Components

### 1. `magefiles/` package

A single mage main package with these targets:

| Target       | Behavior |
|--------------|----------|
| `Tools`      | install `golangci-lint` + `mockery` into `./bin` at pinned versions |
| `Lint`       | `./bin/golangci-lint run ./...` |
| `Test`       | `go test -race ./...` |
| `Mocks`      | `./bin/mockery` — generate mocks per `.mockery.yaml` |
| `MocksCheck` | run `Mocks`, then `git diff --exit-code` — fail if generated mocks are stale (CI drift guard) |
| `CI`         | run `Lint`, `Test`, `MocksCheck` in sequence — the single CI entry point |

Mage is the single source of truth for these commands, used identically locally
and in CI.

### 2. golangci-lint (`.golangci.yml`, v2 schema)

- **Enabled linters (starter set):** `govet`, `errcheck`, `staticcheck`,
  `ineffassign`, `unused`, `gosimple`, `gofmt`, `goimports`, `misspell`,
  `unconvert`, `bodyclose`, `revive` (lenient ruleset).
- **Excluded paths (generated code):** `internal/pb/`, goverter-generated
  converters, the generated mocks directory, and `magefiles/` if needed.
- **Policy:** lint the whole repo; fix findings until green. Expect a batch of
  fixes across existing code — done package by package, `mage lint` green before
  moving to the next.

### 3. mockery (`.mockery.yaml`, v2) + migration

- Config in `packages:` mode, `with-expecter: true`, mocks written to a `mocks/`
  tree (mirror layout; per-package placement confirmed during planning based on
  actual usage).
- **Targets only the interfaces that currently have hand-written fakes** (~10
  test files), not all 31 interfaces in the repo — YAGNI.
- **Migration is package-by-package:**
  1. Generate the mock for the interface.
  2. Rewrite the test to use the mockery mock instead of the hand-written fake.
  3. `go test ./<pkg>/...` must be green.
  4. Only then delete the hand-written fake.
  No red tests at any step.

### 4. CI — modify `.github/workflows/main.yaml`

Single workflow, three jobs:

- **Triggers:** add `pull_request` + `push` on all branches (currently only push
  to `main`/`master`).
- `checks` — runs on every trigger: `actions/setup-go` (with cache) →
  `mage tools` → `mage ci` (lint + test + mocks-drift).
- `image-build-and-push` — `needs: checks`, `if:` push to `main` only.
- `deploy-image` — `needs: image-build-and-push` (unchanged behavior).

Result: checks run on every PR/push; **red checks → no build, no deploy**; deploy
still only from `main`.

## Testing

- `mage test` (`go test -race ./...`) must stay green throughout, especially
  during the fake→mock migration (per-package gate).
- `mage lint` must reach green on the whole repo.
- `mage mocksCheck` must be green (no drift) — enforced in CI.
- CI dry-run: verify the `checks` job fails the pipeline when lint/test/mock-drift
  fails, and that build/deploy do not run on failure.

## Out of scope

- Wrapping the existing protoc/goverter `generate` targets into mage (kept in the
  Makefile).
- Enabling stricter linters beyond the starter set (future iteration).
- Generating mocks for interfaces that have no test double today.
