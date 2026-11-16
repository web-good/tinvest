# Density shared account implementation plan

> **For agentic workers:** Use superpowers:executing-plans inline; final independent review via requesting-code-review.

**Goal:** Connect continuous density execution to the existing live account's ownership and entry coordination.
**Architecture:** Durable account coordinator shared by the candle runner and density engine; existing candle strategy interface unchanged. DI inherits RSI account/token. App starts the continuous participant.
**Tech stack:** Go 1.25, existing gRPC/protobuf, flock + synced JSON ownership records.
**Spec:** docs/superpowers/specs/2026-11-16-density-shared-account-design.md

## Constraints and review focus

- No real orders or app launch; preserve existing dirty worktrees and previous density implementation.
- One ticker owner includes pending and uncertain orders; fail closed on missing/corrupt ownership.
- Holding the candle-pass entry lock must not block density cancellation/exit.
- A closed ownership marker must prevent adoption from an older candle portfolio snapshot.
- Persist closed-episode accounting before releasing ownership; recovery must finish release without reopening a trade.
- Fixed one-lot allocation, inherited RSI credentials, all existing guests retain behavior.

## Tasks

1. [x] `livecore/accountcoord`: account flock, synced account ownership registry, legacy-state query, idempotent reserve/closed/remove operations; contention/corruption/restart tests RED→GREEN.
2. [x] Density engine: attach coordinator, reserve before BUY, check only candidate instrument, persist pending release, recover ownership, reject uncoordinated shared state; tests RED→GREEN.
3. [x] Candle runner: take common pass lock, skip density owners before reconstruction, retain snapshot exclusion after density close, reserve live BUY before submission and hand off to legacy state after Save; regression tests RED→GREEN.
4. [x] Config, DI, continuous App worker and disabled-standalone guard. Inherit account/token/client; startup failure isolated; no live activation or env changes. Tests RED→GREEN and operator docs.
5. [x] Relevant race suites, vet, build, lint, independent review and verified important fixes.

## Ledger

User asked “надо подключить” after selecting the shared-account integration; continue that authorized implementation without another account-choice or implementation permission round.
Ruling: preserve a continuous participant instead of a candle adapter — order-book safety conditions need an independent stream and executor.
Ruling: use persistent separate ownership records alongside the existing state format — avoid inventing ATR entries and preserve existing strategies' state.
Ruling: postpone the merge into the main checkout as the user requested; implement and verify in the existing density worktree.


Task 1 complete: account entry flock, synced owners.json and independent initialized marker, legacy-state checks, closed-claim CAS, cross-instance density lifetime lease and fixed journal binding. Unit/race tests pass.
Task 2 complete: ownership recorded after journal intent and before BUY; candidate-only conflict skipping; pending release replay; orphan/root mismatch rejection. Cancellation/exit under a held candle-pass entry lease passes.
Task 3 complete: immutable pre-portfolio exclusions include closed claims; pending legacy reservations survive uncertainty and state Save failure (including visible-but-failed writes). Durable FileStore Save syncs content and directory chain before handoff. Pending owners outside configured entry universe are visited.
Task 4 complete: inherits RSI account/token/client/coordinator, continuous App participant, fixed one lot and one density episode, standalone trade retired. Optional density-live configuration is loaded separately so parse/validation errors cannot abort other live workers. Before any candle entry lease a synchronous read-only guard verifies known journal/shared-root binding, including cached original journal location.
Task 5 complete: affected race suites pass including all density/paper, livecore, RSI and reversion packages, provider/app/config/grpc and both CLI commands; go vet and full supported build pass; relevant golangci-lint reports 0 issues; git diff --check clean. No real orders, order cancellations, main App launch or live-environment changes performed.

Independent review: first reviewer reported three Important issues before a service usage limit interrupted it. Remaining lifecycle/startup coverage was completed by density_shared_design_check, which reported two further Important issues and no others. All five were fixed through observed RED→GREEN regressions; no re-review of already verified fixes.
Final: fixed busy-first candidate starvation — TestBrokerBusyFirstCandidateStillTriesFreeNeighbor.
Final: fixed durable legacy handoff — TestSaveSyncFailureDoesNotPublishState plus TestVisibleButFailedSaveStillKeepsLegacyClaim.
Final: fixed multi-container density singleton/journal writer race — TestDensityLifetimeLeaseIsSharedAcrossInstancesAndBindsJournal and TestDensityReservationsAreAccountWide (guard-removal mutation also observed FAIL, then restored PASS).
Final: fixed optional config aborting whole App — TestMalformedDensityLiveDoesNotBreakMainConfigLoad and TestMalformedDensityLiveSettingsRemainLocal.
Final: fixed changed-root startup takeover — TestCandleGuardRejectsDifferentCoordinatorBeforeDecisions; guard executes synchronously before either account entry lease, while ordinary market/config failures remain local to density.
Final: no deferred minor findings.

Ruling: one locked, atomically written registry with a separate initialized marker rather than independent per-ticker JSON files — CAS closes cannot overwrite newer owners and deleted registry cannot look like an empty account.
Ruling: runtime density singleton and journal-directory binding live on the common state volume — per-user cache locks alone cannot exclude separate containers.
Ruling: invalid optional density settings are isolated, but a known ownership/root integrity failure blocks candle decisions — continuing from a fresh empty registry could adopt the density holding.
Ruling: disable standalone real-trading CLI; keep it read-only without opening any state journal — the shared application is the sole supported coordinated execution path.

Compatibility evidence: disposable three-way overlay of these changes onto main HEAD 02cdf486 in /tmp/tinvest-density-main-compat-6sfy5o43 merges without conflicts. Its newer FVG/core/shared tests pass race and its supported full build passes. The actual main checkout and its unrelated dirty backtest work remain untouched. Re-run after any later changes to main before the eventual user-requested merge.
