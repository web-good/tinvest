package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	investapi "tinvest/internal/pb/v1"
	"tinvest/pkg/client/grpc/converter"
	"tinvest/pkg/client/grpc/model"
)

type OperationsServiceClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]*model.Position, error)
	GetOperation(ctx context.Context, accountID string, figi string) ([]*model.Operation, error)
	GetCashOperations(ctx context.Context, accountID string, from, to time.Time) ([]model.CashOperation, error)
	GetPortfolioTotal(ctx context.Context, accountID string) (float64, error)
}

type operationsServiceClient struct {
	operationApi investapi.OperationsServiceClient
	auth         *Auth
}

func NewOperationsServiceClient(conn grpc.ClientConnInterface, token string) OperationsServiceClient {
	return &operationsServiceClient{
		operationApi: investapi.NewOperationsServiceClient(conn),
		auth:         NewAuth(token),
	}
}

func (o *operationsServiceClient) GetPortfolio(ctx context.Context, accountID string) ([]*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var cur investapi.PortfolioRequest_CurrencyRequest
	cur = investapi.PortfolioRequest_RUB
	resp, err := o.operationApi.GetPortfolio(ctx, &investapi.PortfolioRequest{
		AccountId: accountID,
		Currency:  &cur,
	}, NewRPCCredential(o.auth))

	if err != nil {
		return nil, err
	}

	return converter.ConvertPortfolioFromBp(resp), nil
}

func (o *operationsServiceClient) GetOperation(ctx context.Context, accountID string, figi string) ([]*model.Operation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := o.operationApi.GetOperations(ctx, &investapi.OperationsRequest{
		AccountId: accountID,
		Figi:      &figi,
	}, NewRPCCredential(o.auth))

	if err != nil {
		return nil, err
	}

	return converter.ConvertOperationFromBp(resp.Operations), nil
}

// relevantOperationTypes lists the operation types fetched for portfolio-yield
// calculation: cash deposits/withdrawals (for XIRR) plus income operations
// (coupons, dividends, sales) and their taxes (for the income breakdown).
var relevantOperationTypes = []investapi.OperationType{
	// Deposits
	investapi.OperationType_OPERATION_TYPE_INPUT,
	investapi.OperationType_OPERATION_TYPE_INPUT_SWIFT,
	investapi.OperationType_OPERATION_TYPE_INPUT_ACQUIRING,
	investapi.OperationType_OPERATION_TYPE_INP_MULTI,
	// Withdrawals
	investapi.OperationType_OPERATION_TYPE_OUTPUT,
	investapi.OperationType_OPERATION_TYPE_OUTPUT_SWIFT,
	investapi.OperationType_OPERATION_TYPE_OUTPUT_ACQUIRING,
	investapi.OperationType_OPERATION_TYPE_OUT_MULTI,
	// Coupons + tax
	investapi.OperationType_OPERATION_TYPE_COUPON,
	investapi.OperationType_OPERATION_TYPE_BOND_TAX,
	investapi.OperationType_OPERATION_TYPE_BOND_TAX_PROGRESSIVE,
	// Dividends + tax
	investapi.OperationType_OPERATION_TYPE_DIVIDEND,
	investapi.OperationType_OPERATION_TYPE_DIV_EXT,
	investapi.OperationType_OPERATION_TYPE_DIVIDEND_TAX,
	investapi.OperationType_OPERATION_TYPE_DIVIDEND_TAX_PROGRESSIVE,
	// Sales (realized profit via Yield)
	investapi.OperationType_OPERATION_TYPE_SELL,
	investapi.OperationType_OPERATION_TYPE_SELL_CARD,
	investapi.OperationType_OPERATION_TYPE_SELL_MARGIN,
}

// GetCashOperations fetches all executed operations relevant to portfolio-yield
// computation (deposits/withdrawals, coupons, dividends, sales and their taxes)
// for the given account and time range using cursor-based pagination.
func (o *operationsServiceClient) GetCashOperations(ctx context.Context, accountID string, from, to time.Time) ([]model.CashOperation, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	state := investapi.OperationState_OPERATION_STATE_EXECUTED
	limit := int32(1000)
	fromPb := timestamppb.New(from)
	toPb := timestamppb.New(to)

	var result []model.CashOperation
	var cursor string

	for {
		req := &investapi.GetOperationsByCursorRequest{
			AccountId:      accountID,
			From:           fromPb,
			To:             toPb,
			State:          &state,
			Limit:          &limit,
			OperationTypes: relevantOperationTypes,
		}
		if cursor != "" {
			req.Cursor = &cursor
		}

		resp, err := o.operationApi.GetOperationsByCursor(ctx, req, NewRPCCredential(o.auth))
		if err != nil {
			return nil, err
		}

		result = append(result, converter.ConvertCursorItemsToCashOperations(resp.GetItems())...)

		if !resp.GetHasNext() {
			break
		}
		next := resp.GetNextCursor()
		if next == "" {
			break
		}
		cursor = next
	}

	return result, nil
}

// GetPortfolioTotal returns the total RUB portfolio value from the Tinkoff API.
func (o *operationsServiceClient) GetPortfolioTotal(ctx context.Context, accountID string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cur := investapi.PortfolioRequest_RUB
	resp, err := o.operationApi.GetPortfolio(ctx, &investapi.PortfolioRequest{
		AccountId: accountID,
		Currency:  &cur,
	}, NewRPCCredential(o.auth))
	if err != nil {
		return 0, err
	}

	total := resp.GetTotalAmountPortfolio()
	if total == nil {
		return 0, nil
	}
	return float64(total.GetUnits()) + float64(total.GetNano())/1e9, nil
}
