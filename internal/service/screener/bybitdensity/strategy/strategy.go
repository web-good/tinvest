// Package strategy detects persistent Bybit order book levels and simulates limit entries.
package strategy

import (
	"time"

	"tinvest/internal/service/screener/bounce"
	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/volume"
)

type Config struct {
	RequireInPlay                     bool
	RequirePreviousHigh               bool
	ZoneAlerts                        bool
	ZoneWidthUSDT                     float64
	SpotVolumeMultiplier              float64
	SpotZoneFilter                    ZoneFilter
	AlertDistanceTicks                int64
	AlertM5VolumeMultiplier           float64
	FixedLimitEntry                   bool
	CandleEntries                     bool
	RequireAverageVolume              bool
	MinNotional, MinMultiple          float64
	Persistence, Freshness            time.Duration
	PaperNotional, SpotFee, LinearFee float64
	StopTicks, EntryTicks             int64
	RewardRisk, MinTarget             float64
}

func DefaultConfig() Config {
	return Config{
		AlertDistanceTicks:      2,
		AlertM5VolumeMultiplier: .5,
		MinNotional:             100000, MinMultiple: 5,
		Persistence: time.Minute, Freshness: 3 * time.Second,
		PaperNotional: 250, SpotFee: .001, LinearFee: .00055,
		StopTicks: 2, EntryTicks: 1, RewardRisk: 2, MinTarget: .003,
	}
}

type priceLevel struct {
	side  model.Side
	ticks int64
}

type Strategy struct {
	inPlay                   InPlayWindow
	previousHigh             int64
	highUntil                time.Time
	zoneWidth                int64
	bounce                   bounce.Watch[model.Signal]
	averageVolume            volume.Average
	candidates               []model.Signal
	alertCandidates          []model.Signal
	instrument               model.Instrument
	cfg                      Config
	book                     *model.Book
	needSnapshot, tradeReady bool
	episodes                 map[priceLevel]*episode
	alertEpisodes            map[priceLevel]*episode
	sent                     map[priceLevel]time.Time
	nextCleanup              time.Time
	trades                   tradeIDs
	pending                  *setup
	position                 *position
	executor                 Executor
	orderID                  uint64
	outstanding              *submittedOrder
}

func New(instrument model.Instrument, cfg Config) *Strategy {
	return NewWithExecutor(instrument, cfg, &paperExecutor{instrument: instrument, spent: make(map[priceLevel]float64)})
}

// NewWithExecutor routes decisions for one instrument to the given executor.
func NewWithExecutor(instrument model.Instrument, cfg Config, executor Executor) *Strategy {
	return &Strategy{
		zoneWidth:     max(1, instrument.Ticks(cfg.ZoneWidthUSDT)),
		instrument:    instrument,
		cfg:           cfg,
		needSnapshot:  true,
		episodes:      make(map[priceLevel]*episode),
		alertEpisodes: make(map[priceLevel]*episode),
		sent:          make(map[priceLevel]time.Time),
		executor:      executor,
	}
}

func (s *Strategy) usesSpotZones() bool {
	return s.cfg.SpotZoneFilter.Enabled() || s.cfg.SpotVolumeMultiplier > 0
}

func (s *Strategy) usesZoneAlerts() bool {
	return s.cfg.ZoneAlerts || s.cfg.SpotZoneFilter.Enabled()
}

func (s *Strategy) SetPreviousHigh(ticks int64, until time.Time) {
	s.previousHigh, s.highUntil = ticks, until
	s.pending = nil
}

func (s *Strategy) abovePreviousHigh(now time.Time) bool {
	return !s.cfg.RequirePreviousHigh || (s.previousHigh > 0 && now.Before(s.highUntil) && s.book.Bids[0].Ticks > s.previousHigh)
}

// SetAverageVolume replaces the reference and restarts entry confirmations.
func (s *Strategy) SetAverageVolume(average volume.Average) {
	s.averageVolume = average
	if s.cfg.SpotZoneFilter.Enabled() {
		return
	}
	clear(s.episodes)
	if !s.cfg.ZoneAlerts || s.cfg.SpotVolumeMultiplier > 0 {
		clear(s.alertEpisodes)
	}
	s.pending = nil
}

