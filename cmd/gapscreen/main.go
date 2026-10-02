// Command gapscreen picks the gap_fade universe by structure, not by profit: a ticker passes when
// its 07:00 bar trades almost every weekday, that bar is liquid enough to absorb the entry, the
// price tick is small and the daily turnover is high. The profit check of the picked set is
// cmd/gapwf. All gRPC/file I/O lives here; the metrics, gates and report are pure
// (internal/service/backtest/gap_screen*.go). Spec: docs/superpowers/specs/2026-10-02-gap-fade-screener-design.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

func main() {
	def := svc.DefaultGapScreenOptions()
	var (
		months          = flag.Int("months", 12, "lookback window in months")
		minMorningDays  = flag.Float64("min-morning-days", def.MinMorningDays, "gate: minimum share of weekdays with a 07:00 bar (0..1)")
		minMorningTurnM = flag.Float64("min-morning-turnover", def.MinMorningTurnoverM, "gate: minimum median 07:00-bar turnover in millions of RUB")
		maxTickPct      = flag.Float64("max-tick-pct", def.MaxTickPct, "gate: maximum two price ticks as a percentage of the last close (0 = off)")
		minTurnoverM    = flag.Float64("min-turnover", def.MinTurnoverM, "gate: minimum mean daily turnover in millions of RUB")
		tickersCSV      = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		workers         = flag.Int("workers", 8, "concurrent tickers")
		outDir          = flag.String("out", "reports/gap_fade", "report output directory")
		refresh         = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		pause           = flag.Duration("pause", 0, "idle time after each ticker, e.g. 500ms or 2s: trades wall-clock for a cooler CPU")
	)
	flag.Parse()
	logger.Init()

	if err := run(context.Background(), runCfg{
		months: *months, workers: *workers,
		opts: svc.GapScreenOptions{
			MinMorningDays: *minMorningDays, MinMorningTurnoverM: *minMorningTurnM,
			MaxTickPct: *maxTickPct, MinTurnoverM: *minTurnoverM,
		},
		tickers: screenrun.SplitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, pause: *pause,
	}); err != nil {
		log.Fatalf("gapscreen: %v", err)
	}
}

type runCfg struct {
	months, workers int
	opts            svc.GapScreenOptions
	tickers         []string
	outDir          string
	refresh         bool
	pause           time.Duration
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	switch {
	case cfg.months <= 0:
		return fmt.Errorf("-months must be > 0 (got %d)", cfg.months)
	case cfg.workers < 1:
		return fmt.Errorf("-workers must be >= 1 (got %d)", cfg.workers)
	case cfg.opts.MinMorningDays < 0 || cfg.opts.MinMorningDays > 1:
		return fmt.Errorf("-min-morning-days must be within [0, 1] (got %v)", cfg.opts.MinMorningDays)
	case cfg.opts.MinMorningTurnoverM < 0:
		return errors.New("-min-morning-turnover must be >= 0")
	case cfg.opts.MaxTickPct < 0:
		return errors.New("-max-tick-pct must be >= 0 (0 disables the gate)")
	case cfg.opts.MinTurnoverM < 0:
		return errors.New("-min-turnover must be >= 0")
	}
	return nil
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	if workers, capped := screenrun.EffectiveWorkers(cfg.workers, cfg.refresh); capped {
		fmt.Printf("gapscreen: -refresh forces -workers %d -> %d to avoid punching holes in the local candle cache (see screenrun.RefreshWorkerCap)\n",
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
	dailyFrom := from.AddDate(-1, 0, 0) // a year of lead-in warms the daily ATR, as in cmd/gapwf
	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.GapScreenRow
	var noData []string
	skip := func(ticker, why string) {
		mu.Lock()
		noData = append(noData, ticker+": "+why)
		mu.Unlock()
		fmt.Printf("gapscreen %s: skip (%s)\n", ticker, why)
	}

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u screenrun.ShareInfo) {
			defer wg.Done()
			defer sem.Release()
			// Held inside the semaphore slot on purpose: releasing first would let the next
			// ticker start while this one idles, and the pool would stay just as hot.
			defer screenrun.PauseAfterTicker(ctx, cfg.pause)

			bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, cfg.refresh)
			if err != nil {
				skip(u.Ticker, fmt.Sprintf("свечи 30m: %v", err))
				return
			}
			if len(bars) == 0 {
				skip(u.Ticker, "нет свечей 30m")
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				skip(u.Ticker, fmt.Sprintf("дневные свечи: %v", err))
				return
			}
			row := svc.ScreenGap(svc.GapScreenInput{
				Ticker: u.Ticker, Name: u.Name, Bars: bars, Lot: u.Lot, MinPriceIncrement: u.MinPriceIncrement,
				SignalsPerYear: svc.GapSignalsPerYear(u.Ticker, bars, daily, u.Lot, cfg.months),
			}, cfg.opts)
			fmt.Printf("gapscreen %s: pass=%v morning=%.0f%% morningTurnover=%.1fM turnover=%.0fM tick=%.2f%% signals/yr=%.1f\n",
				u.Ticker, row.Pass, row.MorningDays*100, row.MorningTurnoverM, row.TurnoverM, row.TickPct, row.SignalsPerYear)
			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()
		}(u)
	}
	wg.Wait()
	sort.Strings(noData)

	md := svc.RenderGapScreenMarkdown(rows, svc.GapScreenMeta{
		Generated: to, Months: cfg.months, Scanned: len(universe), Opts: cfg.opts, NoData: noData,
	})
	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("screen_%s.md", to.Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	passed := len(svc.GapScreenPassed(rows))
	if passed == 0 {
		fmt.Println("gapscreen: ВНИМАНИЕ: ни один тикер не прошёл гейты")
	}
	fmt.Printf("gapscreen: прошли %d из %d (нет данных %d); отчёт %s\n%s\n",
		passed, len(universe), len(noData), path, svc.GapScreenTickersLine(rows))
	return nil
}
