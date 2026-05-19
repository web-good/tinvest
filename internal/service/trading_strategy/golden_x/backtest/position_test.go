package backtest

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestPosition_OpenAndStop(t *testing.T) {
	entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
	p := OpenPosition("SBER", entry, 100.0, dto.Stop{Price: 90.0, DistancePct: 10}, dto.StrategyKindDividend)

	if p.UnitsRemaining() != 1.0 {
		t.Fatalf("UnitsRemaining = %v, want 1.0", p.UnitsRemaining())
	}
	if p.FullyClosed() {
		t.Fatalf("FullyClosed = true at open, want false")
	}
	exitWeek := entry.AddDate(0, 0, 7)
	tr := p.CloseAll(exitWeek, 90.0, ExitReasonStop)
	if tr.ExitReason != ExitReasonStop {
		t.Fatalf("ExitReason = %v, want %v", tr.ExitReason, ExitReasonStop)
	}
	if !p.FullyClosed() {
		t.Fatalf("FullyClosed = false after CloseAll")
	}
	// Returns: (90-100)/100 * 100 = -10
	if math.Abs(tr.ReturnPct - -10.0) > 1e-9 {
		t.Fatalf("ReturnPct = %v, want -10", tr.ReturnPct)
	}
	if tr.WeeksHeld != 1 {
		t.Fatalf("WeeksHeld = %d, want 1", tr.WeeksHeld)
	}
}

func TestPosition_GoldPartialExits(t *testing.T) {
	entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
	p := OpenPosition("SBER", entry, 100.0, dto.Stop{Price: 80.0}, dto.StrategyKindDividend)

	// Week 4: RSI crosses p80 first time → 1/3 sold at 110.
	w4 := entry.AddDate(0, 0, 7*3)
	partials := p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 65)
	if len(partials) != 0 {
		t.Fatalf("RSI 65 below P80 70 → no exits, got %d", len(partials))
	}
	partials = p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 72)
	if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP80 {
		t.Fatalf("expected one P80 partial, got %+v", partials)
	}
	if math.Abs(partials[0].Units-1.0/3.0) > 1e-9 {
		t.Fatalf("Units = %v, want 1/3", partials[0].Units)
	}
	if math.Abs(p.UnitsRemaining()-2.0/3.0) > 1e-9 {
		t.Fatalf("UnitsRemaining after P80 = %v, want 2/3", p.UnitsRemaining())
	}

	// Week 5: RSI now in P90 zone → P90 partial. P80 already triggered, does NOT re-fire.
	w5 := entry.AddDate(0, 0, 7*4)
	partials = p.EvaluateSellExits(w5, 120.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 85)
	if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP90 {
		t.Fatalf("expected one P90 partial, got %+v", partials)
	}

	// Week 6: RSI in P95 zone → P95 partial. Position fully closed.
	w6 := entry.AddDate(0, 0, 7*5)
	partials = p.EvaluateSellExits(w6, 130.0, dto.SellThresholds{P80: 70, P90: 80, P95: 90}, 95)
	if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP95 {
		t.Fatalf("expected one P95 partial, got %+v", partials)
	}
	if !p.FullyClosed() {
		t.Fatalf("FullyClosed = false after all 3 partials")
	}
}

func TestPosition_GoldAllTiersInOneWeek(t *testing.T) {
	// Spec §4.3 framing: "three independent decisions". If a single week
	// jumps RSI past P95 with no prior partials, all three must fire.
	entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
	p := OpenPosition("X", entry, 100.0, dto.Stop{Price: 80.0}, dto.StrategyKindDividend)
	st := dto.SellThresholds{P80: 70, P90: 80, P95: 90}
	partials := p.EvaluateSellExits(entry.AddDate(0, 0, 7), 120.0, st, 95)
	if len(partials) != 3 {
		t.Fatalf("expected 3 partials in one week, got %d: %+v", len(partials), partials)
	}
	reasons := [3]ExitReason{partials[0].ExitReason, partials[1].ExitReason, partials[2].ExitReason}
	if reasons != [3]ExitReason{ExitReasonSellP80, ExitReasonSellP90, ExitReasonSellP95} {
		t.Fatalf("reason order = %v, want P80,P90,P95", reasons)
	}
	if !p.FullyClosed() {
		t.Fatalf("FullyClosed = false after all three fired")
	}
}

func TestPosition_GrowthSingleSellExit(t *testing.T) {
	entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
	p := OpenPosition("YNDX", entry, 100.0, dto.Stop{}, dto.StrategyKindGrowth)

	w4 := entry.AddDate(0, 0, 7*3)
	// Below P90 → no exit.
	partials := p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P90: 80}, 70)
	if len(partials) != 0 {
		t.Fatalf("RSI below P90 → no exits, got %d", len(partials))
	}
	partials = p.EvaluateSellExits(w4, 110.0, dto.SellThresholds{P90: 80}, 85)
	if len(partials) != 1 || partials[0].ExitReason != ExitReasonSellP90 {
		t.Fatalf("expected one P90 full exit, got %+v", partials)
	}
	if math.Abs(partials[0].Units-1.0) > 1e-9 {
		t.Fatalf("Growth exit Units = %v, want 1.0", partials[0].Units)
	}
	if !p.FullyClosed() {
		t.Fatalf("FullyClosed = false after Growth P90 exit")
	}
}

func TestPosition_WeeksHeld(t *testing.T) {
	entry := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)
	p := OpenPosition("X", entry, 100.0, dto.Stop{}, dto.StrategyKindDividend)
	// Exactly 52 weeks later → WeeksHeld must equal 52.
	exit := entry.AddDate(0, 0, 7*52)
	if got := p.WeeksHeldAt(exit); got != 52 {
		t.Fatalf("WeeksHeldAt = %d, want 52", got)
	}
}
