// Package sber holds the SBER baseline for the backtest-only M30 FVG strategy.
package sber

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the selected SBER calibration candidate.
// Selected 2026-11-11 from the 2023-11-11—2025-11-11 training window
// (no-stochastic-exit branch, expectancy 2459.31 RUB, 23 closed trades:
// 12/11 by year, PF 2.755, MaxDD 9.18%). The reserved 2025-11-11—2026-11-11
// check produced 12 closed trades, +12304.79 RUB net, PF 1.909, MaxDD 10.56%,
// and an open position; it failed the pre-set MaxDD <= 10% limit. Kept as the
// best training candidate at the user's request, not as an OOS-passed calibration.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,  // Период дневного ATR.
		MinGapATR:                 0.3, // Минимальный размер FVG в дневных ATR.
		MaxWaitBars:               45,  // Сколько M30-баров ждать сигнал после FVG.
		StochPeriod:               10,  // Период стохастика %K.
		UseStochExit:              0,   // Выход по пересечению %K уровня 80; 0 отключает.
		StopATR:                   2,   // Расстояние стопа от цены входа в ATR.
		TakeProfitATR:             2,   // Расстояние цели в ATR; 0 отключает ценовую цель.
		HTFTrendEMA:               0,   // Период дневной EMA; 0 отключает фильтр тренда.
		RequireEntryAboveFVGLower: 1,
	}
}
