// density-paper is a market-data-only forward test. It never creates an order client.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
	"tinvest/internal/config"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
	marketgrpc "tinvest/pkg/client/grpc"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "density-paper:", err)
		os.Exit(1)
	}
}
func run(ctx context.Context, args []string, out io.Writer) (retErr error) {
	pc := paper.DefaultConfig()
	dc := config.NewDensityConfig()
	fs := flag.NewFlagSet("density-paper", flag.ContinueOnError)
	fs.SetOutput(out)
	root := fs.String("out", "reports/density-paper", "root directory for uniquely named sessions")
	envFile := fs.String("env-file", "", "optional dotenv file; existing environment takes precedence")
	address := fs.String("address", "invest-public-api.tinkoff.ru:443", "TLS market data API endpoint")
	tlsClassic := fs.Bool("tls-classic", false, "use X25519/P-256 key exchange for TLS compatibility with some network routes")
	maxInstruments := fs.Int("instruments", 10, "maximum selected instruments (1..10)")
	tickers := fs.String("tickers", "", "optional comma-separated ticker allowlist, e.g. T,SBER")
	duration := fs.Duration("duration", 0, "stop after this duration; 0 means until Ctrl+C")
	fs.Float64Var(&pc.InitialCash, "cash", pc.InitialCash, "initial virtual RUB balance")
	fs.Float64Var(&pc.PositionRubles, "position-rubles", pc.PositionRubles, "maximum allocation per entry, including entry commission")
	fs.Float64Var(&pc.Commission, "commission", pc.Commission, "commission per side as fraction: 0.0005 = 0.05%")
	fs.IntVar(&pc.MaxPositions, "max-positions", pc.MaxPositions, "maximum concurrent pending/held instruments")
	fs.Int64Var(&pc.EntryTicks, "entry-ticks", pc.EntryTicks, "entry above density in instrument price steps")
	fs.Int64Var(&pc.StopTicks, "stop-ticks", pc.StopTicks, "stop below original density in price steps")
	fs.Float64Var(&pc.RewardRisk, "rr", pc.RewardRisk, "take distance / entry-to-stop distance")
	fs.Float64Var(&pc.MinRemaining, "min-remaining", pc.MinRemaining, "minimum retained density fraction before cancel/exit")
	fs.DurationVar(&pc.Latency, "latency", pc.Latency, "simulated order activation/exit latency")
	fs.DurationVar(&pc.EntryTTL, "entry-ttl", pc.EntryTTL, "pending entry lifetime")
	fs.DurationVar(&pc.MaxHold, "max-hold", pc.MaxHold, "maximum holding time")
	fs.DurationVar(&pc.Cooldown, "cooldown", pc.Cooldown, "per-instrument cooldown after cancellation/close")
	// Explicit flags override DENSITY_* environment values.
	minRubles := fs.Float64("min-rubles", dc.MinRubles, "minimum density RUB (otherwise DENSITY_MIN_RUBLES)")
	minMultiple := fs.Float64("min-multiple", dc.MinMultiple, "minimum median multiple (otherwise DENSITY_MIN_MULTIPLE)")
	maxDistance := fs.Float64("max-distance", dc.MaxDistance, "relative distance: 0.01 = 1% (otherwise DENSITY_MAX_DISTANCE)")
	persistence := fs.Duration("persistence", dc.Persistence, "density confirmation time (otherwise DENSITY_PERSISTENCE)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if err := pc.Validate(); err != nil {
		return err
	}
	if *maxInstruments < 1 || *maxInstruments > 10 || *duration < 0 {
		return fmt.Errorf("instruments must be 1..10 and duration nonnegative")
	}
	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			return fmt.Errorf("dotenv: %w", err)
		}
	}
	if err := confita.NewLoader(env.NewBackend()).Load(ctx, dc); err != nil {
		return fmt.Errorf("density environment: %w", err)
	}
	dc.Enabled = false // Telegram chat/account settings do not enable anything in this executable.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "min-rubles":
			dc.MinRubles = *minRubles
		case "min-multiple":
			dc.MinMultiple = *minMultiple
		case "max-distance":
			dc.MaxDistance = *maxDistance
		case "persistence":
			dc.Persistence = *persistence
		}
	})
	if err := dc.Validate(); err != nil {
		return err
	}
	pc.Freshness = dc.Freshness
	token := os.Getenv("T_BANK")
	if token == "" {
		return fmt.Errorf("T_BANK is required for read-only market data; set environment or -env-file")
	}
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}
	conn, err := grpc.NewClient(*address, grpc.WithTransportCredentials(credentials.NewTLS(marketTLS(*tlsClassic))))
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, conn.Close()) }()
	market := marketgrpc.NewDensityMarketData(conn, token)
	detector := density.DefaultConfig()
	detector.MinRubles = dc.MinRubles
	detector.TurnoverFraction = dc.TurnoverFraction
	detector.Support.Enabled = dc.SupportEnabled
	detector.Support.LookbackDays = dc.SupportLookbackDays
	detector.Support.PivotBars = dc.SupportPivotBars
	detector.Support.MinTouches = dc.SupportMinTouches
	detector.Support.MinSeparation = dc.SupportMinSeparation
	detector.Support.BounceFraction = dc.SupportBounceFraction
	detector.Support.BounceBars = dc.SupportBounceBars
	detector.Support.Tolerance = dc.SupportTolerance
	detector.MinMultiple = dc.MinMultiple
	detector.MaxDistance = dc.MaxDistance
	detector.MinNeighbors = dc.MinNeighbors
	detector.MaxNeighbors = dc.MaxNeighbors
	detector.Persistence = dc.Persistence
	detector.Freshness = dc.Freshness
	detector.MinTurnover = dc.MinTurnover
	detector.MaxLotPrice = dc.MaxLotPrice
	detector.BookDepth = int32(dc.BookDepth)
	detector.MaxBooks = min(*maxInstruments, dc.MaxBooks)
	source := filteredSource{UniverseSource: market, tickers: make(map[string]bool)}
	for _, t := range strings.Split(*tickers, ",") {
		if t = strings.TrimSpace(strings.ToUpper(t)); t != "" {
			source.tickers[t] = true
		}
	}
	if _, err := fmt.Fprintln(out, "PAPER: loading market data universe; no real orders. History requests may take several minutes."); err != nil {
		return err
	}
	universe, err := density.RefreshUniverse(ctx, source, detector, dc.HistoryInterval)
	if err != nil {
		return err
	}
	if len(universe) == 0 {
		return fmt.Errorf("no instruments passed liquidity, lot-price and ticker filters")
	}
	j, err := paper.NewJournal(*root)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, j.Close()) }()
	e, err := paper.New(pc, universe, j.Record)
	if err != nil {
		return err
	}
	if err = j.Session(pc, detector, universe); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "PAPER: %d instruments; session %s\n", len(universe), j.Dir); err != nil {
		return err
	}
	report := func(s paper.Summary) error {
		if err := j.WriteSummary(s); err != nil {
			return err
		}
		unrealized := "unavailable"
		if s.UnrealizedPnL != nil {
			unrealized = fmt.Sprintf("%+.2f", *s.UnrealizedPnL)
		}
		_, err := fmt.Fprintf(out, "PAPER closed=%d pending=%d open=%d realized=%+.2f RUB unrealized~=%s RUB fees=%.2f RUB data_complete=%t valuation_fresh=%t\n", s.ClosedTrades, s.PendingOrders, s.OpenPositions, s.RealizedPnL, unrealized, s.Fees, s.DataComplete, s.ValuationFresh)
		return err
	}
	if err = report(e.Summary(time.Now())); err != nil {
		return err
	}
	runner := paper.Runner{Source: market, Universe: universe, Engine: e, Detector: detector, Report: report, HistorySource: source, HistoryInterval: dc.HistoryInterval}
	runErr := runner.Run(ctx)
	e.Stop(time.Now())
	summaryErr := report(e.Summary(time.Now()))
	if errors.Is(runErr, context.DeadlineExceeded) && ctx.Err() != nil {
		runErr = nil
	}
	return finishError(runErr, e.Err(), summaryErr)
}

