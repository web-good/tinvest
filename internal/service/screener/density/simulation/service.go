package simulation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
)

// Market exposes no account, broker or order placement operations.
type Market interface {
	density.UniverseSource
	density.StreamSource
	InstrumentsByTickers(context.Context, []string) ([]density.UniverseInput, error)
}

type Service struct {
	FollowScreener  bool
	AllowEntry      func(density.Instrument, density.Signal) bool
	Market          Market
	Config          paper.Config
	Detector        density.Config
	Tickers         []string
	Root            string
	HistoryInterval time.Duration
	Notify          func(context.Context, string) error
	Warn            func(error)
}

func (s *Service) Run(parent context.Context) (retErr error) {
	if s.Market == nil || s.Notify == nil || s.Root == "" || len(s.Tickers) < 1 || len(s.Tickers) > 10 {
		return fmt.Errorf("simulation requires market data, recipient, state directory and 1..10 tickers")
	}
	if s.FollowScreener {
		if s.AllowEntry == nil {
			return fmt.Errorf("following density screener requires an entry admission filter")
		}
		s.Config.BookExecution = true
	}
	if err := s.Config.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	root := filepath.Join(s.Root, "paper")
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(root, "runner.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, lock.Close()) }()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another density simulation uses this directory: %w", err)
	}
	outbox, err := OpenOutbox(filepath.Join(root, "outbox.json"))
	if err != nil {
		return err
	}
	deliveryDone := make(chan struct{})
	go func() { defer close(deliveryDone); outbox.Run(ctx, s.Notify, s.Warn) }()
	defer func() { cancel(); <-deliveryDone }()
	source := selectedMarket{Market: s.Market, tickers: s.Tickers, followScreener: s.FollowScreener}
	instruments, err := density.RefreshUniverse(ctx, source, s.Detector, s.HistoryInterval)
	if err != nil {
		return err
	}
	if len(instruments) == 0 {
		return fmt.Errorf("density simulation: no instruments passed entry filters")
	}
	journal, err := paper.NewJournal(root)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, journal.Close()) }()
	record := func(r paper.Record) error {
		return recordEvent(journal, outbox, r)
	}
	engine, err := paper.New(s.Config, instruments, record)
	if err != nil {
		return err
	}
	if err = journal.Session(s.Config, s.Detector, instruments); err != nil {
		return err
	}
	runner := paper.Runner{AllowEntry: s.AllowEntry, Source: s.Market, Universe: instruments, Engine: engine, Detector: s.Detector, HistorySource: source, HistoryInterval: s.HistoryInterval, Report: journal.WriteSummary}
	runErr := runner.Run(ctx)
	engine.Stop(time.Now())
	summaryErr := journal.WriteSummary(engine.Summary(time.Now()))
	if runErr == context.Canceled || runErr == context.DeadlineExceeded {
		runErr = nil
	}
	return errors.Join(runErr, engine.Err(), summaryErr)
}

// The outbox commits the closed trade and its alert together before the
// diagnostic session log: a crash between writes cannot lose a completed-trade alert.
func recordEvent(journal *paper.Journal, outbox *Outbox, r paper.Record) error {
	if err := outbox.Record(r); err != nil {
		return err
	}
	return journal.Record(r)
}

type selectedMarket struct {
	Market
	tickers        []string
	followScreener bool
}

func (s selectedMarket) Instruments(ctx context.Context) ([]density.UniverseInput, error) {
	if s.followScreener {
		return s.Market.Instruments(ctx)
	}
	return s.InstrumentsByTickers(ctx, s.tickers)
}
func (s selectedMarket) SupportCandles(ctx context.Context, id string, days int) ([]density.SupportCandle, error) {
	source, ok := s.Market.(density.SupportSource)
	if !ok {
		return nil, fmt.Errorf("simulation support history source is required")
	}
	return source.SupportCandles(ctx, id, days)
}
