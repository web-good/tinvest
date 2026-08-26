package ydex

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал прибит снимком 2026-08-26. ПЛАНКА НЕ ВЗЯТА, и это записано пункт за пунктом. Планка
// требует от ОБЕИХ ключевых тем pooled OOS PF >= 1.5 при >= 20 сделках И устойчивость ведущей оси
// >= 3 фолда из 4. (1) Тема entry: pooled OOS PF 1.916 на 52 сделках — критерий PF ВЗЯТ; ведущая
// ось RSILower по фолдам 10/25/35/25 — устойчивость 2 из 4, критерий устойчивости ПРОВАЛЕН.
// (2) Тема trend: pooled OOS PF 1.642 на 71 сделке — критерий PF ВЗЯТ; ведущая ось EMASlow по
// фолдам 200/50/200/70 — устойчивость 2 из 4, критерий устойчивости ПРОВАЛЕН. Итого два критерия
// из четырёх провалены. ФОРМА ПРОВАЛА НЕ НОВАЯ — она повторяет ELFV (entry 1.575/106 при
// устойчивости 1 из 4, trend 1.545/147 при устойчивости 0 из 4): обе темы взяли PF и обе провалили
// устойчивость. Это второй такой случай в каталоге, а не первый; YDEX отличается от ELFV в лучшую
// сторону по обеим осям сразу (запас по PF больше, устойчивость выше — 2 из 4 против 1 и 0).
// Тикер принят по правилу владельца, объявленному ДО прогонов: литерал ставится и при непройденной
// планке, если принятая точка не упирается в стоп-условие плана. Стоп-условие проверено по всем
// трём пунктам и НЕ сработало: pooled OOS PF 3.014 против порога 1.0 (запас 3.0x), 77 сделок в
// OOS-пуле против порога 20 (запас 3.85x), под удвоенными издержками -commission 0.001 pooled OOS
// PF 2.167 на тех же 77 сделках. Фолды точки 2.915(17)/1.849(20)/4.920(26)/2.765(14) — все четыре
// прибыльные. СХЕМА ПРОГОНОВ АДАПТИРОВАНА (окно укорочено из-за редомициляции YNDX->YDEX):
// -interval Minutes30 -months 25 -train-months 9 -test-months 4 -min-trades 20
// -metric profit_factor, комиссия по умолчанию 0.0005 (круг 0.1%). От core.DefaultParams()
// литерал отличается РОВНО ТРЕМЯ полями: RSIPeriod 4->3, RSILower 30->25, SpentDayATR 0.8->0.9.
// Разбор каждого поля — в доке пакета (ydex.go), все числа — в
// data/params/rsi_pullback/ydex/plateau_point.json.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       3,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
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
		t.Fatalf("откалиброванный литерал YDEX изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Снимок выше поймает возврат к
// дефолтам только пока baseline отличается хоть одним полем — а он может однажды совпасть по всем
// восемнадцати и молча снова связать тикер с ядром.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("YDEX вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

func TestTickerIsYDEX(t *testing.T) {
	if Ticker != "YDEX" {
		t.Fatalf("Ticker = %q, want YDEX", Ticker)
	}
}

// Стоп удержан на ДЕФОЛТЕ ЯДРА (0.5) ВОПРЕКИ моде темы risk: фолды выбрали 0.7/1.3/1.3/1.3 —
// широкий стоп победил в трёх из четырёх. Капкан широкого стопа воспроизведён ПОВЕРХ СОБРАННОЙ
// ТОЧКИ (reports/YDEX_point_trap/, полная 25-месячная история, стоп -> сделок -> PF -> просадка):
// 0.3 -> 115 -> 1.636 -> 4728.69; 0.5 -> 110 -> 2.866 -> 4030.54; 0.7 -> 110 -> 2.950 -> 5585.98;
// 1.0 -> 109 -> 3.152 -> 5076.01; 1.3 -> 109 -> 3.126 -> 4784.38. По PF широкий стоп выше
// (3.152 против 2.866), но просадка на 0.5 минимальна — отказ от широкого стопа тай-брейк
// контроллера внутри узкого по PF участка, а не вывод самого прогона.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

func TestStopStaysAboveTheCostFloor(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR < 0.3 {
		t.Fatalf("StopDailyATR = %v: на стопе 0.3 ATR реальный круг издержек 0.125%% съедает 14.6%% риска, уже — ещё больше", p.StopDailyATR)
	}
}

// Цель осталась на ДЕФОЛТЕ ЯДРА (0.6) — мода фолдов темы risk (0.4/0.6/1.0/0.6) и одновременно
// максимум точечного прогона под остальными осями точки по PF и net (reports/YDEX_pt_probe_risk/,
// строка стопа 0.5): 0.6 -> 2.866/110/38672.27; 0.5 -> 2.835/111; 1.0 -> 2.819/109; 0.8 -> 2.729/109.
func TestTargetIsArmed(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
}

func TestTargetStaysReachable(t *testing.T) {
	if p := DefaultParams(); p.TPDailyATR > 1.5 {
		t.Fatalf("TPDailyATR = %v: на YDEX цель шире 1.5 дневного ATR недостижима — колонки 1.5, 2.0 и 2.5 замера совпадают побайтово", p.TPDailyATR)
	}
}

func TestTrendPairIsValid(t *testing.T) {
	if p := DefaultParams(); p.EMAFast >= p.EMASlow {
		t.Fatalf("EMAFast = %d >= EMASlow = %d: трендовый фильтр вырожден или инвертирован", p.EMAFast, p.EMASlow)
	}
}

func TestEntryBandIsBelowTheExitBand(t *testing.T) {
	if p := DefaultParams(); p.RSILower >= p.RSIUpper {
		t.Fatalf("RSILower = %v >= RSIUpper = %v: вход и выход перепутаны", p.RSILower, p.RSIUpper)
	}
}
