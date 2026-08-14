package lent

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// TestCalibratedLiteralIsPinned держит снимок литерала, выбранного 2026-08-14 по лидербордам
// девяти тем. Тест существует не ради «значения не поменялись», а ради того, чтобы правка
// параметров была ЗАМЕТНОЙ: каждое поле ниже стоит там из-за конкретного замера, и половина из
// них выглядит как очевидный кандидат на «улучшение», пока не знаешь, чем за него платят.
//
// Поля, по которым чаще всего ошибаются на этом тикере:
//
//   - StopDailyATR = 0.7 — НЕ то, что выбрала тема risk. Она выбрала 1.3 во всех четырёх фолдах
//     и дала pooled 2.861, а точечная проверка соседа 1.0 показывает 2.537 против 2.487 у
//     принятой точки. Соблазн сдвинуть стоп шире максимальный. Цена: стоп 1.3 переживается
//     целиком в 78.9% дней (1.0 — в 59.6%, 0.7 — в 29.1%), то есть перестаёт быть выходом и
//     становится страховкой от гэпа, а прирост profit factor берётся из вытеснения убыточных
//     сделок в RSI-выход. Замер на соседе 1.0: фолд 1 даёт OOS PF 15.347, фолд 3 проваливается
//     до 0.987, худшая просадка по фолду растёт с 7.90% до 19.39%. На принятых 0.7 стоп даёт
//     19.0% всех выходов — он работает.
//   - SpentDayATR = 0.8 — НЕ то, что выбрали темы day и day_spent. Обе указали на 1.25, каждая в
//     трёх фолдах из четырёх. Цена этого выбора замерена точкой: сделок остаётся 42 вместо 120,
//     а фолд 1 вырождается в OOS PF 28976 при нуле убыточных сделок — число, которое нельзя
//     ставить в отчёт. Порог 0.8 отбирает 45.5% будних баров и сохраняет выборку.
//   - RSIUpper = 75 при том, что тема exit выбрала 80 в трёх фолдах из четырёх. Она мерила
//     полосу при RSILower = 30; на принятом входе 25 полоса меряется иначе: 75 даёт pooled
//     2.487, 80 — 2.133, 85 — 2.024. Полоса меряется ЦЕЛИКОМ, двигать одну её границу, не
//     перемеряя вторую, нельзя.
//   - RSILower = 25 — устойчивый выбор УЗКОЙ редакции cal_entry.json (25/25/20/25). На широкой
//     редакции той же темы выбор разъезжается (35/50/20/25), а pooled PF падает с 1.935 до
//     1.355. Возврат к baseline 30 стоит дорого: pooled 1.569 при максимальной просадке фолда
//     20.48% против 7.90%.
//   - EMAFast = 20 при EMASlow = 100 — тема trend устойчивости не дала вовсе (EMASlow 150 / 100
//     / 100 / 50), поэтому медленная EMA оставлена на самом частом значении, а быстрая поднята
//     точечным замером: 20 даёт 2.487 на 120 сделках против 2.408 на 111 у 10.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        75,
		EMAFast:         20,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.7,
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
		t.Fatalf("DefaultParams() = %+v, want %+v — литерал обязан совпадать с прогоном, на котором он принят (pooled PF 2.487 на 120 сделках, in-sample 153 сделки при PF 2.083)", got, want)
	}
}

// TestParamsDoNotTrackTheBaseline ловит противоположную беду: литерал, схлопнувшийся обратно в
// core.DefaultParams(). Такой пакет выглядит откалиброванным и молча торгует общими значениями,
// а имя тикера отчёт несёт в обоих случаях. Здесь расходятся пять полей, и это же перечисление
// показывает, что именно калибровка изменила против baseline.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	p, base := DefaultParams(), core.DefaultParams()
	if p == base {
		t.Fatal("LENT вернул baseline: литерал потерян")
	}
	diff := 0
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"RSILower", p.RSILower, base.RSILower},
		{"RSIUpper", p.RSIUpper, base.RSIUpper},
		{"EMAFast", float64(p.EMAFast), float64(base.EMAFast)},
		{"StopDailyATR", p.StopDailyATR, base.StopDailyATR},
		{"TPDailyATR", p.TPDailyATR, base.TPDailyATR},
	} {
		if c.got != c.want {
			diff++
		}
	}
	if diff != 5 {
		t.Fatalf("против baseline отличаются %d поля из пяти ожидаемых — литерал изменился, а обоснование в доке пакета осталось от прежнего", diff)
	}
}

// TestStopIsArmed пинует единственный параметр, обнуление которого молча снимает защиту:
// стратегия держит позицию через ночи и выходные, и StopDailyATR = 0 означает лонг без стопа.
// На LENT стоп даёт 19.0% всех выходов, то есть это не формальность.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// TestRSIExitIsArmed — ловушка нулевого значения: core.Params задаётся литералом, и забытое поле
// даёт UseRSIExit = 0, то есть выключенный ОСНОВНОЙ выход. На LENT по RSI закрывается 75.2%
// сделок и на них приходится вся прибыль (+156 195 из +102 621 чистых), так что его отключение
// меняет стратегию до неузнаваемости, ничего не роняя.
func TestRSIExitIsArmed(t *testing.T) {
	if p := DefaultParams(); p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1", p.UseRSIExit)
	}
}
