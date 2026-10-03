package live

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

// prodTickers читает RSI_ZONE_TICKERS из env/prod.env.
func prodTickers(t *testing.T) []string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../internal/service/trading_strategy/rsi_zone/live -> корень репо
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "env", "prod.env"))
	if err != nil {
		t.Fatalf("read env/prod.env: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if v, found := strings.CutPrefix(line, "RSI_ZONE_TICKERS="); found {
			var out []string
			for _, tk := range strings.Split(v, ",") {
				if tk = strings.TrimSpace(tk); tk != "" {
					out = append(out, tk)
				}
			}
			if len(out) == 0 {
				t.Fatal("RSI_ZONE_TICKERS в env/prod.env пуст")
			}
			return out
		}
	}
	t.Fatal("RSI_ZONE_TICKERS не найден в env/prod.env")
	return nil
}

// Каждый тикер из RSI_ZONE_TICKERS env/prod.env зарегистрирован — иначе раннер его молча
// не торгует (алерт «не зарегистрирован» на каждом пассе).
func TestEveryProdTickerIsRegistered(t *testing.T) {
	for _, tk := range prodTickers(t) {
		if _, ok := ParamsFor(tk); !ok {
			t.Errorf("%s есть в RSI_ZONE_TICKERS env/prod.env, но не зарегистрирован в live-реестре", tk)
		}
	}
}

// Тикер на дефолтах ядра в боевой вселенной — только именным исключением с причиной.
// BAZA была им до 2026-10-03: поверх дефолтов ядра включён выход PROFIT (док пакета strategy/baza).
var baselineByDesignTickers = map[string]string{}

func TestBaselineTickersInProdAreNamedExceptions(t *testing.T) {
	prod := map[string]bool{}
	for _, tk := range prodTickers(t) {
		prod[tk] = true
		p, ok := ParamsFor(tk)
		if !ok {
			continue // покрыто TestEveryProdTickerIsRegistered
		}
		_, exception := baselineByDesignTickers[tk]
		isBaseline := p == core.DefaultParams()
		switch {
		case isBaseline && !exception:
			t.Errorf("%s торгует дефолты ядра без именного исключения: либо калибровка не внесена, либо добавь причину в baselineByDesignTickers", tk)
		case !isBaseline && exception:
			t.Errorf("%s значится исключением «на дефолтах», но литерал отличается от core.DefaultParams() — убери исключение", tk)
		}
	}
	for tk, why := range baselineByDesignTickers {
		if !prod[tk] {
			t.Errorf("исключение %s (%s) не относится к боевой вселенной — устарело", tk, why)
		}
	}
}

func TestSBERIsNotRegistered(t *testing.T) {
	if _, ok := ParamsFor("SBER"); ok {
		t.Fatal("SBER планку не взял и в живой реестр rsi_zone не заводится")
	}
}
