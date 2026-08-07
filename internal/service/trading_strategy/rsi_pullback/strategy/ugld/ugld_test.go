package ugld

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// UGLD — единственный тикер стратегии, прошедший walk-forward, и единственный, которым она
// торгует в проде по умолчанию. Его литерал и есть та стратегия, чьи PF 2.555 / OOS 2.4–3.6
// стоят в отчётах: любая правка любого числа здесь делает прод НЕ тем, что валидировалось, а
// отчёты — недействительными. Снимок ловит и прямую правку, и случайный дрейф вслед за
// core.DefaultParams(). Менять его можно только вместе с новым прогоном калибровки и
// обновлением docs/rsi_pullback/strategy.md.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       6,
		RSILower:        15,
		RSIUpper:        60,
		EMAFast:         20,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1,
		UseVolume:       1,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал UGLD изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Если тикер снова начнёт
// возвращать core.DefaultParams(), снимок выше поймает это только пока baseline отличается —
// а он может однажды совпасть по всем полям и молча снова связать прод с дефолтами.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	base := core.DefaultParams()
	if DefaultParams() == base {
		t.Fatal("UGLD вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}
