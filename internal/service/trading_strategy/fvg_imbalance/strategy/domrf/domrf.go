// Package domrf holds the uncalibrated DOMRF baseline for the backtest-only M30 FVG strategy.
package domrf

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns an explicit starting point for DOMRF calibration.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                    14,  // Период Wilder ATR на M30-свечах.
		MinGapATR:                    0.6, // Минимальная высота FVG в ATR при формировании.
		MaxRetestBars:                10,  // Сколько M30-свечей после формирования зона остаётся активной.
		MinRetestDepthPct:            80,  // Минимальная глубина ретеста от верхней границы FVG, %.
		TakeProfitR:                  2,   // Тейк-профит в R от риска сделки; 0 отключает цель.
		RequireThirdCandleHigherHigh: 0,   // 1 — high третьей свечи должен быть выше high средней; 0 — не требовать.
		RetestDepthByBody:            0,   // 1 — считать глубину по телу ретеста; 0 — по тени.
		UseStochFilter:               0,   // 1 — ждать медленный %K(14,3) < 20 после ретеста.

		HTFTrendEMA:   200, // Период EMA на H4; вход разрешён только если последний закрытый H4 close выше EMA. 0 — фильтр выключен.
		StopBufferATR: 1,   // Дополнительный отступ стопа ниже нижней границы FVG в ATR; 0 — без буфера.
	}
}
