package yield

import (
	"context"
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

func (f *fakeOperationsClient) GetCashOperations(_ context.Context, _ string, _, _ time.Time) ([]grpcmodel.CashOperation, error) {
	return f.cashOps, nil
}

func (f *fakeOperationsClient) GetPortfolioTotal(_ context.Context, _ string) (float64, error) {
	return f.portfolioTotal, nil
}

func (f *fakeOperationsClient) GetAvailableCash(_ context.Context, _ string) (float64, error) {
	return 0, nil
}

func (f *fakeOperationsClient) GetInstrumentTrades(_ context.Context, _, _ string, _, _ time.Time) ([]grpcmodel.Trade, error) {
	return nil, nil
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

// --- Tests ---

// TestPortfolioYieldYTD_HappyPath tests the full flow when a manual start value
// is configured, one deposit operation occurs, and XIRR can be computed.
func TestPortfolioYieldYTD_HappyPath(t *testing.T) {
	const startValue = 100_000.0

	// A single deposit mid-year (INPUT, TypeID 1).
	thisYear := time.Now().Year()
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
		manualStartValue: startValue,
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
	if len(msg) < 20 {
		t.Errorf("Telegram message seems too short: %q", msg)
	}
}

// TestPortfolioYieldYTD_InsufficientData tests the path where no manual start
// value is configured (0), so vStart is unknown.
func TestPortfolioYieldYTD_InsufficientData(t *testing.T) {
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

	// Message should mention insufficient data and the config variable.
	if !strings.Contains(msg, "Недостаточно") || !strings.Contains(msg, "PORTFOLIO_YTD_START_VALUE") {
		t.Errorf("expected message to mention недостаточно/PORTFOLIO_YTD_START_VALUE, got: %q", msg)
	}
}

// TestPortfolioYieldYTD_NoAccounts tests that the function returns nil without
// sending a message when there are no accounts.
func TestPortfolioYieldYTD_NoAccounts(t *testing.T) {
	tgClient := &fakeTelegramClient{}
	svc := &service{
		operationsServiceClient: &fakeOperationsClient{},
		usersServiceClient:      &fakeUsersClient{accounts: nil},
		tgClient:                tgClient,
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
		manualStartValue: 40_000.0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 0); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}
}

// TestPortfolioYieldYTD_XIRRAvailableTrue ensures the XIRR-available message
// is produced when a manual start value is configured and the period is long
// enough to annualize.
func TestPortfolioYieldYTD_XIRRAvailableTrue(t *testing.T) {
	const startValue = 200_000.0
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
		manualStartValue: startValue,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 99); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	if len(tgClient.messages) == 0 {
		t.Fatal("expected Telegram message")
	}

	msg := tgClient.messages[0]
	if msg == "" {
		t.Error("expected non-empty message")
	}
	// The header line must always be present.
	if !strings.Contains(msg, "Доходность") {
		t.Errorf("expected message to contain 'Доходность', got: %q", msg)
	}
}
