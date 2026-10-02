package live

import (
	"testing"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/shared"
)

// Сторожевой тест литерала: живые параметры правятся только осознанно, вместе с этим тестом.
func TestSharedParamsLiteral(t *testing.T) {
	want := core.Params{GapATR: 0.7, MaxGapATR: 3.0, EntryBar: 0, TargetFill: 0.75,
		StopDailyATR: 1.0, DailyATRPeriod: 14, EODHour: 18, EODMinute: 30}
	if got := shared.Params(); got != want {
		t.Fatalf("shared.Params() = %+v, want %+v", got, want)
	}
}

// Любой непустой тикер получает общий литерал: позиция тикера, убранного из вселенной,
// доводится до выхода. Входы ограничивает вселенная, а не реестр.
func TestRegistryServesSharedParamsForAnyTicker(t *testing.T) {
	for _, tk := range []string{"SBER", "NOT_IN_UNIVERSE"} {
		p, ok := ParamsFor(tk)
		if !ok || p != shared.Params() {
			t.Fatalf("ParamsFor(%q) = %+v, %v", tk, p, ok)
		}
		st, ok := StrategyFor(tk)
		if !ok || st.Ticker() != tk || st.Lookback() != 64 {
			t.Fatalf("StrategyFor(%q) = %v, %v", tk, st, ok)
		}
	}
}

func TestRegistryRejectsBlankTicker(t *testing.T) {
	if _, ok := ParamsFor(" "); ok {
		t.Fatal("пустой тикер зарегистрирован")
	}
	if st, ok := StrategyFor(""); ok || st != nil {
		t.Fatal("пустой тикер дал стратегию")
	}
}
