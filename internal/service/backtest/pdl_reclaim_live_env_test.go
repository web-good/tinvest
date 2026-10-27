package backtest

import "testing"

// Вселенная pdl_reclaim есть в env/prod.env, и каждый её тикер имеет пакет с литералом: тикер
// без пакета торговал бы на дефолтах ядра, которые никто не калибровал.
func TestPDLReclaimLiveTickersAreRegistered(t *testing.T) {
	const key = "PDL_RECLAIM_TICKERS_M15"
	tickers, ok := prodEnvTickerLists(t)[key]
	if !ok {
		t.Fatalf("%s не найден в env/prod.env", key)
	}
	for _, tk := range tickers {
		if _, ok := pdlReclaimRegistry[tk]; !ok {
			t.Errorf("%s есть в %s env/prod.env, но не зарегистрирован в pdlReclaimRegistry", tk, key)
		}
	}
}
