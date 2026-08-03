// Command pullscreen ranks the tradable RUB share universe by how well the
// rsi_pullback strategy fits each ticker. Unlike the older -volrank screener it does
// NOT use a cheap entry proxy: it replays the real strategy engine over a fixed
// 24-configuration grid and ranks by the MEDIAN profit factor, because a proxy that
// only measured the entry side failed to predict realized walk-forward results
// (docs/reversion/screener.md). All gRPC/file I/O lives here; the scoring is pure.
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

	"github.com/joho/godotenv"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

const (
	apiAddress = "invest-public-api.tinkoff.ru:443"
	cacheDir   = "data/candles"
)

func main() {
	var (
		months        = flag.Int("months", 36, "lookback period in months")
		holdoutMonths = flag.Int("holdout-months", 6, "trailing months held out of the ranking window")
		minTurnoverM  = flag.Float64("min-turnover", 50, "gate: minimum mean daily turnover in millions of RUB")
		minATRPct     = flag.Float64("min-atr-pct", 1.5, "gate: minimum mean weekday daily ATR as a percentage of close")
		topN          = flag.Int("top", 50, "ranking rows in the report (0 = all)")
		workers       = flag.Int("workers", 8, "concurrent tickers")
		pfCap         = flag.Float64("pf-cap", 10, "cap on profit factor for ranking (0 = no cap)")
		cash          = flag.Float64("cash", 100000, "starting mock cash")
		fraction      = flag.Float64("fraction", 1.0, "fraction of cash per entry")
		commission    = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		tickersCSV    = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		outDir        = flag.String("out", "reports/pullback_screen", "report output directory")
		refresh       = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
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
		minTurnoverM: *minTurnoverM, minATRPct: *minATRPct,
		tickers: splitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, opts: opts,
	}); err != nil {
		log.Fatalf("pullscreen: %v", err)
	}
}

type runCfg struct {
	months, holdoutMonths, topN, workers int
	minTurnoverM, minATRPct              float64
	tickers                              []string
	outDir                               string
	refresh                              bool
	opts                                 svc.ScreenOpts
}

// shareInfo is the per-ticker metadata the worker pool needs.
type shareInfo struct {
	Ticker string
	Name   string
	ID     string
	Lot    int32
}

func run(ctx context.Context, cfg runCfg) error {
	if cfg.holdoutMonths >= cfg.months {
		return fmt.Errorf("-holdout-months (%d) must be smaller than -months (%d)", cfg.holdoutMonths, cfg.months)
	}
	if cfg.workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}

	universe, err := loadUniverse(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	// A year of lead-in warms the daily ATR, exactly as cmd/backtest does.
	dailyFrom := from.AddDate(-1, 0, 0)
	split := to.AddDate(0, -cfg.holdoutMonths, 0)

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	grid := svc.PullbackGrid()

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.PullbackRow
	var done, skipped int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u shareInfo) {
			defer wg.Done()
			defer sem.Release()

			bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("pullscreen %s: skip (load 30m: %v)\n", u.Ticker, err)
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("pullscreen %s: skip (load daily: %v)\n", u.Ticker, err)
				return
			}

			row := svc.ScreenTicker(u.Ticker, u.Name, bars, daily, u.Lot, grid, split, cfg.opts)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("pullscreen [%d/%d] %s: PFmed=%.2f trades=%.0f plateau=%.0f%% turnover=%.0fM atr=%.2f%% bars=%d\n",
				n, len(universe), u.Ticker, row.PFMed, row.TradesMed, row.Plateau*100, row.TurnoverM, row.DailyATRPct, row.Bars)

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	ranked, noSignals, rejected := svc.FilterAndRank(rows, cfg.minTurnoverM, cfg.minATRPct)
	meta := svc.ScreenMeta{
		Months: cfg.months, HoldoutMonths: cfg.holdoutMonths, TopN: cfg.topN, Split: split,
		MinTurnoverM: cfg.minTurnoverM, MinATRPct: cfg.minATRPct, PFCap: cfg.opts.PFCap,
		Scanned: len(universe), Passed: len(ranked), Skipped: int(skipped),
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("pullback_screen_Minutes30_%s.md", time.Now().Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(svc.RenderPullbackScreenMarkdown(ranked, noSignals, meta)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("pullscreen report: %s (scanned=%d ranked=%d no-signal=%d rejected=%d skipped=%d)\n",
		path, len(universe), len(ranked), len(noSignals), len(rejected), skipped)
	return nil
}

// loadUniverse returns the tradable RUB share universe, or just the requested tickers.
func loadUniverse(ctx context.Context, client grpcclient.GrpcClient, only []string) ([]shareInfo, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("load shares: %w", err)
	}
	want := make(map[string]bool, len(only))
	for _, t := range only {
		want[strings.ToUpper(t)] = true
	}
	var universe []shareInfo
	for _, s := range shares {
		if len(want) > 0 {
			if !want[strings.ToUpper(s.Ticker)] {
				continue
			}
		} else if !strings.EqualFold(s.Currency, "rub") || !s.Trading {
			continue
		}
		universe = append(universe, shareInfo{Ticker: s.Ticker, Name: s.Name, ID: s.ID, Lot: s.Lot})
	}
	if len(universe) == 0 {
		return nil, fmt.Errorf("no matching shares found")
	}
	return universe, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadToken() (string, error) {
	_ = godotenv.Load("./env/local.env")
	_ = godotenv.Load("./env/token.env")
	token := os.Getenv("T_BANK")
	if token == "" {
		return "", fmt.Errorf("T_BANK is not set (checked env + ./env/local.env, ./env/token.env)")
	}
	return token, nil
}
