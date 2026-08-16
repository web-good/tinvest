package nvtk

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал прибит снимком: он появился 2026-08-16 и заменил состояние «калибровка не
// проводилась», в котором пакет возвращал core.DefaultParams(). Легче всего ошибиться на двух
// полях, и оба выбраны против первого впечатления. RSILower=50 выглядит как «не откат вовсе», но
// ось глубины на NVTK монотонна в обратную каталогу сторону (30 — 1.229, 40 — 2.361, 50 — 2.823
// pooled), и 50 — определительный край шкалы, а не обрезанный. SpentDayATR=0.9 выглядит
// произвольным между 0.8 и 1.0, а на деле это единственная точка оси, где profit factor уже
// вырос (2.823 против 1.868 на 0.8), но фолды ещё не выродились (на 1.0 первый и четвёртый дают
// 6.928 и 6.650 на 20 и 19 сделках). Разбор каждого поля — в доке пакета.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        50,
		RSIUpper:        75,
		EMAFast:         10,
		EMASlow:         50,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
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
		t.Fatalf("откалиброванный литерал NVTK изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Если тикер снова начнёт
// возвращать core.DefaultParams(), снимок выше поймает это только пока baseline отличается —
// а он может однажды совпасть по всем полям и молча снова связать тикер с дефолтами.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("NVTK вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус (68.6%
// сделок), цель срабатывает в 6.6%. На NVTK стоп при этом не формальность, а четверть всех
// выходов (24.8%), и его ширина — настоящий внутренний максимум оси (0.4 — 2.035, 0.5 — 2.823,
// 0.7 — 2.345 pooled), а не результат вытеснения убытков в другой выход, как это было на GAZP.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// RSI-выход держит две трети сделок: точечный замер с UseRSIExit=0 при прочих равных даёт
// pooled OOS PF 1.567 на 78 сделках против 2.823 на 93, win rate падает с 76.3% до 46.1%, а
// третий фолд уходит под единицу (0.782). Отключить его — значит сменить стратегию выхода, а не
// настроить параметр.
func TestRSIExitIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1 — без него pooled OOS PF падает с 2.823 до 1.567", p.UseRSIExit)
	}
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("полоса RSI вырождена: RSIUpper = %v, RSILower = %v", p.RSIUpper, p.RSILower)
	}
}

// Дневной гейт на этом тикере и есть источник результата: с UseDayATRGate=0 та же конфигурация
// даёт pooled OOS PF 0.996 на 614 сделках против 2.823 на 93. Работать при этом обязана ровно
// одна ветка — «день исчерпан»: включение ветки «день только начался» роняет результат до 1.373
// (порог 0.3), 1.313 (0.4) и 1.184 (0.5), покупая сделки ценой качества.
func TestOnlySpentDayBranchIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1 — без гейта pooled OOS PF падает с 2.823 до 0.996", p.UseDayATRGate)
	}
	if p.FreshDayATR != 0 {
		t.Fatalf("FreshDayATR = %v, want 0 — ветка «день только начался» на NVTK вредна (1.373 при пороге 0.3)", p.FreshDayATR)
	}
	if p.SpentDayATR <= 0 {
		t.Fatalf("SpentDayATR = %v, want > 0 — иначе гейт взведён и не отсекает ничего", p.SpentDayATR)
	}
}

// Объёмный гейт отвергнут не по вкусу, а по разбору вырождения: с ним pooled PF выше (3.032 при
// множителе 1.2 и базе 10 дней), но первый фолд даёт 17.146 на 19 сделках — тот же артефакт
// ранжирования по profit factor, что разобран на GAZP. Тема screen выбирала гейт во всех
// четырёх фолдах именно поэтому, и это не довод за него.
func TestVolumeGateStaysOff(t *testing.T) {
	if p := DefaultParams(); p.UseVolume != 0 {
		t.Fatalf("UseVolume = %d, want 0 — гейт покупает pooled PF вырожденным первым фолдом (17.146 на 19 сделках)", p.UseVolume)
	}
}

// Цель обязана превышать стоп: иначе конфигурация требует win rate выше 50% просто чтобы выйти
// в ноль. На принятой точке отношение 2:1 (1.0 против 0.5), и ось цели вокруг него пологая
// (0.7 — 2.677, 0.8 — 2.710, 1.0 — 2.823, 1.2 — 2.365).
func TestTargetClearsTheStop(t *testing.T) {
	p := DefaultParams()
	if p.TPDailyATR <= p.StopDailyATR {
		t.Fatalf("TPDailyATR = %v не выше StopDailyATR = %v: асимметрия риска к прибыли потеряна", p.TPDailyATR, p.StopDailyATR)
	}
}
