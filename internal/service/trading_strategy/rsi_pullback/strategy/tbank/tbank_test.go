package tbank

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Литерал прибит снимком. Он появился 2026-08-16 по итогам ПЕРВОЙ полной калибровки тикера и
// заменил числа одиночного in-sample прогона от 2026-07-31, у которого walk-forward не было
// вовсе. Против прежнего литерала изменены ровно два поля — UseTrail и TrailDailyATR, — и
// именно поэтому снимок нужен целиком: следующая правка соседнего поля «заодно» обязана валить
// сборку, а не тихо уезжать в прод вместе с трейлом.
func TestCalibratedLiteralIsPinned(t *testing.T) {
	want := core.Params{
		RSIPeriod:       5,
		RSILower:        20,
		RSIUpper:        65,
		EMAFast:         20,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
	if got := DefaultParams(); got != want {
		t.Fatalf("откалиброванный литерал T изменился:\n got: %+v\nwant: %+v", got, want)
	}
}

// Отдельно от снимка: связь с baseline обязана быть разорвана. Если тикер начнёт возвращать
// core.DefaultParams(), снимок выше поймает это только пока baseline отличается — а он может
// однажды совпасть по всем полям и молча снова связать тикер с дефолтами.
func TestParamsDoNotTrackTheBaseline(t *testing.T) {
	if DefaultParams() == core.DefaultParams() {
		t.Fatal("T вернул core.DefaultParams(): откалиброванный тикер не должен отслеживать baseline")
	}
}

// Фиксированный стоп остаётся полом позиции и при взведённом трейле: трейл считается от
// максимума благоприятных ЗАКРЫТИЙ, поэтому у сделки, которая ни разу не закрылась выше входа,
// он ниже стопа, и DesiredStop (берёт наибольший уровень) отдаёт стоп. На полной истории таких
// сделок 4 из 44. Ноль здесь означал бы удержание через ночи и выходные без пола вообще.
func TestStopIsArmed(t *testing.T) {
	if p := DefaultParams(); p.StopDailyATR <= 0 {
		t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
	}
}

// Трейл — единственное, что принятая точка изменила против прежнего боевого литерала, и оба его
// поля обязаны быть согласованы. UseTrail=1 при TrailDailyATR=0 — мёртвое число (DesiredStop
// охраняет трейл условием TrailDailyATR > 0), ровно та ошибка, которую вычищали из fesh.
// TrailDailyATR СТРОГО МЕНЬШЕ стопа — это уже не трейл, а ужесточение стартового уровня: он
// встаёт ближе фиксированного стопа с первого бара, ещё до всякого движения цены, и такая
// правка принадлежит cal_risk.json. Замер это подтверждает: 0.2 — 1.091, 0.3 — 1.573,
// 0.4 — 1.976 против 3.412 pooled на принятых 0.5.
func TestTrailIsArmedAndDoesNotDisguiseAStopChange(t *testing.T) {
	p := DefaultParams()
	if p.UseTrail != 1 {
		t.Fatalf("UseTrail = %d, want 1 — без трейла pooled OOS PF падает с 3.412 до 2.552, а первый фолд под единицу (0.836)", p.UseTrail)
	}
	if p.TrailDailyATR <= 0 {
		t.Fatalf("TrailDailyATR = %v при UseTrail=1: трейл выключен условием TrailDailyATR > 0, поле стало мёртвым числом", p.TrailDailyATR)
	}
	if p.TrailDailyATR < p.StopDailyATR {
		t.Fatalf("TrailDailyATR = %v уже стопа %v: это ужесточение стартового стопа под именем трейла (0.4 даёт 1.976 против 3.412)", p.TrailDailyATR, p.StopDailyATR)
	}
}

// RSI-выход закрывает семь сделок из десяти и на этом тикере и есть источник результата: сигнал
// входа края не несёт (медиана хода через два торговых дня после кросса вниз — от −0.06% до
// −0.12% при доле роста 44-46% во всех двенадцати замеренных сочетаниях). Точечный замер с
// UseRSIExit=0 при прочих равных даёт pooled 1.912 против 3.412, а четвёртый фолд рушится до
// 0.157. Отключить его — сменить стратегию выхода, а не настроить параметр.
func TestRSIExitIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseRSIExit != 1 {
		t.Fatalf("UseRSIExit = %d, want 1 — без него pooled OOS PF падает с 3.412 до 1.912, четвёртый фолд до 0.157", p.UseRSIExit)
	}
	if p.RSIUpper <= p.RSILower {
		t.Fatalf("полоса RSI вырождена: RSIUpper = %v, RSILower = %v", p.RSIUpper, p.RSILower)
	}
}

// Дневной гейт держит результат: с UseDayATRGate=0 та же конфигурация даёт pooled 1.145 на 173
// сделках против 3.412 на 35. Работать обязана ровно одна ветка — «день исчерпан». Ветка «день
// только начался» на T вредна (порог 0.3 роняет результат до 1.665 на 68 сделках), и выключает
// её ровно ноль: dayStateOK охраняет ветку условием fresh > 0, поэтому 0.01 её ОСТАВИТ.
func TestOnlySpentDayBranchIsArmed(t *testing.T) {
	p := DefaultParams()
	if p.UseDayATRGate != 1 {
		t.Fatalf("UseDayATRGate = %d, want 1 — без гейта pooled OOS PF падает с 3.412 до 1.145", p.UseDayATRGate)
	}
	if p.FreshDayATR != 0 {
		t.Fatalf("FreshDayATR = %v, want 0 — ветка «день только начался» на T вредна (1.665 при пороге 0.3)", p.FreshDayATR)
	}
	if p.SpentDayATR <= 0 {
		t.Fatalf("SpentDayATR = %v, want > 0 — иначе гейт взведён и не отсекает ничего", p.SpentDayATR)
	}
}

// Объёмный гейт отвергнут не по числу: при множителе 2.0 и базе 10 дней он даёт pooled 3.729 на
// 26 сделках без вырожденных фолдов — единственный близкий конкурент принятой точки. Отвергнут
// по цене и по устойчивости: он срезает четверть сделок на тикере, где связывающее ограничение
// — счёт сделок; отсекать ему на обороте 4961 млн ₽ нечего; а соседняя настройка того же гейта
// (множитель 2.5) немедленно вырождает второй фолд (7201 на четырёх сделках). Без трейла тот же
// гейт даёт 2.856 с вырожденным четвёртым фолдом — работает трейл, а не он.
func TestVolumeGateStaysOff(t *testing.T) {
	if p := DefaultParams(); p.UseVolume != 0 {
		t.Fatalf("UseVolume = %d, want 0 — гейт срезает четверть сделок, а его соседняя настройка вырождает фолд (7201 на 4 сделках)", p.UseVolume)
	}
}

// Цель обязана превышать стоп: иначе конфигурация требует win rate выше 50% просто чтобы выйти
// в ноль. На принятой точке отношение 3:1 (1.5 против 0.5), и соседи цели ниже: 1.0 — 3.200,
// 2.0 — 2.930 против 3.412.
func TestTargetClearsTheStop(t *testing.T) {
	p := DefaultParams()
	if p.TPDailyATR <= p.StopDailyATR {
		t.Fatalf("TPDailyATR = %v не выше StopDailyATR = %v: асимметрия риска к прибыли потеряна", p.TPDailyATR, p.StopDailyATR)
	}
}
