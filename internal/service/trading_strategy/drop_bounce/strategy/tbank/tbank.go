// Package tbank supplies the ticker and the drop_bounce Params for T (МКПАО «Т-Технологии»): one
// literal per timeframe — DefaultParams5m for Minutes5, DefaultParams15m for Minutes15,
// DefaultParams1h for Hour1.
//
// Заведён 2026-10-09 по просьбе владельца. Отчёты — reports/drop_bounce/T/ (вне git), сетки —
// data/params/drop_bounce/t/.
//
// # Протокол
//
// 36 мес, комиссия 0.05% на сторону, вход по open следующего бара, ранжирование по PF. Механика
// взята из литералов GAZP: серия выключена, цель — полтела, старт 07:00, EOD 23:00, выход по
// времени. Первый круг — m5_grid.json / m15_grid.json (по 144 комбинации: порог падения × потолок
// × стоп × выход по времени × порог цели), затем окрестность лучшей точки — m5_small_grid.json
// (36) и m15_small_grid.json (24) с rolling walk-forward 12+6.
//
// Дефолты ядра убыточны на обоих таймфреймах: 5m — PF 0.865/33, 15m — 0.902/68.
//
// Общее для обоих таймфреймов: потолок падения и MinTargetPct ничего не дают (при падении от 1.2%
// цель в полтела ≥ 0.6%) — оба выключены.
//
// # 5 минут
//
//   - MinDropPct: 1.0 → PF 0.71/62, 1.1 → 0.84/46, 1.2 → 1.206/37, 1.3 → 1.371/29,
//     1.4 → 1.092/19. Ниже 1.2 убыточно.
//   - Широкий стоп (2 тела) и выход через 3 бара лучше узкого стопа и раннего выхода.
//   - Точка: 29 сделок, PF 1.371, +3.0 тыс., win rate 62%. По полугодиям 2.32 / 5.01 / 0.53 /
//     7.27 / ∞ / 1.05 — минус только 2025-04…2025-10.
//   - Walk-forward: pooled OOS PF 0.863 на 17 сделках, фолды 0.878 / 1.520 / 0.472 / 0.000;
//     MinDropPct гуляет 1.0–1.6. Края вне выборки нет — литерал лучшая in-sample точка, не
//     одобрение для торговли.
//
// # 15 минут
//
//   - MinDropPct — монотонный рост PF с порогом: 1.1 → 1.321/79, 1.2 → 1.471/61, 1.3 → 1.235/44,
//     1.4 → 1.712/33, 1.5 → 2.169/29, 1.7 → 3.226/22. Крупное падение отскакивает надёжнее.
//   - Взят 1.5, а не 1.7: на 1.7 за три года 22 сделки, и одно полугодие в минусе (0.25/3).
//   - Стоп 0.75 лучше 1.0 и 0.5; выход по закрытию бара исполнения (TimeExitBars 1) — как у GAZP.
//   - Точка: 29 сделок, PF 2.169, +6.5 тыс., win rate 69%, просадка 1.9 тыс. По полугодиям
//     2.24 / 7.81 / 1.40 / 204 / ∞ / 1.01 — все шесть не ниже единицы.
//   - Walk-forward: pooled OOS PF 1.767 на 27 сделках, фолды 1.009 / 10.954 / 982 / 2145, но в
//     фолдах 2–4 по 1–4 сделки. MinDropPct гуляет 1.2–1.7, TimeExitBars стабильно 3.
//     Лучший результат drop_bounce из проверенного; на 29 сделках за три года устойчивость
//     не доказана.
//
// # Памп перед падением (2026-10-09)
//
// PumpBars (10, 20) × PumpPct (0.4–1.5%) или PumpRatio (0.5–1.5 тела падения), сетки
// data/params/drop_bounce/t/pump_grid.json и pump_ratio_grid.json поверх обоих литералов, 36 мес.
// 5m проверялся на литерале владельца MinDrop 0.9 / TimeExitBars 4 (PF 0.776/85). Фильтр
// выключен в обоих литералах.
//
//   - 15m: 20 свечей / 0.8% — PF 3.160/21 против 2.169/29, 10 / 0.4% — 3.635/16, но net не растёт
//     (+6.3 и +6.7 тыс. против +6.5 тыс.): фильтр срезает сделки, а не добавляет прибыль. В долях
//     тела падения любой порог хуже литерала или равен ему.
//   - 5m: все варианты убыточны, лучший 10 / 0.4% — 0.868/57.
//
// # Час (2026-10-09)
//
// Сетка data/params/drop_bounce/t/h1_grid.json — максимально широкая по просьбе владельца:
// 31680 комбинаций основных осей (порог падения 0.5–4% × потолок × цель 0.25–1.5 тела × стоп
// 0.5–3 тела × выход через 1–8 баров × порог цели × старт 07/10) и фильтры поверх десяти лучших
// (EOD 19/23, серия, подтверждение, дневная EMA, памп). Затем окрестность h1_small_grid.json
// (216) с явным порогом падения и rolling walk-forward 12+6. 36 мес, отчёты —
// reports/drop_bounce/T/h1/.
//
//   - Дефолты ядра на часовике: PF 0.805/106, −7.7 тыс.
//   - Победитель широкой сетки — цель в четверть тела при MinTargetPct 0.5, то есть неявный
//     порог падения 2% (PF 8.995/25). Ни один фильтр второй фазы точку не улучшил.
//   - MinDropPct: 1.5 → PF 1.550/70, 2.0 → 7.40/25, 2.5 → 24.7/11. Все соседи порога 2.0 по
//     цели, стопу, выходу и часу старта — PF 5–8.
//   - Цель 0.5 тела вместо 0.25: PF почти тот же (7.40 против 7.76), net выше — +18.3 тыс.
//     против +12.8 тыс. Стоп 1.5 и 2.0 тела совпадают: за три года стоп не сработал ни разу.
//   - Точка: 25 сделок, PF 7.40, +18.3 тыс. По полугодиям 3.56 / 1.68 / 22.2 / ∞ / нет сделок /
//     255 — все в плюсе.
//   - Walk-forward: pooled OOS PF 4.307 на 29 сделках, фолды 3.441 / 2.553 / 243 / 63.7, все
//     четыре прибыльны; MinDropPct гуляет 1.5–2.0. Лучший результат drop_bounce на всех
//     таймфреймах и тикерах, но 25 сделок за три года и стоп без единого срабатывания —
//     хвостовой риск выборкой не измерен.
package tbank

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "T"

