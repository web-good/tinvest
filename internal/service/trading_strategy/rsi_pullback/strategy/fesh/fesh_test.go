package fesh

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал FESH принят 2026-08-13 по замеру: rolling walk-forward на 36 месяцах (4 фолда,
// сетка из одной точки — калибратор не выбирает) дал pooled PF 2.366 на 44 сделках, худший
// фолд 1.60, ни одного вырожденного. Снимок держит литерал равным ИМЕННО той конфигурации,
// которую мерили: без него любая перепроверка мерила бы не то, что стоит в коде.
//
// Два поля здесь особенно уязвимы к «улучшению на глаз», и оба прибиты сознательно.
//
// SpentDayATR=0.8 против 1.0. Значение 1.0 стояло в первом выборе владельца и выглядит
// круглее; замер: 1.0 даёт тот же pooled PF (2.310), но на 26 сделках вместо 44, худший фолд
// 1.23 вместо 1.60 и +14.46% вместо +24.75%. Плато лежит на 0.8/0.9/1.0, левый край 0.6
// обваливается до 1.330 — то есть 0.8 не край, а начало плато.
//
// FreshDayATR=0.25 против 0.2. Правка 0.25 → 0.2 была сделана и откачена в тот же день:
// 0.2 тождественно 0.15 (порог уходит ниже минимума дневного размаха, между ними нет ни одного
// бара) и роняет результат до 2.114 на 42 сделках при худшем фолде 1.10. Плато — 0.25–0.30.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        17,
		RSIUpper:        62,
		EMAFast:         10,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.25,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1.0,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал FESH изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Снимок выше сравнивает с
// конкретными числами, а не с дефолтами ядра, поэтому правка core.DefaultParams() однажды
// способна совпасть с литералом по всем полям — и снимок этого не заметит.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("FESH вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Нулевой StopDailyATR оставил бы позицию без уровня. На FESH это не абстракция: SL забирает
// 15% выходов, из них ноль выигрышных, и половину брутто-прибыли.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// RSI-выход на FESH — не вспомогательный механизм, а основной: 82% выходов и вся прибыль
// (TP сработал 2 раза из 60 за три года). Забытое поле в литерале дало бы UseRSIExit=0 и
// оставило бы стратегию с одним стопом — замер такой конфигурации: PF 1.373 против 2.310.
func TestRSIExitIsArmed(t *testing.T) {
	if p := DefaultParams(); p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1", p.UseRSIExit)
	}
}
