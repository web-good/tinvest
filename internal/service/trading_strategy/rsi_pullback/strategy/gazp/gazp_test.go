package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// GAZP торгуется в проде, поэтому его литерал прибит снимком: любая правка любого числа здесь
// меняет прод. Снимок держит конфигурацию перекалибровки 2026-08-15 (замеры — в доке пакета),
// заменившую литерал от 2026-08-05 ровно в одном поле: StopDailyATR 0.5 -> 0.7. Легче всего
// ошибиться именно на нём и на RSIUpper: по profit factor лучше выглядят стоп 1.0 (pooled 2.943
// против 2.125) и уровень выхода 75 (2.215), но первый переживается целиком в 60% дней и даёт
// лишь 5.5% всех выходов, а второй не выбрал ни один фолд темы exit и куплен ростом просадки —
// оба выбора сделаны против лидерборда осознанно.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         70,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.7,
		TPDailyATR:      0.7,
		UseVolume:       0,
		VolBaseDays:     7,
		VolLookbackBars: 2,
		VolMult:         1,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал GAZP изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Если тикер снова начнёт
// возвращать core.DefaultParams(), снимок выше поймает это только пока baseline отличается —
// а он может однажды совпасть по всем полям и молча снова связать прод с дефолтами.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("GAZP вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Стоп — единственное, что ограничивает убыток: RSI-выход закрывает и в плюс, и в минус (он
// забирает 82.2% сделок), а цель сработала лишь в 6.8% случаев. Нулевой StopDailyATR оставил бы
// многодневную позицию без уровня вовсе.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// RSI-выход на этом тикере и есть стратегия выхода: точечный замер с UseRSIExit=0 при прочих
// равных даёт pooled OOS PF 0.846 против 2.125. Отключить его — значит сменить стратегию, а не
// настроить параметр.
func TestRSIExitIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1 — без него pooled OOS PF падает с 2.125 до 0.846", p.UseRSIExit)
	}
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("полоса RSI вырождена: RSIUpper = %v, RSILower = %v", p.RSIUpper, p.RSILower)
	}
}

// Ветка «день только начался» выключена намеренно: точечно её включение роняет pooled OOS PF с
// 2.125 до 1.186 (порог 0.3), 0.959 (0.4) и 1.001 (0.5), покупая сделки ценой качества. Гейт
// дня при этом обязан остаться взведённым — работает вторая ветка, «день исчерпан».
func TestOnlySpentDayBranchIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1", p.UseDayATRGate)
	}
	if p.FreshDayATR != 0 {
		t.Fatalf("FreshDayATR = %v, want ровно 0 — ветка «день только начался» выключается только нулём", p.FreshDayATR)
	}
	if p.SpentDayATR <= 0 {
		t.Fatalf("SpentDayATR = %v: обе ветки гейта выключены, UseDayATRGate=1 не отсекает ничего", p.SpentDayATR)
	}
}
