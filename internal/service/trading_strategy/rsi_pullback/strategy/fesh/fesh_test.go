package fesh

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал FESH перекалиброван 2026-08-14 на расширенных сетках (шкала RSI с шагом 5 по обеим
// границам). Замер принятой точки: rolling walk-forward на 36 месяцах, 4 фолда, сетка из одной
// точки — калибратор ничего не выбирает — pooled PF 4.406 на 53 сделках, фолды 5.993 / 21.965 /
// 1.840 / 2.260, худший 1.840 против 1.599 у прежнего литерала. Снимок держит литерал равным
// ИМЕННО той конфигурации, которую мерили.
//
// Три поля здесь особенно уязвимы к «улучшению на глаз», и все прибиты сознательно.
//
// TPDailyATR=0.5 против прежней 1.0. На 1.0 цель срабатывала 2 раза из 60 сделок, то есть ось
// была инертна и выборкой не проверена; на 0.5 цель закрывает 15 сделок из 73 и приносит 35 758
// из 63 372 чистого PnL. Это выбор темы risk в 3 фолдах из 4, а не подгонка точки.
//
// StopDailyATR=0.5 против соблазнительных 0.7. Сосед 0.7 даёт формально pooled 7.995, но два его
// фолда показывают OOS PF 97.3 и 250.5 — почти без убыточных сделок. Так широкий стоп покупает
// profit factor, вытесняя убытки в RSI-выход; 0.5 переживается целиком в 13.0% дней и остаётся
// рабочим выходом.
//
// RSILower=17 вне узлов расширенной сетки (шаг 5). Перемерено заново: 15 -> 2.779, 16 -> 3.674,
// 18 -> 3.524, 19 -> 3.541, 20 -> 2.772 против 4.406 на 17 — заменять его на узел не на что.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        17,
		RSIUpper:        55,
		EMAFast:         20,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
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
// 8.2% выходов и −17 757 из брутто-результата.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// RSI-выход на FESH — основной механизм: 71.2% выходов и 45 371 из 63 372 чистого PnL. Забытое
// поле в литерале дало бы UseRSIExit=0 и оставило бы стратегию со стопом и целью; тема trail,
// где этот режим меряется, ушла в убыток (pooled 0.747).
func TestRSIExitIsArmed(t *testing.T) {
	if p := DefaultParams(); p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1", p.UseRSIExit)
	}
}

// Цель перестала быть декорацией: на 0.5 дневного ATR она закрывает каждую пятую сделку. Ноль в
// этом поле выключил бы TP целиком и вернул бы конфигурацию к профилю прежнего литерала, где
// цель не срабатывала практически никогда.
func TestTargetIsArmedAndTighterThanBefore(t *testing.T) {
	p := DefaultParams()
	if p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
	if p.TPDailyATR > p.StopDailyATR*2 {
		t.Fatalf("TPDailyATR = %v при StopDailyATR = %v: цель снова недостижима раньше RSI-выхода",
			p.TPDailyATR, p.StopDailyATR)
	}
}
