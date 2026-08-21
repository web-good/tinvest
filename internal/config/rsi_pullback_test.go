package config

import (
	"context"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Незаданные RSI_PULLBACK_ACCOUNT_ID/TOKEN НЕ должны валить загрузку конфига. С тегом
// `required` confita (v0.10.0, config.go:234 — `f.Required && isZero(f.Value)`) возвращает
// ошибку и на отсутствующей переменной, и на пустой; ошибка загрузки конфига обрывает
// InitApp, то есть приложение не поднимается ЦЕЛИКОМ — вместе с обоими воркерами живой
// reversion, которые ведут реальные позиции. Новая стратегия не имеет права ронять старую
// на этапе чтения конфига: без своих переменных её раннер просто не поднимается (Ready()).
func TestRSIPullbackConfigLoadsWithoutItsEnvVars(t *testing.T) {
	t.Setenv("RSI_PULLBACK_ACCOUNT_ID", "")
	t.Setenv("RSI_PULLBACK_TOKEN", "")

	cfg := NewRSIPullbackConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без RSI_PULLBACK_ACCOUNT_ID/TOKEN = %v, want nil", err)
	}
}

// Ready — гейт запуска раннера: без счёта и токена запускать нечего, а пустой токен в
// gRPC-клиенте дал бы поток отказов авторизации вместо внятного отказа на старте.
func TestRSIPullbackConfigReadyNeedsAccountAndToken(t *testing.T) {
	cases := []struct {
		name      string
		accountID string
		token     string
		want      bool
	}{
		{"оба заданы", "acc-1", "tok-1", true},
		{"нет счёта", "", "tok-1", false},
		{"нет токена", "acc-1", "", false},
		{"нет ничего", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewRSIPullbackConfig()
			c.AccountID, c.Token = tc.accountID, tc.token
			if got := c.Ready(); got != tc.want {
				t.Fatalf("Ready() = %v, want %v (счёт %q, токен %q)", got, tc.want, tc.accountID, tc.token)
			}
		})
	}
}

func TestNewRSIPullbackConfig_Defaults(t *testing.T) {
	c := NewRSIPullbackConfig()
	want := []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP", "IVAT", "SVAV", "SIBN"}
	if len(c.Tickers) != len(want) {
		t.Fatalf("default Tickers = %v, want %v", c.Tickers, want)
	}
	for i, w := range want {
		if c.Tickers[i] != w {
			t.Fatalf("default Tickers = %v, want %v", c.Tickers, want)
		}
	}
	if c.BuyPct != 5 {
		t.Fatalf("default BuyPct = %v, want 5", c.BuyPct)
	}
	if c.Schedule != "1,31 6-23 * * *" {
		t.Fatalf("default Schedule = %q, want \"1,31 6-23 * * *\"", c.Schedule)
	}
}

// Забытый флаг никогда не должен выставить реальный ордер.
func TestNewRSIPullbackConfig_TradeDisabledByDefault(t *testing.T) {
	if NewRSIPullbackConfig().TradeEnabled {
		t.Fatal("TradeEnabled default = true, want false")
	}
}

func TestNewRSIPullbackConfig_TokenHasNoDefault(t *testing.T) {
	if tok := NewRSIPullbackConfig().Token; tok != "" {
		t.Fatalf("default Token = %q, want empty (must come from RSI_PULLBACK_TOKEN env)", tok)
	}
}
