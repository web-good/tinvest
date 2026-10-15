package backtest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dropBounceLiveKeys — вселенная drop_bounce в env/prod.env (стратегия только на 15m).
var dropBounceLiveKeys = []string{"DROP_BOUNCE_TICKERS_M15"}

// dropBounceLiveTickers читает вселенные drop_bounce из env/prod.env: ключ — переменная.
func dropBounceLiveTickers(t *testing.T) map[string][]string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../internal/service/backtest -> корень репо
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "env", "prod.env"))
	if err != nil {
		t.Fatalf("read env/prod.env: %v", err)
	}
	out := map[string][]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, v, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found {
			continue
		}
		for _, tk := range strings.Split(v, ",") {
			if tk = strings.TrimSpace(tk); tk != "" {
				out[key] = append(out[key], tk)
			}
		}
		if _, ok := out[key]; !ok {
			out[key] = nil
		}
	}
	return out
}

// Каждая вселенная drop_bounce есть в env/prod.env, и каждый её тикер имеет пакет с литералом:
// тикер без пакета торговал бы на дефолтах ядра, которые никто не калибровал.
func TestDropBounceLiveTickersAreRegistered(t *testing.T) {
	lists := dropBounceLiveTickers(t)
	for _, key := range dropBounceLiveKeys {
		tickers, ok := lists[key]
		if !ok {
			t.Errorf("%s не найден в env/prod.env", key)
			continue
		}
		for _, tk := range tickers {
			if _, ok := dropBounceRegistry[tk]; !ok {
				t.Errorf("%s есть в %s env/prod.env, но не зарегистрирован в dropBounceRegistry", tk, key)
			}
		}
	}
}

// Ключи прежних таймфреймов удалены: с 2026-10-15 drop_bounce идёт на 15m (сигнал и MACD 1h,
// подтверждение MACD 15m), и возвращение _M30/_H1 в env значило бы, что кто-то торгует его на
// старых правилах.
func TestDropBounceNoLegacyTimeframeKeys(t *testing.T) {
	lists := dropBounceLiveTickers(t)
	for _, key := range []string{"DROP_BOUNCE_TICKERS_M30", "DROP_BOUNCE_TICKERS_H1"} {
		if _, ok := lists[key]; ok {
			t.Errorf("%s есть в env/prod.env, drop_bounce работает только на 15m", key)
		}
	}
}
