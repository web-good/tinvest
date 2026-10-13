// Command dropscreen ranks the tradable RUB share universe by how well the drop_bounce strategy
// fits each ticker on one timeframe. Like cmd/zonescreen it replays the real strategy engine over a
// fixed grid (16 configurations) and ranks by the median profit factor under STRESSED costs
// (double commission plus two price ticks). The report ends with a shortlist line — the queue for
// the drop-bounce-calibrate skill. All gRPC/file I/O lives here; the scoring is pure
// (internal/service/backtest/drop_screen*.go). See docs/drop_bounce/screener.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

func main() {
	var (
		intervalS     = flag.String("interval", "Hour1", "timeframe: Hour1 (drop_bounce runs on 1h only)")
		months        = flag.Int("months", 36, "lookback period in months")
		holdoutMonths = flag.Int("holdout-months", 6, "trailing months held out of the ranking window")
		minTurnoverM  = flag.Float64("min-turnover", 50, "gate: minimum mean daily turnover in millions of RUB")
		maxTickPct    = flag.Float64("max-tick-pct", 0.3, "gate: maximum two price ticks as a percentage of the last close (0 = off)")
		minTrades     = flag.Float64("min-trades", 20, "gate: minimum median train trades across the grid")
		topN          = flag.Int("top", 50, "ranking rows in the report (0 = all)")
		shortlist     = flag.Int("shortlist", 10, "tickers in the DROP_BOUNCE_CANDIDATES line (0 = whole ranking)")
		workers       = flag.Int("workers", 1, "concurrent tickers (default 1: single-core rule)")
		pfCap         = flag.Float64("pf-cap", 10, "cap on profit factor for ranking (0 = no cap)")
		cash          = flag.Float64("cash", 100000, "starting mock cash")
		fraction      = flag.Float64("fraction", 1.0, "fraction of cash per entry")
		commission    = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		tickersCSV    = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		outDir        = flag.String("out", "reports/drop_screen", "report output directory")
		refresh       = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		pause         = flag.Duration("pause", 0, "idle time after each ticker, e.g. 500ms or 2s: trades wall-clock for a cooler CPU")
	)
	flag.Parse()
	logger.Init()

	interval, err := parseInterval(*intervalS)
	if err != nil {
		log.Fatalf("dropscreen: %v", err)
	}
	opts := svc.DefaultScreenOpts()
	opts.PFCap = *pfCap
	opts.Cash = *cash
	opts.Fraction = *fraction
	opts.Commission = *commission

	if err := run(context.Background(), runCfg{
		interval: interval, months: *months, holdoutMonths: *holdoutMonths, topN: *topN, shortlist: *shortlist,
		workers: *workers, minTurnoverM: *minTurnoverM, maxTickPct: *maxTickPct, minTrades: *minTrades,
		tickers: screenrun.SplitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, pause: *pause, opts: opts,
	}); err != nil {
		log.Fatalf("dropscreen: %v", err)
	}
}

type runCfg struct {
	interval                               enum.Interval
	months, holdoutMonths, topN, shortlist int
	workers                                int
	minTurnoverM, maxTickPct, minTrades    float64
	tickers                                []string
	outDir                                 string
	refresh                                bool
	pause                                  time.Duration
	opts                                   svc.ScreenOpts
}

// parseInterval accepts the one timeframe drop_bounce runs on.
func parseInterval(s string) (enum.Interval, error) {
	if strings.EqualFold(s, enum.Hour1.String()) {
		return enum.Hour1, nil
	}
	return 0, fmt.Errorf("unknown -interval %q (want Hour1: drop_bounce runs on 1h only)", s)
}

