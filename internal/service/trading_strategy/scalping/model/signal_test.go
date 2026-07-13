package model

import "testing"

func TestIsStopReason(t *testing.T) {
	tests := []struct {
		reason string
		want   bool
	}{
		{reason: "SL", want: true},
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
