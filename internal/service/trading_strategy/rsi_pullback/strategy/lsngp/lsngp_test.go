package lsngp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestCalibratedLiteralIsPinned — снимок литерала, принятого 2026-08-14. До калибровки это место
// занимал TestParamsTrackTheBaseline, державший состояние «калибровка не проводилась».
//
// Снимок нужен не ради самого факта равенства: пять полей ниже выбраны ПРОТИВ очевидного
// прочтения лидербордов, и разбор каждого живёт в доке пакета. Правка «по вкусу» тихо унесла бы
// вместе с числом и довод, поэтому каждое уязвимое поле подписано прямо здесь.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        55,
		EMAFast:         10,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    0.7,
		TPDailyATR:      0.5,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}

	p := DefaultParams()
	for _, c := range []struct {
		field string
		got   float64
		want  float64
		why   string
	}{
		{
			"RSIUpper", float64(p.RSIUpper), 55,
			"край шкалы выхода: 55 → pooled 2.129, 60 → 2.055, 65 → 2.002, 70 (baseline) → 1.890 при " +
				"RSILower 30; это же поле задаёт характер сделки — медиана удержания падает до 4 баров",
		},
		{
			"RSILower", float64(p.RSILower), 30,
			"НЕ углубляется: 25 даёт убыточный четвёртый фолд (0.872), 20 при RSIPeriod 5 — 53 сделки, " +
				"15 — 24 сделки и фолд с PF 5139 при нуле убыточных",
		},
		{
			"FreshDayATR", p.FreshDayATR, 0.3,
			"ветка «день только начался» включена ради выборки: 0 даёт pooled 3.299 на 192 сделках, " +
				"0.3 — 2.954 на 271 при более ровных фолдах; взят счёт сделок, а не максимум PF",
		},
		{
			"StopDailyATR", p.StopDailyATR, 0.7,
			"НЕ победитель темы risk (там 1.3 в 3 фолдах из 4) и выбран не по PF: на этой оси PF монотонен " +
				"(0.5→1.893, 0.7→2.493, 1.0→2.904, 1.3→3.148) при падающей доле стоп-выходов (12.9→6.5→3.9→2.1%), " +
				"то есть меряет вытеснение убытков. 0.7 = 84% медианного дневного размаха (0.83 ATR)",
		},
		{
			"TPDailyATR", p.TPDailyATR, 0.5,
			"цель МЕНЬШЕ стопа — асимметрия, подтверждённая темой risk во всех четырёх фолдах при оси, " +
				"свипующей цель до 2.5; держится на win rate 76.83% и делает цель рабочим выходом (15.0% сделок)",
		},
	} {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v — %s", c.field, c.got, c.want, c.why)
		}
	}
}

// Литерал обязан ОТЛИЧАТЬСЯ от baseline ядра ровно в пяти полях. Тест ловит две разные аварии:
// откат литерала к baseline (тикер тихо перестал быть откалиброванным) и расползание правок на
// поля, которых калибровка не касалась.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	got, base := DefaultParams(), core.DefaultParams()
	if got == base {
		t.Fatal("DefaultParams() совпал с baseline: литерал потерян")
	}
	diff := 0
	for _, same := range []bool{
		got.RSIPeriod == base.RSIPeriod,
		got.RSILower == base.RSILower,
		got.RSIUpper == base.RSIUpper,
		got.EMAFast == base.EMAFast,
		got.EMASlow == base.EMASlow,
		got.DailyATRPeriod == base.DailyATRPeriod,
		got.UseDayATRGate == base.UseDayATRGate,
		got.FreshDayATR == base.FreshDayATR,
		got.SpentDayATR == base.SpentDayATR,
		got.StopDailyATR == base.StopDailyATR,
		got.TPDailyATR == base.TPDailyATR,
		got.UseVolume == base.UseVolume,
		got.VolBaseDays == base.VolBaseDays,
		got.VolLookbackBars == base.VolLookbackBars,
		got.VolMult == base.VolMult,
		got.UseRSIExit == base.UseRSIExit,
		got.UseTrail == base.UseTrail,
		got.TrailDailyATR == base.TrailDailyATR,
	} {
		if !same {
			diff++
		}
	}
	if diff != 5 {
		t.Fatalf("литерал отличается от baseline в %d полях, want 5 (RSIUpper, EMASlow, FreshDayATR, StopDailyATR, TPDailyATR)", diff)
	}
}

// Стоп обязан быть взведён: стратегия держит позицию через ночи и выходные, и конфигурация без
// стопа означает открытый риск гэпа. Ни одна сетка каталога не свипует StopDailyATR = 0, и
// литерал не имеет права прийти к нему в обход сеток.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// RSI-выход — основной выход этой конфигурации: 78.9% сделок за 36 месяцев. Его выключение
// оставило бы позицию на стопе и цели, то есть на другой стратегии.
func TestRSIExitIsArmed(t *testing.T) {
	if p := DefaultParams(); p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %v, want 1", p.UseRSIExit)
	}
}

func TestTicker(t *testing.T) {
	if Ticker != "LSNGP" {
		t.Fatalf("Ticker = %q, want LSNGP", Ticker)
	}
}
