// Package afks supplies the pdl_reclaim Params for AFKS (АФК Система).
//
// # 2026-10-21 — заведён
//
// Литерал — общие настройки GAZP на 5m: только возврат над PDL по молоту, стоп не ближе
// PDLMinStopATR 0.4 ATR, цель 1.5R, объём серии 0.5, объём свечей сигнала 0.5, MaxDayRangeATR 1.2,
// IndexTrendEMA 150, подъём стопа 1/0.25. Калибровка — ниже.
package afks

import "tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"

// Params is the AFKS Minutes15 literal.
func Params() core.Params {
	return core.Params{
		EntryPDL:          1,
		EntryPDH:          0,
		HoldBars:          3,
		StopDailyATR:      0.7,
		DailyATRPeriod:    14,
		VolumeMult:        1.5,
		VolBars:           20,
		MaxDayRangeATR:    1.2,
		IndexTrendEMA:     150,
		PDLEngulf:         0,
		PDLHammer:         1,
		PDLDipVolMult:     0.9,
		PDLReclaimBars:    0,
		PDLHammerEntry:    0,
		PDLSignalVolMult:  1,
		PDLPatternTargetR: 1.5,
		PDLMinStopATR:     0.4,
		LockAfterR:        1,
		LockAtR:           0.3,
	}
}
