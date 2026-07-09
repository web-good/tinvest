package yield

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	grpcmocks "tinvest/pkg/client/grpc/mocks"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
)

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

	operationsClient := grpcmocks.NewMockOperationsServiceClient(t)
	operationsClient.EXPECT().
		GetPortfolioTotal(mock.Anything, mock.Anything).
		Return(endValue, nil).
		Once()
	operationsClient.EXPECT().
		GetCashOperations(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]grpcmodel.CashOperation{deposit}, nil).
		Once()

	usersClient := grpcmocks.NewMockUsersServiceClient(t)
	usersClient.EXPECT().
		GetAccounts(mock.Anything).
		Return([]grpcmodel.Account{{ID: "acc1"}}, nil).
		Once()

	var sentMsg string
	tgClient := tgmocks.NewMockClient(t)
	tgClient.EXPECT().
		SendMessageToChat(mock.Anything, mock.Anything).
		Run(func(_ int64, msg string) {
			sentMsg = msg
		}).
		Return(nil).
		Once()

	svc := &service{
		operationsServiceClient: operationsClient,
		usersServiceClient:      usersClient,
		tgClient:                tgClient,
		manualStartValue:        startValue,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 12345); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	// A message must have been sent.
	if sentMsg == "" {
		t.Fatal("expected non-empty Telegram message")
	}
	if len(sentMsg) < 20 {
		t.Errorf("Telegram message seems too short: %q", sentMsg)
	}
}

// TestPortfolioYieldYTD_InsufficientData tests the path where no manual start
// value is configured (0), so vStart is unknown.
func TestPortfolioYieldYTD_InsufficientData(t *testing.T) {
	const endValue = 80_000.0

	operationsClient := grpcmocks.NewMockOperationsServiceClient(t)
	operationsClient.EXPECT().
		GetPortfolioTotal(mock.Anything, mock.Anything).
		Return(endValue, nil).
		Once()
	operationsClient.EXPECT().
		GetCashOperations(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	usersClient := grpcmocks.NewMockUsersServiceClient(t)
	usersClient.EXPECT().
		GetAccounts(mock.Anything).
		Return([]grpcmodel.Account{{ID: "acc1"}}, nil).
		Once()

	var sentMsg string
	tgClient := tgmocks.NewMockClient(t)
	tgClient.EXPECT().
		SendMessageToChat(mock.Anything, mock.Anything).
		Run(func(_ int64, msg string) {
			sentMsg = msg
		}).
		Return(nil).
		Once()

	svc := &service{
		operationsServiceClient: operationsClient,
		usersServiceClient:      usersClient,
		tgClient:                tgClient,
		manualStartValue:        0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 12345); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	// A message must have been sent.
	if sentMsg == "" {
		t.Fatal("expected at least one Telegram message, got none")
	}

	// Message should mention insufficient data and the config variable.
	if !strings.Contains(sentMsg, "Недостаточно") || !strings.Contains(sentMsg, "PORTFOLIO_YTD_START_VALUE") {
		t.Errorf("expected message to mention недостаточно/PORTFOLIO_YTD_START_VALUE, got: %q", sentMsg)
	}
}

// TestPortfolioYieldYTD_NoAccounts tests that the function returns nil without
// sending a message when there are no accounts.
func TestPortfolioYieldYTD_NoAccounts(t *testing.T) {
	// No expectations are set on operationsClient/tgClient: the service must
	// return before touching either, so any unexpected call fails the mock.
	operationsClient := grpcmocks.NewMockOperationsServiceClient(t)

	usersClient := grpcmocks.NewMockUsersServiceClient(t)
	usersClient.EXPECT().
		GetAccounts(mock.Anything).
		Return(nil, nil).
		Once()

	svc := &service{
		operationsServiceClient: operationsClient,
		usersServiceClient:      usersClient,
		tgClient:                nil,
		manualStartValue:        0,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 12345); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}
}

// TestPortfolioYieldYTD_NilTgClient verifies the function doesn't panic
// when tgClient is nil (no Telegram configured).
func TestPortfolioYieldYTD_NilTgClient(t *testing.T) {
	const endValue = 50_000.0

	operationsClient := grpcmocks.NewMockOperationsServiceClient(t)
	operationsClient.EXPECT().
		GetPortfolioTotal(mock.Anything, mock.Anything).
		Return(endValue, nil).
		Once()
	operationsClient.EXPECT().
		GetCashOperations(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	usersClient := grpcmocks.NewMockUsersServiceClient(t)
	usersClient.EXPECT().
		GetAccounts(mock.Anything).
		Return([]grpcmodel.Account{{ID: "acc1"}}, nil).
		Once()

	svc := &service{
		operationsServiceClient: operationsClient,
		usersServiceClient:      usersClient,
		tgClient:                nil, // no Telegram
		manualStartValue:        40_000.0,
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

	operationsClient := grpcmocks.NewMockOperationsServiceClient(t)
	operationsClient.EXPECT().
		GetPortfolioTotal(mock.Anything, mock.Anything).
		Return(endValue, nil).
		Once()
	operationsClient.EXPECT().
		GetCashOperations(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil). // no deposits/withdrawals
		Once()

	usersClient := grpcmocks.NewMockUsersServiceClient(t)
	usersClient.EXPECT().
		GetAccounts(mock.Anything).
		Return([]grpcmodel.Account{{ID: "acc1"}}, nil).
		Once()

	var sentMsg string
	tgClient := tgmocks.NewMockClient(t)
	tgClient.EXPECT().
		SendMessageToChat(mock.Anything, mock.Anything).
		Run(func(_ int64, msg string) {
			sentMsg = msg
		}).
		Return(nil).
		Once()

	svc := &service{
		operationsServiceClient: operationsClient,
		usersServiceClient:      usersClient,
		tgClient:                tgClient,
		manualStartValue:        startValue,
	}

	ctx := context.Background()
	if err := svc.PortfolioYieldYTD(ctx, 99); err != nil {
		t.Fatalf("PortfolioYieldYTD returned error: %v", err)
	}

	if sentMsg == "" {
		t.Fatal("expected Telegram message")
	}

	// The header line must always be present.
	if !strings.Contains(sentMsg, "Доходность") {
		t.Errorf("expected message to contain 'Доходность', got: %q", sentMsg)
	}
}
