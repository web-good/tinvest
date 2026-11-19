package live

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

type SharedMarket interface {
	density.StreamSource
	density.UniverseSource
	InstrumentsByTickers(context.Context, []string) ([]density.UniverseInput, error)
	InstrumentByID(context.Context, string) (density.Instrument, error)
}
type Participant struct {
	FollowScreener    bool
	AllowEntry        func(density.Instrument, density.Signal) bool
	RunPaper          func(context.Context) error
	Account, StateDir string
	Tickers           []string
	TradeEnabled      bool
	Config            Config
	Detector          density.Config
	HistoryInterval   time.Duration
	Market            SharedMarket
	NewBroker         func([]density.Instrument) Broker
	Coordinator       *accountcoord.Coordinator
	Report            func(State) error
	Warn              func(error)
	Notify            func(context.Context, Notification) error
}

func (p *Participant) Run(ctx context.Context) (retErr error) {
	if !p.TradeEnabled {
		if p.RunPaper != nil {
			return p.RunPaper(ctx)
		}
		return nil
	}
	if p.Market == nil || p.NewBroker == nil || p.Coordinator == nil || !p.Coordinator.MatchesAccount(p.Account) {
		return fmt.Errorf("density requires the shared account coordinator and transport")
	}
	if p.FollowScreener && p.AllowEntry == nil {
		return fmt.Errorf("following density screener requires an entry admission filter")
	}
	if err := p.Config.Validate(); err != nil {
		return err
	}
	writer, err := p.Coordinator.LockDensity(ctx, p.StateDir)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, writer.Close()) }()
	store, err := OpenStore(p.StateDir, p.Account)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, store.Close()) }()
	if err = p.recover(ctx, store); err != nil {
		return err
	}
	source := selectedSharedMarket{SharedMarket: p.Market, followScreener: p.FollowScreener}
	for _, t := range p.Tickers {
		source.tickers = append(source.tickers, strings.ToUpper(strings.TrimSpace(t)))
	}
	instruments, err := density.RefreshUniverse(ctx, source, p.Detector, p.HistoryInterval)
	if err != nil {
		return err
	}
	if len(instruments) == 0 {
		return fmt.Errorf("density shared account: no instruments passed entry filters")
	}
	broker := p.NewBroker(instruments)
	if err = broker.CheckAccount(ctx); err != nil {
		return err
	}
	feed := NewFeed(p.Config)
	engine, err := NewEngine(p.Config, broker, store, feed, instruments)
	if err != nil {
		return err
	}
	if err = engine.AttachCoordinator(ctx, p.Coordinator); err != nil {
		return err
	}
	if p.Report != nil {
		if err = p.Report(engine.State()); err != nil {
			return err
		}
	}
	runner := Runner{BookOnly: p.FollowScreener, AllowEntry: p.AllowEntry, Source: p.Market, Universe: instruments, Detector: p.Detector, Feed: feed, Engine: engine, Report: p.Report, Warn: p.Warn, Notify: p.Notify, HistorySource: source, HistoryInterval: p.HistoryInterval}
	runErr := runner.Run(ctx)
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cleanupErr := engine.Shutdown(cleanupCtx)
	cleanupErr = errors.Join(cleanupErr, p.flushNotifications(engine))
	if p.Report != nil {
		cleanupErr = errors.Join(cleanupErr, p.Report(engine.State()))
	}
	return FinishError(runErr, cleanupErr)
}

func (p *Participant) recover(ctx context.Context, store *Store) error {
	state := store.State()
	if state.SharedRoot != "" && state.SharedRoot != p.Coordinator.Root() {
		return fmt.Errorf("density shared coordinator root changed")
	}
	if len(state.Releases) > 0 {
		if state.SharedRoot == "" {
			return fmt.Errorf("pending ownership release without shared binding")
		}
		for id, claim := range state.Releases {
			if err := p.Coordinator.MarkClosed(ctx, claim); err != nil {
				return err
			}
			delete(state.Releases, id)
		}
		if err := store.Save(state, "shared_ownership_closed"); err != nil {
			return err
		}
	}
	if len(state.Episodes) == 0 {
		return p.flushNotifications(&Engine{state: state, store: store})
	}
	var instruments []density.Instrument
	for id := range state.Episodes {
		i, err := p.Market.InstrumentByID(ctx, id)
		if err != nil {
			return err
		}
		instruments = append(instruments, i)
	}
	engine, err := NewEngine(p.Config, p.NewBroker(instruments), store, NewFeed(p.Config), instruments)
	if err != nil {
		return err
	}
	if err = engine.AttachCoordinator(ctx, p.Coordinator); err != nil {
		return err
	}
	recoveryCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	shutdownErr := engine.Shutdown(recoveryCtx)
	return errors.Join(shutdownErr, p.flushNotifications(engine))
}

func (p *Participant) flushNotifications(engine *Engine) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return engine.FlushNotifications(ctx, p.Notify, p.Warn)
}

type selectedSharedMarket struct {
	SharedMarket
	tickers        []string
	followScreener bool
}

func (m selectedSharedMarket) Instruments(ctx context.Context) ([]density.UniverseInput, error) {
	if m.followScreener {
		return m.SharedMarket.Instruments(ctx)
	}
	return m.InstrumentsByTickers(ctx, m.tickers)
}

func (m selectedSharedMarket) SupportCandles(ctx context.Context, id string, days int) ([]density.SupportCandle, error) {
	source, ok := m.SharedMarket.(density.SupportSource)
	if !ok {
		return nil, fmt.Errorf("density support history source is required")
	}
	return source.SupportCandles(ctx, id, days)
}

func (m selectedSharedMarket) AverageVolume(ctx context.Context, id string, from, to time.Time) (float64, error) {
	source, ok := m.SharedMarket.(density.VolumeSource)
	if !ok {
		return 0, fmt.Errorf("density volume history source is required")
	}
	return source.AverageVolume(ctx, id, from, to)
}
