// Package domrf supplies the ticker and calibrated rsi_pullback Params for DOMRF (ДОМ.РФ).
//
// ОТКАЛИБРОВАН 2026-08-10 и ПРИНЯТ ВЛАДЕЛЬЦЕМ В ПРОД — но, в отличие от [ugld], не как тикер,
// прошедший протокол walk-forward, а как осознанно принятый риск. Литерал взят из отчёта
// reports/DOMRF/DOMRF_rsi_pullback_Minutes30_20260810_131435.md (29 сделок, PF 5.206,
// win rate 86.2%, DD 3.38%) и совпадает с ним по всем восемнадцати полям, включая
// UseVolume=0 / UseTrail=0. Совпадение обязано сохраняться: победитель последнего
// walk-forward-прогона того же дня брал UseTrail=1, и если менять литерал на него, то
// отчёт-основание перестаёт описывать прод.
//
// Что именно принято как риск (замерено на данных, не мнение):
//   - История инструмента — 8.6 месяца: IPO 2025-11-20, 235 дневных баров. После прогрева
//     EMA(150) торговля в бэктесте начинается 2025-12-17, то есть окно 6.2 месяца. Заявленный
//     в отчёте период 15 месяцев и посчитанный на нём CAGR 12.97% — артефакт растянутого
//     знаменателя, сравнивать по ним с другими тикерами нельзя.
//   - Вся история — одна фаза рынка: пост-IPO аптренд 1764 -> 2468 (+40% до пика). Лонг-онли
//     откуп откатов в таком режиме даёт завышенный profit factor при любых параметрах.
//     Единственный неблагоприятный участок (июнь-август, -9.5% от пика) дал одну сделку, и ту
//     убыточную; трендовый фильтр при этом честно отработал — с 24 июня входов нет вовсе.
//   - Out-of-sample по существу один фолд: 8 сделок, PF 2.1, +2.3% за квартал. Первый фолд
//     walk-forward обучался на окне, которого у инструмента нет, и его in-sample PF в тысячах —
//     деление на ноль убытков, а не метрика.
//   - Профиль сделок перекошен: payoff 0.83 (средний выигрыш 817 против убытка 981), всё
//     держится на win rate, чей интервал Уилсона по 29 сделкам — [69%, 95%]. По нижней границе
//     PF около 1.9.
//   - TPDailyATR=1.5 не сработал НИ РАЗУ: 28 выходов из 29 по RSI, один по стопу. Цель в
//     литерале стоит, но выборкой не проверена.
//
// Дневной ATR(14) идёт медианой 1.94% цены — половина от 4.28% у UGLD и один класс с GAZP и T;
// поэтому сетки в data/params/rsi_pullback/domrf/ пересчитаны, а не скопированы у соседа.
//
// Пересматривать литерал — только новым walk-forward на 18-24 месяцах истории, когда она
// накопится, и вместе с обновлением docs/rsi_pullback/live.md §10. Литерал прибит domrf_test.go.
package domrf

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "DOMRF"

// DefaultParams returns DOMRF's calibrated rsi_pullback parameters.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         5,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     1,
		StopDailyATR:    1,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     7,
		VolLookbackBars: 2,
		VolMult:         1,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
}
