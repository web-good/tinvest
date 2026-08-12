package reni

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал RENI взят с конкретного прогона 2026-08-12:
// reports/RENI/RENI_rsi_pullback_Minutes30_20260812_005046.md (74 сделки, in-sample PF 1.885 за
// 24 месяца). Снимок держит его равным параметрам ИМЕННО того отчёта — иначе перепроверка
// отчёта мерила бы не то, что стоит в коде.
//
// Особое внимание к TPDailyATR. Walk-forward того же дня
// (RENI_rsi_pullback_Minutes30_20260812_003547_walkforward.md, 4 фолда, pooled OOS PF 2.060 на
// 78 сделках) назвал стабильным во всех фолдах значение 0.6, а здесь стоит 1.5 — единственное
// поле, где литерал расходится с победителем walk-forward. Расхождение оставлено осознанно и
// прибито снимком: «поправить на 0.6, ведь walk-forward так сказал» — правка, которая молча
// подменит основание литерала и сделает отчёт 005046 недействительным.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        20,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал RENI изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Литерал RENI отличается от
// дефолтов ядра всего двумя полями (RSILower и TPDailyATR), поэтому правка core.DefaultParams()
// способна однажды совпасть с ним по всем полям — и снимок выше этого уже не заметит.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("RENI вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR оставил бы позицию без уровня.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
