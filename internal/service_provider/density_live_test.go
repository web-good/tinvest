package service_provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"tinvest/internal/config"
	densitylive "tinvest/internal/service/screener/density/live"
	marketgrpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/client/telegram"

	tgbot "github.com/go-telegram/bot"
	"google.golang.org/grpc"
)

func TestSharedDensityDisabledCreatesNoClients(t *testing.T) {
	s := &ServiceProvider{appConfig: &config.Config{DensityLive: config.NewDensityLiveConfig()}}
	got, err := s.GetDensityLiveService()
	if err != nil || got != nil {
		t.Fatalf("%v %v", got, err)
	}
	if s.client.rsiPullbackGrpcClient != nil {
		t.Fatal("disabled participant created a transport")
	}
}

type sharedClientFake struct {
	marketgrpc.GrpcClient
	connections int
}

func (f *sharedClientFake) Connection() grpc.ClientConnInterface { f.connections++; return nil }
func TestDensityDIReusesRSITransportAndCoordinator(t *testing.T) {
	old := serviceProvider
	t.Cleanup(func() { serviceProvider = old })
	cfg := &config.Config{DensityLive: config.NewDensityLiveConfig(), Density: config.NewDensityConfig(), RSIPullback: &config.RSIPullbackConfig{AccountID: "shared", Token: "shared-secret"}}
	cfg.DensityLive.Enabled = true
	cfg.DensityLive.TradeEnabled = true
	cfg.DensityLive.EntryPct = .3
	cfg.Density.ChatID = -100123
	s := &ServiceProvider{appConfig: cfg}
	serviceProvider = s
	transport := &sharedClientFake{}
	s.client.rsiPullbackGrpcClient = transport
	s.client.telegramBot = &telegram.Bot{}
	p, err := s.GetDensityLiveService()
	if err != nil {
		t.Fatal(err)
	}
	if p.Account != "shared" || p.Coordinator != s.GetAccountCoordinator() || transport.connections != 1 || !p.TradeEnabled {
		t.Fatal("participant was not bound to RSI transport and coordinator")
	}
	if p.Config.EntryPct != .3 {
		t.Fatalf("live entry percentage = %v", p.Config.EntryPct)
	}
}

func TestDensitySimulationNeedsNoRealAccountOrBroker(t *testing.T) {
	cfg := &config.Config{DensityLive: config.NewDensityLiveConfig(), Density: config.NewDensityConfig()}
	cfg.DensityLive.Enabled = true
	cfg.DensityLive.EntryPct = .3
	cfg.Density.ChatID = -100123
	s := &ServiceProvider{appConfig: cfg}
	p, err := s.GetDensityLiveService()
	if err != nil {
		t.Fatal(err)
	}
	if p.TradeEnabled || p.RunPaper == nil || p.NewBroker != nil || p.Coordinator != nil || p.Account != "" {
		t.Fatal("simulation depends on real trading")
	}
	if p.Config.EntryPct != .3 {
		t.Fatalf("simulation entry percentage = %v", p.Config.EntryPct)
	}
	if s.client.rsiPullbackGrpcClient != nil || s.client.grpcClient != nil {
		t.Fatal("factory opened account transport")
	}
}
func TestSharedDensityInheritsRSIAccountAndToken(t *testing.T) {
	cfg := &config.Config{RSIPullback: &config.RSIPullbackConfig{AccountID: "shared-rsi", Token: "shared-token"}, GrpcClient: &config.GrpcClient{TokenProd: "different-market-token"}}
	account, token, err := densityLiveAccount(cfg)
	if err != nil || account != "shared-rsi" || token != "shared-token" {
		t.Fatalf("wrong shared binding: account_matches=%v token_matches=%v err=%v", account == "shared-rsi", token == "shared-token", err)
	}
}
func TestSharedDensityNeverFallsBackToAnotherAccount(t *testing.T) {
	cfg := &config.Config{RSIPullback: &config.RSIPullbackConfig{}, Reversion: &config.ReversionConfig{AccountID: "other", Token: "other-token"}}
	if _, _, err := densityLiveAccount(cfg); err == nil {
		t.Fatal("another account silently selected")
	}
}

