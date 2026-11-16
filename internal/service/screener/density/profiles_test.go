package density

import (
	"context"
	"errors"
	"testing"
	"time"
)

type supportSource struct {
	partialSource
	err  error
	days int
}

func (s *supportSource) SupportCandles(_ context.Context, _ string, days int) ([]SupportCandle, error) {
	s.days = days
	c, _ := supportHistory()
	return c, s.err
}

func TestRefreshProfilesLoadsSupportAndPreservesFailure(t *testing.T) {
	s := &supportSource{}
	cfg := DefaultConfig()
	cfg.Support = DefaultSupportConfig()
	cfg.Support.Enabled = true
	now := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
	input := []Instrument{{ID: "x", Turnover: 2e9}}
	got, err := RefreshProfiles(context.Background(), s, input, cfg, time.Millisecond, now)
	if err != nil || len(got) != 1 || len(got[0].Supports) != 1 || !got[0].ProfileAt.Equal(now) || s.days != 60 {
		t.Fatalf("profiles %+v err %v", got, err)
	}
	if !input[0].ProfileAt.IsZero() {
		t.Fatal("mutated original profile")
	}
	s.err = errors.New("history unavailable")
	if got, err = RefreshProfiles(context.Background(), s, input, cfg, time.Millisecond, now); err == nil || got != nil {
		t.Fatalf("partial result %+v %v", got, err)
	}
	if _, err = RefreshProfiles(context.Background(), partialSource{}, input, cfg, time.Millisecond, now); err == nil {
		t.Fatal("missing support source accepted")
	}
}

func TestRefreshUniverseInstallsHistoryProfiles(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Support = DefaultSupportConfig()
	cfg.Support.Enabled = true
	got, err := RefreshUniverse(context.Background(), &supportSource{}, cfg, time.Millisecond)
	if err != nil || len(got) != 1 || got[0].ProfileAt.IsZero() {
		t.Fatalf("profiles %+v err %v", got, err)
	}
}