func (s *Strategy) SetInPlayWindow(window InPlayWindow, now time.Time) {
	if !s.inPlay.allows(now) || !window.allows(now) {
		s.clearEntries()
	}
	s.inPlay = window
}

func (s *Strategy) inPlayAllows(now time.Time) bool {
	return !s.cfg.RequireInPlay || s.inPlay.allows(now)
}

// Expire ends observation and cancels any pending entry on loss of freshness.
func (s *Strategy) Expire(now time.Time) {
	if s.book != nil && !s.fresh(s.book, now) {
		s.clearObservation()
	}
	if s.cfg.RequireInPlay && s.inPlay.Active && !now.Before(s.inPlay.From.Add(InPlayInterval)) {
		s.clearEntries()
		s.inPlay.Active = false
	}
	if !now.Before(s.nextCleanup) {
		for level, until := range s.sent {
			if !now.Before(until) {
				delete(s.sent, level)
			}
		}
		s.nextCleanup = now.Add(time.Minute)
	}
}

func (s *Strategy) clearObservation() {
	s.clearEntries()
	s.candidates = s.candidates[:0]
	s.alertCandidates = s.alertCandidates[:0]
	s.book = nil
}

func (s *Strategy) clearEntries() {
	s.bounce = bounce.Watch[model.Signal]{}
	clear(s.episodes)
	clear(s.alertEpisodes)
	s.pending = nil
}

func (s *Strategy) fresh(b *model.Book, now time.Time) bool {
	return !b.At.IsZero() && !b.Received.IsZero() && now.Sub(b.At).Abs() <= s.cfg.Freshness && now.Sub(b.Received).Abs() <= s.cfg.Freshness
}

func (s *Strategy) Apply(e model.Event, now time.Time) ([]model.Signal, []model.Fill) {
	s.Expire(now)
	if e.Reset {
		s.clearObservation()
		s.needSnapshot = true
		s.tradeReady = false
	}
	if e.TradeReady {
		s.tradeReady = true
	}
	var signals []model.Signal
	if b := e.Book; b != nil && (!s.needSnapshot || b.Snapshot) {
		if len(b.Bids) == 0 || len(b.Asks) == 0 || b.Bids[0].Ticks >= b.Asks[0].Ticks {
			s.clearObservation()
			s.needSnapshot = true
			return nil, nil
		}
		s.needSnapshot = false
		s.executor.ObserveBook(b)
		if !s.fresh(b, now) {
			s.clearObservation()
			return nil, nil
		}
		if b.Snapshot || s.book != nil && (b.At.Sub(s.book.At) > s.cfg.Freshness || b.Received.Sub(s.book.Received) > s.cfg.Freshness) {
			s.clearObservation()
		}
		s.book = b
		s.findCandidates(b, now)
		if s.tradeReady && s.inPlayAllows(now) && s.inPlayAllows(b.At) {
			s.observe(b, now, s.episodes, s.candidates, nil)
			if s.usesZoneAlerts() {
				signals = s.observeZones(b, now)
			} else {
				signals = s.observe(b, now, s.alertEpisodes, s.alertCandidates, s.sent)
			}
			if s.cfg.CandleEntries {
				s.observeBounceBook(b)
			} else if s.position == nil && s.pending == nil && s.outstanding == nil && s.abovePreviousHigh(now) {
				s.prepareEntry()
			}
		}
	}
	for _, trade := range e.Trades {
		s.observeTrade(trade)
		if s.cfg.RequirePreviousHigh && s.pending != nil && s.book != nil && !trade.At.Before(s.book.At) && trade.Ticks <= s.previousHigh {
			s.pending = nil
		}
	}
	if s.cfg.CandleEntries && s.book != nil && s.inPlayAllows(now) {
		if c := e.Candle; c != nil && c.End.Equal(c.Start.Add(time.Minute)) {
			s.bounce.Candle(c.Start, c.Low, c.Close, now)
		}
		if (e.Book != nil || e.Candle != nil) && s.position == nil && s.outstanding == nil {
			s.prepareBounce(now)
		}
	}
	if s.book == nil || (e.Book == nil && s.pending == nil) {
		return signals, nil
	}
	return signals, s.execute(now)
}
