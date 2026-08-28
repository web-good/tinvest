package banep

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestParamsMatchTheCalibratedSnapshot прибивает литерал BANEP снимком: тест падает и на молчаливом
// дрейфе поля, и на откате к core.DefaultParams(). Снимок сделан 2026-08-28 по принятой точке
// (data/params/rsi_pullback/banep/plateau_point.json): pooled OOS PF 1.799 на 67 сделках, три фолда
// из четырёх прибыльны; на полной истории 96 сделок при PF 1.590, под удвоенными издержками 1.317.
//
// ПЛАНКА НЕ ВЗЯТА (entry — pooled OOS PF 1.004 при устойчивости 2 из 4; trend — 1.258 при 2 из 4),
// литерал стоит по решению владельца, а не потому, что протокол подтвердил edge. Обоснование
// каждого поля — в доке пакета.
func TestParamsMatchTheCalibratedSnapshot(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         5,
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    1.5,
		TPDailyATR:      0.6,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("литерал BANEP разошёлся со снимком:\n got: %+v\nwant: %+v", got, want)
	}
}

// TestParamsAreNotTheBaseline страхует от отката пакета в состояние «калибровка не проводилась»:
// такой откат тихо увёл бы прод на дефолты ядра, которые на BANEP дают PF 1.336 против 1.590 у
// точки и — что важнее — стоп 0.5 ATR, роняющий PF до 1.087 под реальным кругом издержек.
func TestParamsAreNotTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("BANEP вернул core.DefaultParams(): литерал потерян")
	}
}

func TestTickerIsBANEP(t *testing.T) {
	if Ticker != "BANEP" {
		t.Fatalf("Ticker = %q, want BANEP", Ticker)
	}
}
