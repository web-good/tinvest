package config

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменных rsi_zone конфиг грузится, стратегия выключена, торговля выключена: ошибка
// загрузки конфига оборвала бы InitApp целиком, а с ним и живые воркеры.
func TestRSIZoneConfigLoadsWithoutItsEnvVars(t *testing.T) {
	t.Setenv("RSI_ZONE_TICKERS", "")
	t.Setenv("RSI_ZONE_BUY_PCT", "")
	t.Setenv("RSI_ZONE_TRADE_ENABLED", "")
	t.Setenv("RSI_ZONE_NOTIFY_ENABLED", "")

	cfg := NewRSIZoneConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без RSI_ZONE_* = %v, want nil", err)
	}
	if cfg.Enabled() || cfg.TradeEnabled {
		t.Fatalf("без переменных cfg = %+v, want стратегия и торговля выключены", cfg)
	}
}

func TestNewRSIZoneConfig_Defaults(t *testing.T) {
	c := NewRSIZoneConfig()
	if c.Enabled() || c.TradeEnabled || c.NotifyEnabled || c.BuyPct != 5 {
		t.Fatalf("дефолты = %+v, want выключено, BuyPct 5", c)
	}
}

// RSI_ZONE_TICKERS= (пустое значение в env) даёт [""] — это не вселенная.
func TestRSIZoneEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewRSIZoneConfig()
	c.Tickers = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
	c.Tickers = []string{"AFKS"}
	if !c.Enabled() {
		t.Fatal("вселенная AFKS не включила стратегию")
	}
}

// env/prod.env и оба образца несут одну и ту же вселенную rsi_zone (прод-файл уже дважды
// молча терял тикеры rsi_pullback — см. TestRSIPullbackTickersMatchEnvFiles).
func TestRSIZoneTickersMatchAcrossEnvFiles(t *testing.T) {
	want := "AFKS,BAZA,DIAS,IVAT,UPRO,MTSS,AFLT,MSNG,BANE,SVAV,TRNFP,SMLT,TATNP,RENI,FIXR"
	for _, path := range []string{"../../env/prod.env", "../../env/prod.env.example", "../../env/local.env.example"} {
		raw, err := os.ReadFile(path) //nolint:gosec // fixed repository path
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var got string
		for _, line := range strings.Split(string(raw), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "RSI_ZONE_TICKERS="); ok {
				got = v
			}
		}
		if got != want {
			t.Errorf("%s: RSI_ZONE_TICKERS=%q, want %q", path, got, want)
		}
	}
}