type tickerUniverseSource interface {
	density.UniverseSource
	InstrumentsByTickers(context.Context, []string) ([]density.UniverseInput, error)
}

type filteredSource struct {
	UniverseSource tickerUniverseSource
	tickers        map[string]bool
}

func (s filteredSource) Instruments(ctx context.Context) ([]density.UniverseInput, error) {
	if len(s.tickers) == 0 {
		return s.UniverseSource.Instruments(ctx)
	}
	tickers := make([]string, 0, len(s.tickers))
	for ticker := range s.tickers {
		tickers = append(tickers, ticker)
	}
	sort.Strings(tickers)
	return s.UniverseSource.InstrumentsByTickers(ctx, tickers)
}

func (s filteredSource) Candles(ctx context.Context, id string) ([]density.DailyCandle, error) {
	return s.UniverseSource.Candles(ctx, id)
}

func (s filteredSource) SupportCandles(ctx context.Context, id string, days int) ([]density.SupportCandle, error) {
	source, ok := s.UniverseSource.(density.SupportSource)
	if !ok {
		return nil, fmt.Errorf("density support history source is required")
	}
	return source.SupportCandles(ctx, id, days)
}

func marketTLS(classic bool) *tls.Config {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if classic {
		cfg.CurvePreferences = []tls.CurveID{tls.X25519, tls.CurveP256}
	}
	return cfg
}

// Normalize expected interruption before combining it with persistence failures.
func finishError(runErr, journalErr, summaryErr error) error {
	if errors.Is(runErr, context.Canceled) {
		runErr = nil
	}
	return errors.Join(runErr, journalErr, summaryErr)
}
