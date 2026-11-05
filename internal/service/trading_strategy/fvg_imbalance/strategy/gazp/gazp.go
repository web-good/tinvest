// Package gazp holds the GAZP baseline for the backtest-only M30 FVG strategy.
// The 2026-11-05 ordered calibration selected this literal on the 27-month
// training window (PF 1.485 on 80 trades). Its six-month validation was weak
// (PF 1.107 on 27 trades), rolling OOS was unstable (pooled PF 1.043 on 47
// trades), and the final three-month holdout lost (PF 0.709 on 16 trades).
// It is promoted by explicit operator choice despite failing the holdout.
package gazp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the explicit ticker literal, independent of future core defaults.
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
