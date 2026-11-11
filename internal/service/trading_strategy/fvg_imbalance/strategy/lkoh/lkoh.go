// Package lkoh holds the LKOH baseline for the backtest-only M30 FVG strategy.
package lkoh

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the user-requested stage-3 research candidate; it did not pass
// the reserved-year validation. Training 2023-11-11–2025-11-11: no-stochastic-
// exit branch, expectancy 1334.81, 26 trades (13 in each year), Net PnL +34705.11,
// PF 2.961, Max DD 7.11%. Final year 2025-11-11–2026-11-11: 21 trades,
// Net PnL -2088.20, PF 0.921, Max DD 20.09% (limits: positive Net PnL, PF > 1,
// Max DD <= 10%, at least 10 final-year trades). Applied at the user's request
// despite the failed validation; see data/params/fvg_imbalance/lkoh/research_2026-11/outcome.md.
func Params() core.Params {
	return core.Params{
		RequireEntryAboveFVGLower: 1,
		ATRPeriod:                 14,   // Период дневного ATR.
		MinGapATR:                 0.25, // Минимальный размер FVG в дневных ATR.
		MaxWaitBars:               45,   // Сколько M30-баров ждать сигнал после FVG.
		StochPeriod:               9,    // Период стохастика %K.
		UseStochExit:              0,    // Выход по пересечению %K уровня 80; 0 отключает.
		StopATR:                   0.5,  // Расстояние стопа от цены входа в ATR.
		TakeProfitATR:             2,    // Расстояние цели в ATR; 0 отключает ценовую цель.
		HTFTrendEMA:               0,    // Период дневной EMA; 0 отключает фильтр тренда.
	}
}
