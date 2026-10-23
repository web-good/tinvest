// Package nvtk supplies the pdl_rsi Params for NVTK (Новатэк) on Minutes15.
//
// # 2026-10-23 — литерал до калибровки
//
// Тикер заведён по просьбе владельца на правилах из спеки (значения core.DefaultParams на
// 2026-10-23). Записаны литералом, чтобы правка дефолтов ядра не меняла тикер молча.
// Прогон 12 мес, -risk-pct 1: 48 сделок, PF 0.485. Калибровка — скользящий walk-forward.
package nvtk

import "tinvest/internal/service/trading_strategy/pdl_rsi/strategy/core"

// Params returns the NVTK literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIArm:         30,
		RSIEntry:       30,
		RSIGate:        50,
		RSIBreakeven:   50,
		RSIExit:        70,
		ZoneATR:        0.4,
		VolumeMult:     1.5,
		VolBars:        20,
		MinRoomATR:     0.6,
		DailyATRPeriod: 14,
	}
}
