package enum

import (
	"testing"
	"time"
)

func TestMinutes5(t *testing.T) {
	if Minutes5 != 2 {
		t.Fatalf("Minutes5 = %d, want 2", Minutes5)
	}
	if got := Minutes5.String(); got != "Minutes5" {
		t.Fatalf("String() = %q, want Minutes5", got)
	}
	if got := Minutes5.ToTimeDuration(); got != 5*time.Minute {
		t.Fatalf("ToTimeDuration() = %v, want 5m", got)
	}
	if got := Minutes5.ToNumberInvestAPI(); got != 2 {
		t.Fatalf("ToNumberInvestAPI() = %d, want 2", got)
	}
}

func TestHour4ToTimeDuration(t *testing.T) {
	if Hour4 != 11 {
		t.Fatalf("Hour4 = %d, want 11", Hour4)
	}
	if got := Hour4.String(); got != "Hour4" {
		t.Fatalf("String() = %q, want Hour4", got)
	}
	if got := Hour4.ToTimeDuration(); got != 4*time.Hour {
		t.Fatalf("Hour4.ToTimeDuration() = %v, want 4h", got)
	}
}
