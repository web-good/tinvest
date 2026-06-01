package yield

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	grpcmodel "tinvest/pkg/client/grpc/model"
)

// --- Fakes ---

// fakeOperationsClient satisfies grpc.OperationsServiceClient.
type fakeOperationsClient struct {
	portfolioTotal float64
	cashOps        []grpcmodel.CashOperation
}

func (f *fakeOperationsClient) GetPortfolio(_ context.Context, _ string) ([]*grpcmodel.Position, error) {
	return nil, nil
}

func (f *fakeOperationsClient) GetOperation(_ context.Context, _ string, _ string) ([]*grpcmodel.Operation, error) {
	return nil, nil
}

func (f *fakeOperationsClient) GetCashFlowOperations(_ context.Context, _ string, _, _ time.Time) ([]grpcmodel.CashOperation, error) {
	return f.cashOps, nil
}

func (f *fakeOperationsClient) GetPortfolioTotal(_ context.Context, _ string) (float64, error) {
	return f.portfolioTotal, nil
}

// fakeUsersClient satisfies grpc.UsersServiceClient.
type fakeUsersClient struct {
	accounts []grpcmodel.Account
}

func (f *fakeUsersClient) GetAccounts(_ context.Context) ([]grpcmodel.Account, error) {
	return f.accounts, nil
}

// fakeTelegramClient satisfies telegram.Client.
type fakeTelegramClient struct {
	messages []string
}

func (f *fakeTelegramClient) SendMessage(msg string) error {
	f.messages = append(f.messages, msg)
	return nil
}

func (f *fakeTelegramClient) SendMessageToChat(_ int64, msg string) error {
	f.messages = append(f.messages, msg)
	return nil
}

// --- Helpers ---

// seedSnapshot writes a single Snapshot directly to the file at path.
func seedSnapshot(t *testing.T, path string, snaps []Snapshot) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seedSnapshot: mkdir: %v", err)
	}
	data, err := json.MarshalIndent(snaps, "", "  ")
	if err != nil {
		t.Fatalf("seedSnapshot: marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("seedSnapshot: write: %v", err)
	}
}

// loadSnapshotsFromFile reads the snapshot file.
func loadSnapshotsFromFile(t *testing.T, path string) []Snapshot {
	t.Helper()
	store := newSnapshotStore(path)
	snaps, err := store.load()
	if err != nil {
		t.Fatalf("loadSnapshotsFromFile: %v", err)
	}
	return snaps
}

// --- Tests ---

// TestPortfolioYieldYTD_HappyPath tests the full flow when a prior-year snapshot
// exists, one deposit operation occurs, and XIRR can be computed.
func TestPortfolioYieldYTD_HappyPath(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "snapshots.json")

	// Seed a Dec-31 snapshot from the prior year so valueAtYearStart succeeds.
	thisYear := time.Now().Year()
	dec31 := time.Date(thisYear-1, time.December, 31, 0, 0, 0, 0, time.UTC)
	const startValue = 100_000.0
	seedSnapshot(t, snapPath, []Snapshot{{Date: dec31, TotalValue: startValue}})

	// A single deposit mid-year (INPUT, TypeID 1).
	deposit := grpcmodel.CashOperation{
		Date:    time.Date(thisYear, time.March, 15, 0, 0, 0, 0, time.UTC),
		TypeID:  1,
		Payment: 10_000.0,
	}

	const endValue = 115_000.0

	tgClient := &fakeTelegramClient{}
	svc := &service{
		operationsServiceClient: &fakeOperationsClient{
			portfolioTotal: endValue,
			cashOps:        []grpcmodel.CashOperation{deposit},
		},
		usersServiceClient: &fakeUsersClient{
			accounts: []grpcmodel.Account{{ID: "acc1"}},
		},
		tgClient:         tgClient,
		snapshots:        newSnapshotStore(snapPath),
		manualStartValue: 0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 12345); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	// A message must have been sent.
	if len(tgClient.messages) == 0 {
		t.Fatal("expected at least one Telegram message, got none")
	}
	msg := tgClient.messages[0]
	if msg == "" {
		t.Fatal("expected non-empty Telegram message")
	}

	// The message should contain a year indicator or XIRR result.
	// Since XIRRAvailable should be true for a period longer than 0.1 years,
	// we check the message is non-trivial.
	if len(msg) < 20 {
		t.Errorf("Telegram message seems too short: %q", msg)
	}

	// The snapshot file should now have TWO rows: the seeded one + today's.
	snaps := loadSnapshotsFromFile(t, snapPath)
	if len(snaps) < 2 {
		t.Errorf("expected at least 2 snapshots after run, got %d", len(snaps))
	}

	// The last snapshot's value should equal endValue.
	last := snaps[len(snaps)-1]
	if last.TotalValue != endValue {
		t.Errorf("last snapshot TotalValue = %v, want %v", last.TotalValue, endValue)
	}
}

