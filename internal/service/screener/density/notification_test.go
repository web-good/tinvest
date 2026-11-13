package density

import (
	"strings"
	"testing"
	"time"
)

func TestFormatSignalEscapesAndShowsCoverage(t *testing.T) {
	s := Signal{Ticker: "A<&", Price: Price{Units: 123, Nano: 500000000}, Quantity: 42, Rubles: 5_000_000, Multiple: 5, Distance: .01, Age: time.Minute, TradeLots: 7, TradeCoverage: 15 * time.Second}
	msg := FormatSignal(s)
	for _, part := range []string{"A&lt;&amp;", "123.5", "42", "5000000", "60", "1.00%", "5.00", "7", "15"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %s", part, msg)
		}
	}
}
