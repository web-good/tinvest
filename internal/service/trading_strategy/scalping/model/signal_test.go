package model

import "testing"

func TestIsStopReason(t *testing.T) {
	tests := []struct {
		reason string
		want   bool
	}{
		{reason: "SL", want: true},
		{reason: "SL_CLOSE", want: true},
		{reason: "SL_EMERGENCY", want: true},
		{reason: "CATSL", want: true},
		{reason: "TRAIL", want: true},
		{reason: "ATRSL", want: true},
		{reason: "TP", want: false},
		{reason: "OB", want: false},
		{reason: "RSI50", want: false},
		{reason: "MACD", want: false},
		{reason: "RSI", want: false},
		{reason: "", want: false},
	}
	for _, tt := range tests {
		name := tt.reason
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			if got := IsStopReason(tt.reason); got != tt.want {
				t.Errorf("IsStopReason(%q) = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}

func TestIsCloseConfirmedStopReason(t *testing.T) {
	for _, reason := range []string{"SL_CLOSE", "SL_EMERGENCY", "SL", "TRAIL", "ATRSL", "CATSL", "TP", "OB", "RSI50", "", "sl_close", "SL_CLOSE "} {
		t.Run(reason, func(t *testing.T) {
			want := reason == "SL_CLOSE"
			if got := IsCloseConfirmedStopReason(reason); got != want {
				t.Errorf("IsCloseConfirmedStopReason(%q)=%v, want %v", reason, got, want)
			}
		})
	}
}
