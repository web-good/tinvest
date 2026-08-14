package domrf

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// DOMRF торгуется в проде, поэтому его литерал прибит снимком: любая правка любого числа здесь
// меняет прод. Снимок держит конфигурацию перекалибровки 2026-08-14 (замеры — в доке пакета),
// заменившую литерал от 2026-08-10. Особенно легко ошибиться на StopDailyATR и SpentDayATR:
// по profit factor лучше выглядят широкий стоп 1.3 и порог дня 1.25, но первый переживается
// целиком в 83.7% дней, а второй срезает выборку до 28 сделок — оба выбора сделаны против
// лидерборда осознанно.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        35,
		RSIUpper:        70,
		EMAFast:         20,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     1.0,
		StopDailyATR:    0.7,
		TPDailyATR:      1.0,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
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

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус, а цель
// на выборке из 65 сделок сработала дважды. Нулевой StopDailyATR оставил бы позицию без уровня.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// Ветка «день только начался» включена перекалибровкой 2026-08-14 впервые и держит 24 OOS-сделки
// из 42: без неё третий фолд протокола схлопывается до одной сделки. Гейт дня защищён условием
// fresh > 0, поэтому ровно ноль выключает ветку целиком — это и стережёт тест.
func TestFreshDayBranchIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1", p.UseDayATRGate)
	}
	if p.FreshDayATR <= 0 || p.FreshDayATR >= p.SpentDayATR {
		t.Fatalf("FreshDayATR = %v при SpentDayATR = %v: ветка «день только начался» выключена или перекрыта",
			p.FreshDayATR, p.SpentDayATR)
	}
}
