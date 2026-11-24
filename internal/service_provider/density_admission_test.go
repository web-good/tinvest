package service_provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"tinvest/internal/config"
	"tinvest/internal/service/screener/density"

	tgbot "github.com/go-telegram/bot"
)

func TestDensityAdmissionRequiresAlertExceptForConfiguredTickers(t *testing.T) {
	s := &ServiceProvider{}
	allow := s.densityEntryFilter([]string{" sber "})
	signal := density.Signal{InstrumentID: "extra"}
	if !allow(density.Instrument{ID: "base", Ticker: "SBER"}, signal) {
		t.Fatal("configured ticker blocked")
	}
	if allow(density.Instrument{ID: "extra", Ticker: "VTBR"}, signal) {
		t.Fatal("unannounced ticker allowed")
	}
	s.service.densityAlerts.Record(signal)
	if !allow(density.Instrument{ID: "extra", Ticker: "VTBR"}, signal) {
		t.Fatal("announced ticker blocked")
	}
	if allow(density.Instrument{ID: "other", Ticker: "OTHER"}, signal) {
		t.Fatal("unrelated ticker admitted")
	}
}

func TestDensityAlertAdmitsTickerOnlyAfterSuccessfulDelivery(t *testing.T) {
	for _, success := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "success"}[success], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if !success {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"bad request"}`))
					return
				}
				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":1,"chat":{"id":1,"type":"private"}}}`))
			}))
			defer server.Close()
			bot, err := tgbot.New("test", tgbot.WithSkipGetMe(), tgbot.WithServerURL(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			s := &ServiceProvider{}
			signal := density.Signal{InstrumentID: "extra", Since: time.Now()}
			sender := densitySender{bot: bot, chatID: 1, onSent: s.service.densityAlerts.Record}
			err = sender.Send(context.Background(), signal)
			if (err == nil) != success {
				t.Fatalf("unexpected delivery: %v", err)
			}
			if s.densityEntryFilter(nil)(density.Instrument{ID: "extra"}, signal) != success {
				t.Fatal("admission does not match successful delivery")
			}
		})
	}
}

func TestDensityFollowScreenerUsesFullUniverseAndSharedAdmission(t *testing.T) {
	cfg := &config.Config{DensityLive: config.NewDensityLiveConfig(), Density: config.NewDensityConfig()}
	cfg.DensityLive.Enabled = true
	cfg.DensityLive.FollowScreener = true
	cfg.Density.ChatID = 1
	cfg.Density.Persistence = 2 * time.Second
	cfg.Density.AlertDistanceTicks = 3
	cfg.Density.AlertM5VolumeMultiplier = .7
	s := &ServiceProvider{appConfig: cfg}
	p, err := s.GetDensityLiveService()
	if err != nil {
		t.Fatal(err)
	}
	if !p.FollowScreener || p.Detector.MaxBooks != cfg.Density.MaxBooks || p.AllowEntry == nil {
		t.Fatalf("expanded mode not wired: %+v", p)
	}
	if p.Detector.Persistence != 5*time.Minute {
		t.Fatalf("simulation density confirmation = %v, want 5m", p.Detector.Persistence)
	}
	if p.Detector.AlertDistanceTicks != 3 || p.Detector.AlertM5VolumeMultiplier != .7 {
		t.Fatalf("participant did not receive the screener thresholds: %+v", p.Detector)
	}
	i := density.Instrument{ID: "extra", Ticker: "EXTRA"}
	signal := density.Signal{InstrumentID: i.ID}
	if p.AllowEntry(i, signal) {
		t.Fatal("new ticker admitted before alert")
	}
	s.service.densityAlerts.Record(signal)
	if !p.AllowEntry(i, signal) {
		t.Fatal("sender registry not shared with participant")
	}
}