// pregated reports whether the daily-candle turnover fails the gate; equality passes.
func pregated(turnoverM, minTurnoverM float64) bool {
	return turnoverM < minTurnoverM
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	if _, err := svc.DropGrid(cfg.interval); err != nil {
		return err
	}
	if cfg.holdoutMonths >= cfg.months {
		return fmt.Errorf("-holdout-months (%d) must be smaller than -months (%d)", cfg.holdoutMonths, cfg.months)
	}
	if cfg.workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	if cfg.maxTickPct < 0 {
		return fmt.Errorf("-max-tick-pct must be >= 0 (0 disables the gate)")
	}
	if cfg.minTrades < 0 {
		return fmt.Errorf("-min-trades must be >= 0")
	}
	if cfg.shortlist < 0 {
		return fmt.Errorf("-shortlist must be >= 0 (0 = whole ranking)")
	}
	return nil
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	if workers, capped := screenrun.EffectiveWorkers(cfg.workers, cfg.refresh); capped {
		fmt.Printf("dropscreen: -refresh forces -workers %d -> %d to avoid punching holes in the local candle cache (see screenrun.RefreshWorkerCap)\n",
			cfg.workers, workers)
		cfg.workers = workers
	}
	token, err := screenrun.LoadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(screenrun.APIAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	universe, err := screenrun.LoadUniverse(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	// A year of lead-in warms the daily ATR, exactly as cmd/backtest does.
	dailyFrom := from.AddDate(-1, 0, 0)
	split := to.AddDate(0, -cfg.holdoutMonths, 0)

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)
	grid, err := svc.DropGrid(cfg.interval)
	if err != nil {
		return err
	}

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.DropRow
	var done, skipped int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u screenrun.ShareInfo) {
			defer wg.Done()
			defer sem.Release()

			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("dropscreen %s: skip (load daily: %v)\n", u.Ticker, err)
				return
			}

			// Turnover is checked on the cheap daily series first: a ticker below the gate would be
			// rejected after the full intraday download and 16-configuration replay anyway.
			if pre := svc.DropPregateRow(u.Ticker, u.Name, daily, from, to, u.Lot, u.MinPriceIncrement); pregated(pre.TurnoverM, cfg.minTurnoverM) {
				n := atomic.AddInt32(&done, 1)
				fmt.Printf("dropscreen [%d/%d] %s: pre-gated (turnover %.0fM < %.0fM by daily candles)\n",
					n, len(universe), u.Ticker, pre.TurnoverM, cfg.minTurnoverM)
				mu.Lock()
				rows = append(rows, pre)
				mu.Unlock()
				screenrun.PauseAfterTicker(ctx, cfg.pause)
				return
			}

			bars, err := provider.Load(ctx, u.Ticker, u.ID, cfg.interval, from, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("dropscreen %s: skip (load %s: %v)\n", u.Ticker, cfg.interval, err)
				return
			}

			row := svc.ScreenDropTicker(svc.DropTickerInput{
				Ticker: u.Ticker, Name: u.Name, Interval: cfg.interval, Bars: bars, Daily: daily,
				Lot: u.Lot, MinPriceIncrement: u.MinPriceIncrement,
			}, grid, split, cfg.opts)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("dropscreen [%d/%d] %s: PFmed=%.2f PFmed×2=%.2f trades=%.0f tick=%.2f%% turnover=%.0fM atr=%.2f%% bars=%d\n",
				n, len(universe), u.Ticker, row.PFMed, row.PFMedStress, row.TradesMed, row.TickPct, row.TurnoverM, row.DailyATRPct, row.Bars)

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()

			// Held inside the semaphore slot on purpose: releasing first would let the next
			// ticker start while this one idles, and the pool would stay just as hot.
			screenrun.PauseAfterTicker(ctx, cfg.pause)
		}(u)
	}
	wg.Wait()

	gates := svc.DropGates{MinTurnoverM: cfg.minTurnoverM, MaxTickPct: cfg.maxTickPct, MinTrades: cfg.minTrades}
	ranked, noSignals, tickRejected, rejected := svc.FilterAndRankDrop(rows, gates)
	meta := svc.DropScreenMeta{
		Interval: cfg.interval, Months: cfg.months, HoldoutMonths: cfg.holdoutMonths, TopN: cfg.topN,
		Shortlist: cfg.shortlist, Split: split, Gates: gates, PFCap: cfg.opts.PFCap,
		Commission: cfg.opts.Commission, Cash: cfg.opts.Cash, Scanned: len(universe), Skipped: int(skipped),
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("drop_screen_%s_%s.md", cfg.interval, time.Now().Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(svc.RenderDropScreenMarkdown(ranked, noSignals, tickRejected, rejected, meta)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("dropscreen report: %s (scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d)\n",
		path, len(universe), len(ranked), len(noSignals), len(tickRejected), len(rejected), skipped)
	fmt.Printf("%s=%s\n", svc.DropShortlistKey, strings.Join(svc.DropShortlist(ranked, cfg.shortlist), ","))
	return nil
}
