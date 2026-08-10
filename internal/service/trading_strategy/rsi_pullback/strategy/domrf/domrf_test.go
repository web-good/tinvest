package domrf

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// DOMRF торгуется в проде по решению владельца, принятому на конкретном отчёте:
// reports/DOMRF/DOMRF_rsi_pullback_Minutes30_20260810_131435.md. Снимок держит литерал равным
// параметрам ИМЕННО того отчёта — любая правка любого числа здесь делает прод не тем, что
// смотрели, а отчёт-основание недействительным. Особенно легко ошибиться на UseTrail: победитель
// walk-forward-прогона того же дня брал UseTrail=1, и подмена выглядела бы улучшением.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         5,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     1,
		StopDailyATR:    1,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     7,
		VolLookbackBars: 2,
		VolMult:         1,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал DOMRF изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Если тикер снова начнёт
// возвращать core.DefaultParams(), снимок выше поймает это только пока baseline отличается —
// а он может однажды совпасть по всем полям и молча снова связать прод с дефолтами.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("DOMRF вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус, а TP на
// выборке из 29 сделок не сработал ни разу. Нулевой StopDailyATR оставил бы позицию без уровня.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}
