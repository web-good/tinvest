// engine_zone_test.go прогоняет zone-сделку через настоящий движок бэктеста (backtest.Run), а не
// через Decide напрямую: так проверяется передача sig.ATR в Position.EntryATR, от которого живёт
// стоп позиции, и то, что причина «zone:» доезжает до журнала сделок.
package core

import (
	"math"
	"strings"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

func TestEngineRunsAZoneTradeEndToEnd(t *testing.T) {
	p := DefaultParams()
	p.EMAFast = p.EMASlow // гейт тренда pullback закрыт: войти может только zone
	p.UseZoneEntry, p.ZoneRSIPeriod, p.ZoneRSILower, p.ZoneEMAPeriod = 1, 4, 25, 200
	s := NewWithParams("TEST", p)
	lookback := s.Lookback()
	if lookback != 420 {
		t.Fatalf("Lookback() = %d, want 420 (тест рассчитан на ZoneEMAPeriod 200)", lookback)
	}

	const width = 10.0                                 // дневной ATR будних дневок
	exitBar := time.Date(2026, 6, 1, 12, 0, 0, 0, msk) // понедельник
	entryBar := exitBar.Add(-30 * time.Minute)

	// Ровный рост +0.08% за бар и три бара по −0.2%: RSI(4) пересекает 25 вниз ровно на третьем
	// (та же форма, на которой проверялось удалённое ядро rsi_zone), close далеко над EMA(200).
	closes := make([]float64, 0, lookback+3)
	price := 100.0
	for i := 0; i < lookback; i++ {
		price *= 1.0008
		closes = append(closes, price)
	}
	for i := 0; i < 3; i++ {
		price *= 0.998
		closes = append(closes, price)
	}
	entryPrice := closes[len(closes)-1]

	start := entryBar.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	candles := make([]bt.Candle, 0, len(closes)+1)
	for i, c := range closes {
		candles = append(candles, bt.Candle{
			Time: start.Add(time.Duration(i) * 30 * time.Minute),
			Open: c, High: c * 1.003, Low: c * 0.997, Close: c,
		})
	}
	// Бар выхода — гэп вниз под уровень стопа: движок обязан исполнить по min(уровень, open).
	level := entryPrice - p.StopDailyATR*width
	gapOpen := entryPrice - 3*width
	candles = append(candles, bt.Candle{
		Time: exitBar, Open: gapOpen, High: gapOpen, Low: gapOpen - 1, Close: gapOpen - 0.5,
	})

	h, l, c, ts := dailyBars(exitBar, 40, width, width/10)
	daily := make([]bt.Candle, len(c))
	for i := range c {
		daily[i] = bt.Candle{Time: ts[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}

	res := bt.Run(s, candles, daily, nil, bt.Config{InitialCash: 1_000_000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1 (zone-вход на кресте, SL на гэпе)", len(res.Trades))
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-entryPrice) > 1e-6 {
		t.Fatalf("EntryPrice = %v, want %v: вход не на задуманном баре", tr.EntryPrice, entryPrice)
	}
	if !strings.HasPrefix(tr.EntryReason, "zone:") {
		t.Fatalf("EntryReason = %q, want prefix zone:", tr.EntryReason)
	}
	if math.Abs(tr.ATR-width) > 1e-9 {
		t.Fatalf("ATR сделки = %v, want %v: ATR сигнала должен дойти до Position.EntryATR", tr.ATR, width)
	}
	if tr.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", tr.Reason)
	}
	if want := math.Min(level, gapOpen); math.Abs(tr.ExitPrice-want) > 1e-6 {
		t.Fatalf("ExitPrice = %v, want %v = min(уровень %v, open гэпа %v)", tr.ExitPrice, want, level, gapOpen)
	}
}
