// Package gazp holds the GAZP baseline for the backtest-only M30 FVG strategy.
// The 2026-11-04 ordered calibration kept this literal unchanged: its best
// training candidate made zero trades on the intermediate validation window,
// and three rolling OOS folds pooled PF 0.530 on 22 trades. These observations
// do not support promoting a newly selected parameter set.
package gazp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the explicit ticker literal, independent of future core defaults.
func Params() core.Params {
	return core.Params{
		ATRPeriod:         14,  // Период Wilder ATR на M30-свечах.
		MinGapATR:         0.8, // Минимальная высота FVG в ATR при формировании.
		MaxRetestBars:     12,  // Сколько M30-свечей после формирования зона остаётся активной.
		MinRetestDepthPct: 50,  // Минимальная глубина минимума (тени) ретеста от верхней границы FVG, %.
		TakeProfitR:       1,   // Тейк-профит в R от риска сделки; 0 отключает цель.

		RequireFirstRetest:    0,  // 1 — входить только при первом касании зоны; 0 — разрешить последующие.
		RequireBullishBOS:     0,  // 1 — требовать бычий пробой подтверждённого swing high до/при формировании FVG.
		RequireLiquiditySweep: 0,  // 1 — требовать sweep swing low с возвратом закрытия выше уровня перед FVG.
		SwingK:                2,  // Число свечей слева и справа для подтверждения swing high/low.
		StructureLookbackBars: 50, // Глубина поиска swing и структуры перед FVG, в M30-свечах.
		MaxSweepToFVGBars:     15, // Максимум M30-свечей от sweep до формирования FVG.

		HTFTrendEMA:         150, // Период EMA на H4; вход разрешён только если последний закрытый H4 close выше EMA. 0 — фильтр выключен.
		MaxEntryDistanceATR: 0,   // Максимальный отступ цены следующего открытия выше FVG в ATR; 0 — без ограничения.
		MaxHoldBars:         0,   // Закрыть позицию по времени после этого числа M30-свечей; 0 — без ограничения.
		StopBufferATR:       1,   // Дополнительный отступ стопа ниже нижней границы FVG в ATR; 0 — без буфера.
	}
}
