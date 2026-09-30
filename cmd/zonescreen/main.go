// Command zonescreen ranks the tradable RUB share universe by how well the rsi_zone
// strategy fits each ticker. Like cmd/pullscreen it replays the real strategy engine
// over a fixed 24-configuration grid, but it ranks by the median profit factor under
// STRESSED costs (double commission plus two price ticks) and adds a detailed
// breakdown per ticker: half-year stability, exit profile and trend regime. All
// gRPC/file I/O lives here; the scoring is pure (internal/service/backtest/zone_screen*.go).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		months        = flag.Int("months", 36, "lookback period in months")
		holdoutMonths = flag.Int("holdout-months", 6, "trailing months held out of the ranking window")
		minTurnoverM  = flag.Float64("min-turnover", 50, "gate: minimum mean daily turnover in millions of RUB")
		minATRPct     = flag.Float64("min-atr-pct", 1.5, "gate: minimum mean weekday daily ATR as a percentage of close")
		maxTickPct    = flag.Float64("max-tick-pct", 0.3, "gate: maximum two price ticks as a percentage of the last close (0 = off)")
		topN          = flag.Int("top", 50, "ranking rows in the report (0 = all)")
		workers       = flag.Int("workers", 8, "concurrent tickers")
		pfCap         = flag.Float64("pf-cap", 10, "cap on profit factor for ranking (0 = no cap)")
		cash          = flag.Float64("cash", 100000, "starting mock cash")
		fraction      = flag.Float64("fraction", 1.0, "fraction of cash per entry")
		commission    = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		tickersCSV    = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		outDir        = flag.String("out", "reports/zone_screen", "report output directory")
		refresh       = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		pause         = flag.Duration("pause", 0, "idle time after each ticker, e.g. 500ms or 2s: trades wall-clock for a cooler CPU")
	)
	flag.Parse()
	logger.Init()

	opts := svc.DefaultScreenOpts()
	opts.PFCap = *pfCap
	opts.Cash = *cash
	opts.Fraction = *fraction
	opts.Commission = *commission

	if err := run(context.Background(), runCfg{
		months: *months, holdoutMonths: *holdoutMonths, topN: *topN, workers: *workers,
		minTurnoverM: *minTurnoverM, minATRPct: *minATRPct, maxTickPct: *maxTickPct,
		tickers: screenrun.SplitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, pause: *pause, opts: opts,
	}); err != nil {
		log.Fatalf("zonescreen: %v", err)
	}
}

type runCfg struct {
	months, holdoutMonths, topN, workers int
	minTurnoverM, minATRPct, maxTickPct  float64
	tickers                              []string
	outDir                               string
	refresh                              bool
	pause                                time.Duration
	opts                                 svc.ScreenOpts
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	if cfg.holdoutMonths >= cfg.months {
		return fmt.Errorf("-holdout-months (%d) must be smaller than -months (%d)", cfg.holdoutMonths, cfg.months)
	}
	if cfg.workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	if cfg.maxTickPct < 0 {
		return fmt.Errorf("-max-tick-pct must be >= 0 (0 disables the gate)")
	}
	return nil
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	if workers, capped := screenrun.EffectiveWorkers(cfg.workers, cfg.refresh); capped {
		fmt.Printf("zonescreen: -refresh forces -workers %d -> %d to avoid punching holes in the local candle cache (see screenrun.RefreshWorkerCap)\n",
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
	grid := svc.ZoneGrid()

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.ZoneRow
	var done, skipped int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u screenrun.ShareInfo) {
			defer wg.Done()
			defer sem.Release()

			bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("zonescreen %s: skip (load 30m: %v)\n", u.Ticker, err)
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("zonescreen %s: skip (load daily: %v)\n", u.Ticker, err)
				return
			}

			row := svc.ScreenZoneTicker(svc.ZoneTickerInput{
				Ticker: u.Ticker, Name: u.Name, Bars: bars, Daily: daily, Lot: u.Lot, MinPriceIncrement: u.MinPriceIncrement,
			}, grid, split, cfg.opts)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("zonescreen [%d/%d] %s: PFmed=%.2f PFmed×2=%.2f trades=%.0f tick=%.2f%% turnover=%.0fM atr=%.2f%% bars=%d\n",
				n, len(universe), u.Ticker, row.PFMed, row.PFMedStress, row.TradesMed, row.TickPct, row.TurnoverM, row.DailyATRPct, row.Bars)

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()

			// Held inside the semaphore slot on purpose: releasing first would let the
			// next ticker start while this one idles, and the pool would stay just as hot.
			screenrun.PauseAfterTicker(ctx, cfg.pause)
		}(u)
	}
	wg.Wait()

	gates := svc.ZoneGates{MinTurnoverM: cfg.minTurnoverM, MinATRPct: cfg.minATRPct, MaxTickPct: cfg.maxTickPct}
	ranked, noSignals, tickRejected, rejected := svc.FilterAndRankZone(rows, gates)
	meta := svc.ZoneScreenMeta{
		Months: cfg.months, HoldoutMonths: cfg.holdoutMonths, TopN: cfg.topN, Split: split, Gates: gates,
		PFCap: cfg.opts.PFCap, Commission: cfg.opts.Commission, Cash: cfg.opts.Cash,
		Scanned: len(universe), Skipped: int(skipped), Rejected: len(rejected),
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("zone_screen_Minutes30_%s.md", time.Now().Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(svc.RenderZoneScreenMarkdown(ranked, noSignals, tickRejected, meta)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("zonescreen report: %s (scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d)\n",
		path, len(universe), len(ranked), len(noSignals), len(tickRejected), len(rejected), skipped)
	return nil
}
