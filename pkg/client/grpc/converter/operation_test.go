package converter

import (
	"testing"
	"time"

	investapi "tinvest/internal/pb/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertCursorItemsToCashOperations(t *testing.T) {
	ts1 := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)
	ts2 := time.Date(2024, 11, 1, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name  string
		items []*investapi.OperationItem
		want  []struct {
			date    time.Time
			typeStr string
			typeID  int32
			payment float64
		}
	}{
		{
			name:  "nil input returns empty slice",
			items: nil,
			want:  nil,
		},
		{
			name:  "empty input returns empty slice",
			items: []*investapi.OperationItem{},
			want:  nil,
		},
		{
			name: "single item with positive payment (deposit)",
			items: []*investapi.OperationItem{
				{
					Date:    timestamppb.New(ts1),
					Type:    investapi.OperationType_OPERATION_TYPE_INPUT,
					Payment: &investapi.MoneyValue{Units: 50000, Nano: 500000000},
				},
			},
			want: []struct {
				date    time.Time
				typeStr string
				typeID  int32
				payment float64
			}{
				{
					date:    ts1,
					typeStr: "OPERATION_TYPE_INPUT",
					typeID:  1,
					payment: 50000.5,
				},
			},
		},
		{
			name: "withdrawal item with negative payment",
			items: []*investapi.OperationItem{
				{
					Date:    timestamppb.New(ts2),
					Type:    investapi.OperationType_OPERATION_TYPE_OUTPUT,
					Payment: &investapi.MoneyValue{Units: -10000, Nano: -200000000},
				},
			},
			want: []struct {
				date    time.Time
				typeStr string
				typeID  int32
				payment float64
			}{
				{
					date:    ts2,
					typeStr: "OPERATION_TYPE_OUTPUT",
					typeID:  9,
					payment: -10000.2,
				},
			},
		},
		{
			name: "nil Payment yields zero",
			items: []*investapi.OperationItem{
				{
					Date:    timestamppb.New(ts1),
					Type:    investapi.OperationType_OPERATION_TYPE_INPUT_ACQUIRING,
					Payment: nil,
				},
			},
			want: []struct {
				date    time.Time
				typeStr string
				typeID  int32
				payment float64
			}{
				{
					date:    ts1,
					typeStr: "OPERATION_TYPE_INPUT_ACQUIRING",
					typeID:  54,
					payment: 0,
				},
			},
		},
		{
			name: "multiple items are all converted",
			items: []*investapi.OperationItem{
				{
					Date:    timestamppb.New(ts1),
					Type:    investapi.OperationType_OPERATION_TYPE_INPUT,
					Payment: &investapi.MoneyValue{Units: 100, Nano: 0},
				},
				{
					Date:    timestamppb.New(ts2),
					Type:    investapi.OperationType_OPERATION_TYPE_OUT_MULTI,
					Payment: &investapi.MoneyValue{Units: -200, Nano: 0},
				},
			},
			want: []struct {
				date    time.Time
				typeStr string
				typeID  int32
				payment float64
			}{
				{
					date:    ts1,
					typeStr: "OPERATION_TYPE_INPUT",
					typeID:  1,
					payment: 100,
				},
				{
					date:    ts2,
					typeStr: "OPERATION_TYPE_OUT_MULTI",
					typeID:  59,
					payment: -200,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertCursorItemsToCashOperations(tc.items)

			if tc.want == nil {
				if len(got) != 0 {
					t.Fatalf("expected empty slice, got %d items", len(got))
				}
				return
			}

			if len(got) != len(tc.want) {
				t.Fatalf("got %d items, want %d", len(got), len(tc.want))
			}

			for i, w := range tc.want {
				g := got[i]
				if !g.Date.Equal(w.date) {
					t.Errorf("item[%d].Date = %v, want %v", i, g.Date, w.date)
				}
				if g.Type != w.typeStr {
					t.Errorf("item[%d].Type = %q, want %q", i, g.Type, w.typeStr)
				}
				if g.TypeID != w.typeID {
					t.Errorf("item[%d].TypeID = %d, want %d", i, g.TypeID, w.typeID)
				}
				const tol = 1e-9
				diff := g.Payment - w.payment
				if diff < -tol || diff > tol {
					t.Errorf("item[%d].Payment = %v, want %v", i, g.Payment, w.payment)
				}
			}
		})
	}
}

func TestConvertCursorItemsToCashOperations_Yield(t *testing.T) {
	ts := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	items := []*investapi.OperationItem{
		{
			Date:    timestamppb.New(ts),
			Type:    investapi.OperationType_OPERATION_TYPE_SELL,
			Payment: &investapi.MoneyValue{Units: 1000, Nano: 0},
			Yield:   &investapi.MoneyValue{Units: 120, Nano: 500000000},
		},
		{
			Date:    timestamppb.New(ts),
			Type:    investapi.OperationType_OPERATION_TYPE_COUPON,
			Payment: &investapi.MoneyValue{Units: 50, Nano: 0},
			Yield:   nil, // нет Yield → 0
		},
	}

	got := ConvertCursorItemsToCashOperations(items)
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}

	const tol = 1e-9
	if d := got[0].Yield - 120.5; d < -tol || d > tol {
		t.Errorf("item[0].Yield = %v, want 120.5", got[0].Yield)
	}
	if d := got[1].Yield - 0; d < -tol || d > tol {
		t.Errorf("item[1].Yield = %v, want 0", got[1].Yield)
	}
}