// DefaultParams5m is the Minutes5 literal.
func DefaultParams5m() core.Params {
	return core.Params{
		MinDropPct:     0.9,
		MaxDropPct:     0,
		TargetBody:     0.5,
		MinTargetPct:   0,
		StopBody:       2.0,
		StartHour:      7,
		EODHour:        23,
		EODMinute:      0,
		TimeExit:       1,
		TimeExitBars:   4,
		DropSeries:     0,
		DropSeriesBars: 3,
		ConfirmBar:     0,
	}
}

// DefaultParams15m is the Minutes15 literal.
func DefaultParams15m() core.Params {
	return core.Params{
		MinDropPct:     1.5,
		MaxDropPct:     0,
		TargetBody:     0.5,
		MinTargetPct:   0,
		StopBody:       0.75,
		StartHour:      7,
		EODHour:        23,
		EODMinute:      0,
		TimeExit:       1,
		TimeExitBars:   1,
		DropSeries:     0,
		DropSeriesBars: 3,
		ConfirmBar:     0,
	}
}

// DefaultParams1h is the Hour1 literal.
func DefaultParams1h() core.Params {
	return core.Params{
		MinDropPct:     2.0,
		MaxDropPct:     0,
		TargetBody:     0.5,
		MinTargetPct:   0,
		StopBody:       1.5,
		StartHour:      7,
		EODHour:        23,
		EODMinute:      0,
		TimeExit:       1,
		TimeExitBars:   2,
		DropSeries:     0,
		DropSeriesBars: 3,
		ConfirmBar:     0,
	}
}
