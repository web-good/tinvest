package live

import (
	"sort"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// closeStopTickers — тикеры, торгующие стопом по закрытию бара с аварийной биржевой заявкой
// (спека docs/superpowers/specs/2026-10-13-pullback-stop-on-close-live-design.md). Список
// пинится, чтобы режим не включился и не выключился у тикера молча, правкой одного литерала.
var closeStopTickers = []string{
	"AQUA", "BSPB", "CNRU", "DIAS", "ELFV", "FESH", "GAZP", "IVAT", "LENT", "LSNGP", "MDMG",
	"SFIN", "SPBE", "SVAV", "SVCB", "T", "TGKA", "VSMO", "WUSH", "X5", "YDEX",
}

// TestCloseStopLiteralsKeepAnExchangeStop: в режиме StopOnClose на бирже стоит только аварийная
// заявка. Литерал, у которого она не получается (CatStopMult или StopDailyATR не положительны),
// оставил бы позицию между закрытиями баров вовсе без биржевой защиты. CatStopMult без режима —
// мёртвое поле, которое вводит в заблуждение.
func TestCloseStopLiteralsKeepAnExchangeStop(t *testing.T) {
	for ticker, p := range realParams {
		switch {
		case p.StopOnClose == 1 && core.CatastropheStop(p, 100, 10) <= 0:
			t.Errorf("%s: StopOnClose=1, но аварийного уровня нет (CatStopMult=%v, StopDailyATR=%v)", ticker, p.CatStopMult, p.StopDailyATR)
		case p.StopOnClose != 1 && p.CatStopMult != 0:
			t.Errorf("%s: CatStopMult=%v без StopOnClose=1", ticker, p.CatStopMult)
		case p.StopOnClose != 0 && p.StopOnClose != 1:
			t.Errorf("%s: StopOnClose=%d, want 0 или 1", ticker, p.StopOnClose)
		}
	}
}

// TestCloseStopUniverseIsPinned: режим включён ровно у тикеров closeStopTickers.
func TestCloseStopUniverseIsPinned(t *testing.T) {
	var got []string
	for ticker, p := range realParams {
		if p.StopOnClose == 1 {
			got = append(got, ticker)
		}
	}
	sort.Strings(got)
	want := append([]string(nil), closeStopTickers...)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("режим StopOnClose у %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("режим StopOnClose у %v, want %v", got, want)
		}
	}
}
