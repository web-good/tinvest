package reni

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал прибит снимком. Он появился 2026-08-12 и переписан 2026-08-16 по итогам
// перекалибровки на расширенных сетках: против прежних чисел изменены EMASlow 100 -> 200 и
// TPDailyATR 1.5 -> 0.6, точка закреплена файлом
// data/params/rsi_pullback/reni/plateau_point.json (pooled OOS PF 2.467 на 79 сделках, фолды
// 2.078 / 3.832 / 1.351 / 2.469). Прежний снимок нёс оговорку «цель 1.5 расходится с
// победителем walk-forward 0.6, и это оставлено осознанно» — оговорка снята: цель 1.5 на RENI
// не срабатывала ни разу за 36 месяцев, поле было инертным, и 0.6 принята.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        20,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         200,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
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
		t.Fatalf("откалиброванный литерал RENI изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Литерал RENI отличается от
// дефолтов ядра немногими полями, поэтому правка core.DefaultParams() способна однажды совпасть
// с ним по всем — и снимок выше этого уже не заметит.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("RENI вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп ограничивает убыток и на этом тикере остаётся РАБОЧИМ выходом — 19% всех выходов на
// полной истории. Его ширина при этом настоящий внутренний максимум оси (0.3 — 1.901,
// 0.5 — 2.467, 0.7 — 2.398 pooled), а не результат вытеснения убытков в RSI-выход: строка 1.0
// даёт больше (2.590) ценой третьего фолда (0.818 против 1.351).
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// Цель обязана быть НИЖЕ 1.0 дневного ATR, иначе она перестаёт срабатывать вовсе. Это замер, а
// не вкус: на 1.0 и на 1.5 точечный прогон даёт побайтово одинаковый результат (2.123), а в
// смеси выходов прежнего литерала с целью 1.5 доля TP равна нулю на 102 сделках за 36 месяцев.
// Такая «цель» — декоративное поле, а стратегия при ней идёт фактически без цели. На принятых
// 0.6 цель закрывает 17% сделок.
func TestTargetActuallyBinds(t *testing.T) {
	p := DefaultParams()
	if p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
	if p.TPDailyATR >= 1.0 {
		t.Fatalf("TPDailyATR = %v: цель от 1.0 дневного ATR на RENI не срабатывает ни разу за 36 месяцев (точки 1.0 и 1.5 дают одинаковые 2.123 против 2.467 на 0.6)", p.TPDailyATR)
	}
}

// Медленная EMA — суть перекалибровки 2026-08-16. Быстрая половина оси разрушает третий фолд,
// то есть глубокий нисходящий режим 2025H2-2026H1: EMASlow 50 / 70 / 100 / 120 дают там
// 0.485 / 0.347 / 0.337 / 0.500, медленная половина 150 / 170 / 200 — 1.078 / 1.073 / 1.351.
// Возврат к прежним 100 вернул бы тикеру ровно тот провал, ради которого его перекалибровали.
func TestTrendFilterStaysSlow(t *testing.T) {
	p := DefaultParams()
	if p.EMASlow < 150 {
		t.Fatalf("EMASlow = %d: быстрее 150 третий фолд разрушается (на 100 он даёт 0.337 против 1.351)", p.EMASlow)
	}
	if p.EMAFast >= p.EMASlow {
		t.Fatalf("трендовая пара вырождена: EMAFast = %d, EMASlow = %d", p.EMAFast, p.EMASlow)
	}
}

// RSI-выход закрывает почти две трети сделок и обязателен: точечный замер с UseRSIExit=0 при
// прочих равных даёт 1.580 против 2.467. Полоса выхода 70 сохранена от прежнего литерала
// осознанно: ранний выход 55 выигрывал только поверх СТАРОЙ быстрой EMA (2.459 против 1.864), а
// поверх принятой EMA 200 проигрывает (2.374) и превращает многодневную стратегию во
// внутридневную — доля сделок длиннее одного дня падает с 29% до 3%.
func TestRSIExitIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1 — без него pooled OOS PF падает с 2.467 до 1.580", p.UseRSIExit)
	}
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("полоса RSI вырождена: RSIUpper = %v, RSILower = %v", p.RSIUpper, p.RSILower)
	}
}

// Дневной гейт держит результат: с UseDayATRGate=0 та же конфигурация даёт 1.476 на 184 сделках
// против 2.467 на 79. Работать обязана ровно одна ветка — «день исчерпан»; ветка «день только
// начался» выключается РОВНО нулём, поскольку dayStateOK охраняет её условием fresh > 0.
// Порог 0.8 — внутренний максимум оси (0.6 — 1.812, 0.9 — 1.934, 1.0 — 2.920 при третьем фолде
// 0.181 на двух сделках).
func TestOnlySpentDayBranchIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1 — без гейта pooled OOS PF падает с 2.467 до 1.476", p.UseDayATRGate)
	}
	if p.FreshDayATR != 0 {
		t.Fatalf("FreshDayATR = %v, want 0 — на RENI работает только ветка «день исчерпан»", p.FreshDayATR)
	}
	if p.SpentDayATR <= 0 {
		t.Fatalf("SpentDayATR = %v, want > 0 — иначе гейт взведён и не отсекает ничего", p.SpentDayATR)
	}
}

// Оба необязательных механизма выключены замером, а не по умолчанию: объёмный гейт даёт 2.449
// ценой одиннадцати сделок из семидесяти девяти, трейл — 2.275. Каждый из них стоит дешевле
// принятой точки, поэтому включение любого обязано начинаться с нового прогона, а не с правки
// литерала.
func TestOptionalMechanicsStayOff(t *testing.T) {
	p := DefaultParams()
	if p.UseVolume != 0 {
		t.Fatalf("UseVolume = %d, want 0 — гейт даёт 2.449 против 2.467 и срезает сделки", p.UseVolume)
	}
	if p.UseTrail != 0 {
		t.Fatalf("UseTrail = %d, want 0 — трейл даёт 2.275 против 2.467", p.UseTrail)
	}
}
