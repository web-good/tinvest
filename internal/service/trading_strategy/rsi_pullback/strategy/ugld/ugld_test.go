package ugld

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// UGLD торгует живыми деньгами с самого ввода стратегии в прод, и его литерал и есть та
// стратегия, чьи числа стоят в отчётах: любая правка любого поля здесь делает прод НЕ тем, что
// валидировалось, а отчёты — недействительными. Снимок ловит и прямую правку, и случайный дрейф
// вслед за core.DefaultParams(). Менять его можно только вместе с новым прогоном калибровки и
// обновлением docs/rsi_pullback/strategy.md. Текущий снимок — перекалибровка 2026-08-17:
// pooled OOS PF 3.627 на 23 сделках (фолды 3.720 / 2.194 / 7.457 / 0.587), три поля против
// литерала 2026-08-07 — RSIUpper 60 → 55, FreshDayATR 0.3 → 0, TPDailyATR 1.0 → 1.5.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       6,
		RSILower:        15,
		RSIUpper:        55,
		EMAFast:         20,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1.5,
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

// Дневной гейт на UGLD — не украшение: с выключенным гейтом принятая точка даёт pooled OOS PF
// 1.353 на 40 сделках против 3.627 на 23. А вот ветка «день только начался» перекалибровкой
// 2026-08-17 выключена намеренно (она роняла PF до 3.535 и поднимала просадку до 4.87%), и
// перепутать эти два состояния легко: и то и другое выглядит в структуре как «ноль в поле».
// Тест держит именно эту пару: гейт включён, ранняя ветка выключена.
func TestDayGateStaysOnWithTheFreshBranchOff(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1: без дневного гейта конфигурация даёт pooled PF 1.353 вместо 3.627", p.UseDayATRGate)
	}
	if p.FreshDayATR != 0 {
		t.Fatalf("FreshDayATR = %v, want 0: ветка «день только начался» выключена замером 2026-08-17", p.FreshDayATR)
	}
	if p.SpentDayATR <= 0 {
		t.Fatalf("SpentDayATR = %v: при выключенной свежей ветке порог исчерпанности — единственное, что гейтит вход", p.SpentDayATR)
	}
}

// Риском на UGLD правит трейл, а не поле стопа: точки StopDailyATR 0.5, 0.7, 1.0 и 1.3 дают
// побайтово одинаковый результат, потому что desiredStop берёт наибольший уровень, а трейл
// 0.5 ATR от максимума стоит с первого бара ровно там, где фиксированный стоп 0.5. Тест
// сторожит инвариант, при котором это верно (трейл включён и не шире стопа): если трейл
// однажды выключат или расширят, фиксированный стоп станет живым полем, и его значение придётся
// перекалибровать заново — молча этого произойти не должно.
func TestTrailBindsTheRiskNotTheFixedStop(t *testing.T) {
	p := DefaultParams()
	if p.UseTrail != 1 || p.TrailDailyATR <= 0 {
		t.Fatalf("UseTrail = %d, TrailDailyATR = %v: с выключенным трейлом та же точка даёт 2.528 против 3.627, и поле стопа перестаёт быть инертным", p.UseTrail, p.TrailDailyATR)
	}
	if p.TrailDailyATR > p.StopDailyATR {
		t.Fatalf("TrailDailyATR = %v > StopDailyATR = %v: трейл перестал связывать с первого бара, риск поехал", p.TrailDailyATR, p.StopDailyATR)
	}
}

// Стратегия многодневная по замыслу, и полоса выхода — единственное поле, которое этот замысел
// ломает: на принятой полосе 55 медиана удержания 8 баров и 29% сделок длиннее дня, на 45 —
// 6 баров и 14%. Ниже средней линии RSI выход означает закрытие позиции, пока осциллятор ещё в
// нижней половине, то есть скальп вместо отката в тренде.
func TestExitBandStaysAboveTheRSIMidline(t *testing.T) {
	p := DefaultParams()
	if p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1: без RSI-выхода конфигурация даёт pooled PF 0.973", p.UseRSIExit)
	}
	if p.RSIUpper < 50 {
		t.Fatalf("RSIUpper = %v: ниже средней линии RSI стратегия перестаёт быть многодневной (на 45 остаётся 14%% сделок длиннее дня)", p.RSIUpper)
	}
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("RSIUpper = %v <= RSILower = %v: полоса вырождена, выход срабатывал бы на любом отскоке после входа", p.RSIUpper, p.RSILower)
	}
}
