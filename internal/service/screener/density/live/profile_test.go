package live

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestSharedDetectorFiltersActualBuy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		qty     int64
		support int64
		want    int
	}{
		{"size and support pass", 1000, 100, 1},
		{"below turnover threshold", 999, 100, 0},
		{"outside support zone", 1000, 95, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := setup(t)
			cfg := density.DefaultConfig()
			cfg.MinRubles = 1
			cfg.TurnoverFraction = .005
			cfg.Support = density.DefaultSupportConfig()
			cfg.Support.Enabled = true
			cfg.Persistence = 0
			d := density.NewDetector(cfg)
			i := x.i
			i.Turnover = 200e6
			i.ProfileAt = x.now
			i.Supports = []density.Support{{Price: density.Price{Units: tc.support}, Touches: 2}}
			d.SetInstruments([]density.Instrument{i})
			x.now = x.now.Add(time.Millisecond)
			b := density.Book{InstrumentID: i.ID, Lot: 10, Time: x.now, Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}
			for n := int64(0); n < 6; n++ {
				qty := int64(1)
				if n == 0 {
					qty = tc.qty
				}
				b.Bids = append(b.Bids, density.Level{Price: density.Price{Units: 100 - n}, Quantity: qty})
			}
			x.f.Book(b, d.Update(b, x.now), x.now)
			if err := x.e.Step(context.Background(), x.now); err != nil {
				t.Fatal(err)
			}
			if len(x.b.posts) != tc.want {
				t.Fatalf("buy requests: %d, want %d", len(x.b.posts), tc.want)
			}
		})
	}
}

type blockingHistory struct{ started chan struct{} }

func (h blockingHistory) Instruments(context.Context) ([]density.UniverseInput, error) {
	return []density.UniverseInput{{Instrument: density.Instrument{ID: "uid", Ticker: "T", Lot: 10}, Currency: "rub", Exchange: "MOEX", BuyAvailable: true}}, nil
}
func (h blockingHistory) Candles(ctx context.Context, _ string) ([]density.DailyCandle, error) {
	close(h.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestSlowHistoryRefreshDoesNotDelayExit(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	x.e.Now = time.Now
	exited := make(chan struct{}, 1)
	x.b.onPost = func() { exited <- struct{}{} }
	i := x.i
	i.ProfileAt = time.Now().Add(-density.ProfileLifetime)
	history := blockingHistory{started: make(chan struct{})}
	cfg := density.DefaultConfig()
	cfg.TurnoverFraction = .005
	r := Runner{Source: channelSource{events: make(chan []density.Event)}, Universe: []density.Instrument{i}, Detector: cfg, Feed: x.f, Engine: x.e, HistorySource: history, HistoryInterval: time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	defer func() { cancel(); <-done }()
	select {
	case <-history.started:
	case <-time.After(time.Second):
		t.Fatal("history did not start")
	}
	x.f.Gap(time.Now())
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("history refresh blocked exit")
	}
}

func TestSimulationRefusesUnfinishedRealEpisode(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	x := setup(t)
	x.step(t)
	if err := CheckSimulationState(filepath.Dir(x.s.Dir)); err == nil {
		t.Fatal("simulation hid active real episode")
	}
}

func TestSimulationRefusesLegacyRealEpisode(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	root := t.TempDir()
	dir := filepath.Join(root, "legacy")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(journalRecord{State: State{Version: 1, Episode: &Episode{Instrument: density.Instrument{ID: "x"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "events.jsonl"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err = CheckSimulationState(root); err == nil {
		t.Fatal("legacy position was hidden")
	}
}

func TestSimulationChecksAccountBindingAfterRootChange(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	x := setup(t)
	x.step(t)
	if err := CheckSimulationState(t.TempDir()); err == nil {
		t.Fatal("changed state root hid real episode")
	}
}

func TestPrepareRejectsExpiredProfileAfterAllocation(t *testing.T) {
	x := setup(t)
	candidates := x.f.Candidates(x.now)
	if len(candidates) != 1 {
		t.Fatal("missing fixture candidate")
	}
	s := candidates[0].Signal
	s.ProfileUntil = x.now
	b := candidates[0].Book
	b.Time = x.now
	x.f.Book(b, []density.Signal{s}, x.now)
	if _, err := Plan(x.f.cfg, x.i, s, b, x.now); err == nil {
		t.Fatal("expired profile permitted entry")
	}
}

func TestProfileRefreshOnlyClearsEntrySignals(t *testing.T) {
	x := setup(t)
	x.step(t)
	before := len(x.e.State().Episodes)
	x.f.ClearSignals()
	if len(x.f.Candidates(x.now)) != 0 {
		t.Fatal("old entry signals retained")
	}
	if len(x.e.State().Episodes) != before || x.f.Reason(x.now.Add(time.Millisecond)) != "" {
		t.Fatal("history update interrupted position watcher")
	}
}