func TestDensityLiveTradeRequiresNotificationRecipientBeforeClients(t *testing.T) {
	cfg := &config.Config{DensityLive: config.NewDensityLiveConfig(), Density: config.NewDensityConfig()}
	cfg.DensityLive.Enabled = true
	cfg.DensityLive.TradeEnabled = true
	s := &ServiceProvider{appConfig: cfg}
	_, err := s.GetDensityLiveService()
	if err == nil || !strings.Contains(err.Error(), "DENSITY_CHAT_ID") {
		t.Fatalf("missing recipient not rejected: %v", err)
	}
	if s.client.rsiPullbackGrpcClient != nil {
		t.Fatal("created transport before validation")
	}
}

func TestDensityLiveSenderUsesConfiguredTopic(t *testing.T) {
	var got struct {
		ChatID  int64  `json:"chat_id"`
		TopicID int    `json:"message_thread_id"`
		Text    string `json:"text"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
		}
		got.ChatID, _ = strconv.ParseInt(r.FormValue("chat_id"), 10, 64)
		got.TopicID, _ = strconv.Atoi(r.FormValue("message_thread_id"))
		got.Text = r.FormValue("text")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":1,"chat":{"id":-100123,"type":"supergroup"}}}`))
	}))
	defer server.Close()
	bot, err := tgbot.New("fake-token", tgbot.WithSkipGetMe(), tgbot.WithServerURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	n := densitylive.Notification{ID: "trade:buy:1", Side: "BUY", Ticker: "SBER", TradeID: "trade", Lots: 1, Units: 10, Amount: 2500 * 1e9, Complete: true}
	err = (densityLiveSender{bot: bot, chatID: -100123, topicID: 42}).Send(context.Background(), n)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChatID != -100123 || got.TopicID != 42 || got.Text != densitylive.FormatNotification(n) {
		t.Fatalf("wrong destination or body: %+v", got)
	}
}

func TestMissingDensityJournalSelectsMode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg := &config.Config{
			DensityLive: config.NewDensityLiveConfig(),
			RSIPullback: &config.RSIPullbackConfig{AccountID: "test-account"},
		}
		cfg.DensityLive.StateDir = t.TempDir()
		cfg.DensityLive.Enabled = true
		cfg.DensityLive.TradeEnabled = enabled
		s := &ServiceProvider{appConfig: cfg}
		snapshot, err := s.GetAccountCoordinator().DensitySnapshot(context.Background())
		if err != nil || (snapshot != nil) != enabled {
			t.Fatalf("live=%v: positions=%v, err=%v", enabled, snapshot, err)
		}
	}
}

func TestSimulationKeepsSharedModeWithRealJournal(t *testing.T) {
	cfg := &config.Config{
		DensityLive: config.NewDensityLiveConfig(),
		RSIPullback: &config.RSIPullbackConfig{AccountID: "test-account"},
	}
	cfg.DensityLive.StateDir = t.TempDir()
	cfg.DensityLive.Enabled = true
	dir := filepath.Join(cfg.DensityLive.StateDir, densitylive.AccountHash(cfg.RSIPullback.AccountID))
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	frame := `{"State":{"Version":2}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(frame), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &ServiceProvider{appConfig: cfg}
	snapshot, err := s.GetAccountCoordinator().DensitySnapshot(context.Background())
	if err != nil || snapshot == nil {
		t.Fatalf("real journal requires shared accounting: positions=%v, err=%v", snapshot, err)
	}
}

func TestSimulationFailsClosedWhenBoundDensityJournalIsMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := &config.Config{
		DensityLive: config.NewDensityLiveConfig(),
		RSIPullback: &config.RSIPullbackConfig{AccountID: "test-bound-account"},
	}
	cfg.DensityLive.StateDir = t.TempDir()
	s := &ServiceProvider{appConfig: cfg}
	coord := s.GetAccountCoordinator()
	lease, err := coord.LockDensity(context.Background(), cfg.DensityLive.StateDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := coord.DensitySnapshot(context.Background()); err == nil {
		t.Fatal("a missing bound real journal switched the runner to ordinary reconciliation")
	}
}