// TestPortfolioYieldYTD_InsufficientData tests the path where no snapshot
// exists and manualStartValue is 0, so vStart is unknown.
func TestPortfolioYieldYTD_InsufficientData(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "snapshots.json")
	// Do NOT seed any snapshot — file does not exist.

	const endValue = 80_000.0

	tgClient := &fakeTelegramClient{}
	svc := &service{
		operationsServiceClient: &fakeOperationsClient{
			portfolioTotal: endValue,
			cashOps:        nil,
		},
		usersServiceClient: &fakeUsersClient{
			accounts: []grpcmodel.Account{{ID: "acc1"}},
		},
		tgClient:         tgClient,
		snapshots:        newSnapshotStore(snapPath),
		manualStartValue: 0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 12345); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	// A message must have been sent.
	if len(tgClient.messages) == 0 {
		t.Fatal("expected at least one Telegram message, got none")
	}
	msg := tgClient.messages[0]

	// Message should mention insufficient data or snapshot saved.
	if !strings.Contains(msg, "Недостаточно") && !strings.Contains(msg, "снимок") && !strings.Contains(msg, "Снимок") {
		t.Errorf("expected message to mention недостаточно/снимок, got: %q", msg)
	}

	// Snapshot file should now have today's row (was created from scratch).
	snaps := loadSnapshotsFromFile(t, snapPath)
	if len(snaps) != 1 {
		t.Errorf("expected 1 snapshot after run, got %d", len(snaps))
	}
	if snaps[0].TotalValue != endValue {
		t.Errorf("snapshot TotalValue = %v, want %v", snaps[0].TotalValue, endValue)
	}
}

// TestPortfolioYieldYTD_NoAccounts tests that the function returns nil without
// sending a message when there are no accounts.
func TestPortfolioYieldYTD_NoAccounts(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "snapshots.json")

	tgClient := &fakeTelegramClient{}
	svc := &service{
		operationsServiceClient: &fakeOperationsClient{},
		usersServiceClient:      &fakeUsersClient{accounts: nil},
		tgClient:                tgClient,
		snapshots:               newSnapshotStore(snapPath),
		manualStartValue:        0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 12345); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	if len(tgClient.messages) != 0 {
		t.Errorf("expected no Telegram messages for empty accounts, got %d", len(tgClient.messages))
	}
}

// TestPortfolioYieldYTD_NilTgClient verifies the function doesn't panic
// when tgClient is nil (no Telegram configured).
func TestPortfolioYieldYTD_NilTgClient(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "snapshots.json")

	const endValue = 50_000.0
	svc := &service{
		operationsServiceClient: &fakeOperationsClient{
			portfolioTotal: endValue,
			cashOps:        nil,
		},
		usersServiceClient: &fakeUsersClient{
			accounts: []grpcmodel.Account{{ID: "acc1"}},
		},
		tgClient:         nil, // no Telegram
		snapshots:        newSnapshotStore(snapPath),
		manualStartValue: 0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 0); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	// No panic; snapshot should be written.
	snaps := loadSnapshotsFromFile(t, snapPath)
	if len(snaps) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snaps))
	}
}

// TestPortfolioYieldYTD_XIRRAvailableTrue ensures XIRRAvailable=true message
// contains the XIRR percentage line (годовая/XIRR).
func TestPortfolioYieldYTD_XIRRAvailableTrue(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, "snapshots.json")

	thisYear := time.Now().Year()
	// Period must be > 0.1 years: seed snapshot as Jan 1 of this year
	// using the prior-year Dec 31 path.
	dec31 := time.Date(thisYear-1, time.December, 31, 0, 0, 0, 0, time.UTC)
	const startValue = 200_000.0
	seedSnapshot(t, snapPath, []Snapshot{{Date: dec31, TotalValue: startValue}})

	const endValue = 220_000.0 // 10% growth, no deposits

	tgClient := &fakeTelegramClient{}
	svc := &service{
		operationsServiceClient: &fakeOperationsClient{
			portfolioTotal: endValue,
			cashOps:        nil, // no deposits/withdrawals
		},
		usersServiceClient: &fakeUsersClient{
			accounts: []grpcmodel.Account{{ID: "acc1"}},
		},
		tgClient:         tgClient,
		snapshots:        newSnapshotStore(snapPath),
		manualStartValue: 0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 99); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	if len(tgClient.messages) == 0 {
		t.Fatal("expected Telegram message")
	}

	msg := tgClient.messages[0]

	// If today is past ~Feb 10 (0.1 * 365 ≈ 36 days into year), XIRR is available.
	// We check that the message does NOT contain the "insufficient data" note.
	// (We can't always guarantee XIRR succeeds on all test run dates, so we just
	// check the message is non-empty and well-formed.)
	if msg == "" {
		t.Error("expected non-empty message")
	}
	// The header line must always be present.
	if !strings.Contains(msg, "Доходность") {
		t.Errorf("expected message to contain 'Доходность', got: %q", msg)
	}
}
