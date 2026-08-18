package ivat

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал прибит снимком: он появился 2026-08-18 и заменил состояние «калибровка не
// проводилась», в котором пакет возвращал core.DefaultParams(). Два поля выбраны против выбора
// тематических прогонов, и оба — сознательно. EMASlow=70 стоит там, где тема trend не дала
// ответа вовсе: её выбор развернулся на середине истории (200 / 170 / 50 / 50), а точечный
// замер показал плоское поле — тринадцать пар дают 1.59-2.02, и 70 это середина полки быстрых
// пар с соседями 10/50 (1.868) и 10/100 (1.769). StopDailyATR=0.7 стоит против выбора темы
// risk (1.3 в трёх фолдах из четырёх), потому что ось стопа на IVAT — учебный случай
// вытеснения убытков: profit factor растёт монотонно 1.192 -> 1.781, доля стоп-выходов падает
// 45.0% -> 6.6%, а счёт сделок не меняется (149 -> 137). Разбор каждого поля — в доке пакета.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        30,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         70,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.7,
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
		t.Fatalf("откалиброванный литерал IVAT изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Если тикер снова начнёт
// возвращать core.DefaultParams(), снимок выше поймает это только пока baseline отличается —
// а он может однажды совпасть по всем полям и молча снова связать тикер с дефолтами.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("IVAT вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

func TestTickerIsIVAT(t *testing.T) {
	if Ticker != "IVAT" {
		t.Fatalf("Ticker = %q, want IVAT", Ticker)
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус.
// Ширина 0.7 выбрана как крайняя точка, где стоп ещё работает: по оси доля стоп-выходов идёт
// 45.0% (0.3) / 27.5% (0.5) / 18.7% (0.7) / 10.9% (1.0) / 6.6% (1.3), и дальше стоп
// перестаёт быть стопом. На нём же минимальная просадка всей оси (13.14% против 17.02% у 0.5 и
// 15.79% у 1.3), хотя profit factor одиночного прогона там не максимален.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// RSI-выход закрывает больше половины сделок (51-55% на замерах точки). Точечный замер с
// UseRSIExit=0 при прочих равных даёт pooled OOS PF 1.422 на 49 сделках против 1.988 на 55.
// Тема trail выключала его в трёх фолдах из четырёх, но живой раннер держит RSI-выход
// обязательным для всех тикеров реестра, и на IVAT это подтверждено числом.
func TestRSIExitIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1 — без него pooled OOS PF падает с 1.988 до 1.422", p.UseRSIExit)
	}
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("полоса RSI вырождена: RSIUpper = %v, RSILower = %v", p.RSIUpper, p.RSILower)
	}
}

// Дневной гейт отбирает больше, чем любой другой фильтр конфигурации: с UseDayATRGate=0 та же
// точка даёт pooled OOS PF 1.140 на 199 сделках против 1.988 на 55. Тема screen выключала его
// во всех четырёх фолдах — она мерила гейт поверх дефолтного порога 0.8, и это довод против
// порога, а не против гейта. Работает при этом ровно одна ветка, «день исчерпан»: включение
// ветки «день только начался» роняет точку до 1.663 (порог 0.3) и 1.224 (0.5), покупая сделки
// ценой качества.
func TestOnlySpentDayBranchIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1 — без гейта pooled OOS PF падает с 1.988 до 1.140", p.UseDayATRGate)
	}
	if p.FreshDayATR != 0 {
		t.Fatalf("FreshDayATR = %v, want 0 — ветка «день только начался» на IVAT вредна (1.663 при пороге 0.3)", p.FreshDayATR)
	}
	if p.SpentDayATR <= 0 {
		t.Fatalf("SpentDayATR = %v, want > 0 — иначе гейт взведён и не отсекает ничего", p.SpentDayATR)
	}
}

// Объёмный гейт отвергнут по разбору выборки, а не по вкусу: на теме он давал pooled 1.388, но
// весь отрыв делал фолд из восьми сделок (5.237), а на самой точке UseVolume=1 даёт 2.053 на
// 48 сделках против 1.988 на 55 — прибавка 0.065 profit factor куплена четвертью выборки,
// причём первый фолд при этом падает с 1.271 до 0.982.
func TestVolumeGateStaysOff(t *testing.T) {
	if p := DefaultParams(); p.UseVolume != 0 {
		t.Fatalf("UseVolume = %d, want 0 — гейт забирает четверть выборки ради 0.065 profit factor", p.UseVolume)
	}
}

// Цель НИЖЕ стопа (0.6 против 0.7), и это осознанная асимметрия наоборот — та же, что у WUSH и
// LSNGP (0.5 при 0.7). Она держится на замере: ось цели даёт 2.035 (0.5) / 2.018 (0.6) / 1.548
// (0.8) / 1.619 (1.0) / 1.415 (1.5), то есть за 0.6 идёт обрыв, потому что цель перестаёт
// достигаться раньше RSI-выхода — доля TP-выходов падает с 26% до 15%. Конфигурация с целью
// ближе стопа требует win rate выше 50%, и он есть: 63.6% на принятой точке. Тест сторожит
// границу, за которой цель становится недостижимой: шире удвоенного стопа её ставить нельзя.
func TestTargetIsArmedAndTighterThanTheStop(t *testing.T) {
	p := DefaultParams()
	if p.TPDailyATR <= 0 {
		t.Fatalf("TPDailyATR = %v, want > 0", p.TPDailyATR)
	}
	if p.TPDailyATR > p.StopDailyATR*2 {
		t.Fatalf("TPDailyATR = %v при StopDailyATR = %v: цель недостижима раньше RSI-выхода (1.415 на 1.5 ATR)",
			p.TPDailyATR, p.StopDailyATR)
	}
}

// Трейл выключен: тема с принудительным UseTrail дала pooled 1.083 против 1.296 у той же
// конфигурации без него, а на самой точке трейл 0.5 ATR роняет 1.988 до 1.754. Поле
// TrailDailyATR при этом обязано быть нулём, иначе выключенный трейл несёт непроверенное
// значение, которое оживёт при первом же переключении флага.
func TestTrailStaysOff(t *testing.T) {
	p := DefaultParams()
	if p.UseTrail != 0 {
		t.Fatalf("UseTrail = %d, want 0 — трейл 0.5 ATR роняет точку с 1.988 до 1.754", p.UseTrail)
	}
	if p.TrailDailyATR != 0 {
		t.Fatalf("TrailDailyATR = %v при выключенном трейле, want 0", p.TrailDailyATR)
	}
}
