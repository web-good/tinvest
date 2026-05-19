package backtest

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func makeTrade(share string, entry time.Time, retPct, units float64, reason ExitReason) Trade {
	return Trade{
		ShareID:    share,
		EntryDate:  entry,
		EntryPrice: 100,
		ExitDate:   entry.AddDate(0, 0, 7*4),
		ExitPrice:  100 + retPct,
		Units:      units,
		ExitReason: reason,
		ReturnPct:  retPct,
		WeeksHeld:  4,
	}
}

func TestStats_WinRateAndCumulative(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	trades := []Trade{
		makeTrade("A", base, 10, 1, ExitReasonSellP90),
		makeTrade("A", base.AddDate(0, 0, 7), -5, 1, ExitReasonStop),
		makeTrade("B", base, 0, 1, ExitReasonTimeout),
		makeTrade("C", base, 20, 1, ExitReasonOpen),
	}
	st := AggregateStats(trades)
	if st.Count != 4 || st.Wins != 1 || st.Losses != 1 || st.Open != 1 {
		t.Fatalf("counts: %+v", st)
	}
	if math.Abs(st.WinRate-0.5) > 1e-9 {
		t.Fatalf("WinRate=%v, want 0.5", st.WinRate)
	}
	if math.Abs(st.CumulativeReturn-25.0) > 1e-9 {
		t.Fatalf("CumulativeReturn=%v, want 25", st.CumulativeReturn)
	}
}

func TestStats_MaxDrawdownNonMonotonic(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	trades := []Trade{
		makeTrade("A", base.AddDate(0, 0, 0), 20, 1, ExitReasonSellP90),
		makeTrade("A", base.AddDate(0, 0, 7), -30, 1, ExitReasonStop),
		makeTrade("A", base.AddDate(0, 0, 14), 10, 1, ExitReasonSellP90),
		makeTrade("A", base.AddDate(0, 0, 21), -5, 1, ExitReasonStop),
	}
	st := AggregateStats(trades)
	if math.Abs(st.MaxDrawdown-30.0) > 1e-9 {
		t.Fatalf("MaxDrawdown=%v, want 30", st.MaxDrawdown)
	}
}

func TestRenderMarkdown_ContainsSections(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	trades := []Trade{
		makeTrade("A", base, 10, 1, ExitReasonSellP90),
		makeTrade("B", base, -5, 1, ExitReasonStop),
	}
	r := Report{
		Kind:    dto.StrategyKindDividend,
		From:    base,
		To:      base.AddDate(0, 1, 0),
		Trades:  trades,
		Overall: AggregateStats(trades),
		PerShare: map[string]Stats{
			"A": AggregateStats(trades[:1]),
			"B": AggregateStats(trades[1:]),
		},
	}
	out := RenderMarkdown(r)
	for _, want := range []string{"## Overall", "## Exit reasons", "## Per share", "## Trades"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing section %q", want)
		}
	}
}
