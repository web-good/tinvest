// Command gapwf validates the gap_fade strategy on a whole ticker universe at once: one set of
// parameters for every ticker, chosen per walk-forward fold by the pooled profit factor of all
// tickers' training trades, scored on the pooled out-of-sample trades and judged by the spec's
// four-condition gate. Every grid combination runs once per ticker over the full window; folds
// slice those trades by entry date, which is exact for an intraday strategy. Trades entered on a
// dividend ex-day are dropped. All gRPC/file I/O lives here; the fold logic, gate and report are
// pure (internal/service/backtest/gap_wf*.go). Spec: docs/superpowers/specs/2026-10-02-gap-fade-design.md.
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
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

// engineCash is the mock account per ticker run; with Fraction 1.0 lot rounding is negligible
// and the pooled PF is taken on per-trade returns anyway.
const engineCash = 1e7

// stressCommissions are the per-side costs of the sensitivity table; the last one feeds gate 4.
var stressCommissions = []float64{0.001, 0.002}

func main() {
	var (
		tickersCSV  = flag.String("tickers", "", "comma-separated universe (default: RSI_PULLBACK_TICKERS from -env-file)")
		envFile     = flag.String("env-file", "env/prod.env", "file the default universe is read from")
		months      = flag.Int("months", 36, "history window in months")
		trainMonths = flag.Int("train-months", 12, "walk-forward training window in months")
		testMonths  = flag.Int("test-months", 3, "walk-forward test window in months")
		gridPath    = flag.String("grid", "data/params/gap_fade/grid.json", "one-phase calibration grid")
		commission  = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		minTrades   = flag.Int("min-trades", 30, "minimum pooled training trades for a combination to be chosen")
		outDir      = flag.String("out", "reports/gap_fade", "report output directory")
		workers     = flag.Int("workers", 8, "concurrent tickers")
		refresh     = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
	)
	flag.Parse()
	logger.Init()

	tickers := screenrun.SplitCSV(*tickersCSV)
	if len(tickers) == 0 {
		var err error
		if tickers, err = defaultTickers(*envFile); err != nil {
			log.Fatalf("gapwf: %v", err)
		}
	}
	if err := run(context.Background(), runCfg{
		tickers: tickers, months: *months, trainMonths: *trainMonths, testMonths: *testMonths,
		minTrades: *minTrades, workers: *workers, commission: *commission,
		gridPath: *gridPath, outDir: *outDir, refresh: *refresh,
	}); err != nil {
		log.Fatalf("gapwf: %v", err)
	}
}

type runCfg struct {
	tickers                                             []string
	months, trainMonths, testMonths, minTrades, workers int
	commission                                          float64
	gridPath, outDir                                    string
	refresh                                             bool
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	switch {
	case len(cfg.tickers) == 0:
		return errors.New("empty universe")
	case cfg.trainMonths <= 0 || cfg.testMonths <= 0:
		return fmt.Errorf("train-months and test-months must be > 0 (got %d/%d)", cfg.trainMonths, cfg.testMonths)
	case cfg.months < cfg.trainMonths+cfg.testMonths:
		return fmt.Errorf("months %d < train-months+test-months %d: no fold fits", cfg.months, cfg.trainMonths+cfg.testMonths)
	case cfg.minTrades < 1:
		return fmt.Errorf("min-trades must be >= 1 (got %d)", cfg.minTrades)
	case cfg.commission < 0:
		return fmt.Errorf("commission must be >= 0 (got %v)", cfg.commission)
	case cfg.workers < 1:
		return fmt.Errorf("workers must be >= 1 (got %d)", cfg.workers)
	}
	return nil
}

// defaultTickers reads the rsi_pullback production universe — gap_fade is meant as its companion.
func defaultTickers(path string) ([]string, error) {
	env, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	t := screenrun.SplitCSV(env["RSI_PULLBACK_TICKERS"])
	if len(t) == 0 {
		return nil, fmt.Errorf("%s: RSI_PULLBACK_TICKERS is empty", path)
	}
	return t, nil
}

// missingTickers lists requested tickers the share list did not return.
func missingTickers(requested, found []string) []string {
	have := make(map[string]bool, len(found))
	for _, f := range found {
		have[strings.ToUpper(f)] = true
	}
	var out []string
	for _, r := range requested {
		if !have[strings.ToUpper(r)] {
			out = append(out, strings.ToUpper(r)+": нет в списке акций")
		}
	}
	return out
}

// tickerData is everything one ticker contributes: its bars and the ex-days its trades skip.
type tickerData struct {
	ticker      string
	lot         int32
	bars, daily []domain.Candle
	exDays      map[string]bool
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	raw, err := os.ReadFile(cfg.gridPath)
	if err != nil {
		return fmt.Errorf("read grid: %w", err)
	}
	combos, err := svc.ExpandGapGrid(raw)
	if err != nil {
		return err
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
	sort.Slice(universe, func(i, j int) bool { return universe[i].Ticker < universe[j].Ticker })

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	dailyFrom := from.AddDate(-1, 0, 0) // a year of lead-in warms the daily ATR
	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)

	found := make([]string, len(universe))
	for i, u := range universe {
		found[i] = u.Ticker
	}
	skipped := missingTickers(cfg.tickers, found)
	var data []tickerData
	for _, u := range universe {
		d, err := loadTicker(ctx, client, provider, u, from, dailyFrom, to, cfg.refresh)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", u.Ticker, err))
			fmt.Printf("gapwf %s: skip (%v)\n", u.Ticker, err)
			continue
		}
		data = append(data, d)
	}
	if len(data) == 0 {
		return errors.New("no ticker loaded")
	}
	fmt.Printf("gapwf: %d tickers, %d combos, base commission %.4f\n", len(data), len(combos), cfg.commission)

	base, exDays := runCombos(data, combos, nil, cfg.commission, cfg.workers)
	folds, err := svc.RunGapWalkForward(base, from, to, cfg.trainMonths, cfg.testMonths, cfg.minTrades)
	if err != nil {
		return err
	}
	oos := svc.PooledGapOOS(folds)
	selected := svc.SelectedGapCombos(folds)
	sens := []svc.GapCostRow{{Commission: cfg.commission, Trades: oos}}
	var stressed []svc.GapTrade
	for _, c := range stressCommissions {
		alt, _ := runCombos(data, combos, selected, c, cfg.workers)
		stressed = svc.ReplayGapFolds(folds, alt)
		sens = append(sens, svc.GapCostRow{Commission: c, Trades: stressed})
	}
	last12 := to.AddDate(0, -12, 0)
	verdict := svc.GapGate(oos, stressed, last12)

	loaded := make([]string, len(data))
	for i, d := range data {
		loaded[i] = d.ticker
	}
	md := svc.RenderGapWFMarkdown(svc.GapReport{
		Generated: to, From: from, To: to, Last12From: last12,
		TrainMonths: cfg.trainMonths, TestMonths: cfg.testMonths, MinTrades: cfg.minTrades, Commission: cfg.commission,
		Universe: loaded, Skipped: skipped, Runs: base, Folds: folds, Sensitivity: sens, ExDays: exDays, Verdict: verdict,
	})
	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("gapwf_%s.md", to.Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	status := "ОТКАЗ"
	if verdict.Pass {
		status = "ПРОЙДЕН"
	}
	fmt.Printf("gapwf: гейт %s, pooled OOS %d сделок; отчёт %s\n", status, len(oos), path)
	return nil
}

// loadTicker fetches one ticker's 30m and daily bars and its dividend ex-days. A ticker whose
// dividends cannot be loaded is skipped whole: unfiltered, its ex-day gaps would pose as signals.
func loadTicker(ctx context.Context, client grpcclient.GrpcClient, provider *svc.CandleProvider, u screenrun.ShareInfo,
	from, dailyFrom, to time.Time, refresh bool) (tickerData, error) {
	bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, refresh)
	if err != nil {
		return tickerData{}, fmt.Errorf("свечи 30m: %w", err)
	}
	daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, refresh)
	if err != nil {
		return tickerData{}, fmt.Errorf("дневные свечи: %w", err)
	}
	divs, err := client.InstrumentsServiceClient().GetDividends(ctx, u.ID, dailyFrom, to)
	if err != nil {
		return tickerData{}, fmt.Errorf("дивиденды: %w", err)
	}
	var lastBuy []time.Time
	for _, d := range divs {
		if d.DividendType == "Cancelled" {
			continue
		}
		lastBuy = append(lastBuy, d.LastBuyDate)
	}
	return tickerData{ticker: u.Ticker, lot: u.Lot, bars: bars, daily: daily, exDays: svc.DividendExDays(lastBuy)}, nil
}

// runCombos runs the engine for every combination (or only those in `only`, when non-nil) on
// every ticker at one commission, drops ex-day trades and returns runs index-aligned with
// combos plus, per ticker, the number of distinct ex-days on which some trade was dropped.
// Tickers run concurrently; the merge is in ticker order, so the output is deterministic.
func runCombos(data []tickerData, combos []any, only map[int]bool, commission float64, workers int) ([]svc.GapComboRun, map[string]int) {
	perTicker := make([][][]svc.GapTrade, len(data))
	exHit := make([]int, len(data))
	sem := semaphore.New(workers)
	var wg sync.WaitGroup
	for ti := range data {
		wg.Add(1)
		sem.Acquire()
		go func(ti int) {
			defer wg.Done()
			defer sem.Release()
			d := data[ti]
			binding := svc.GapFadeLookupOrGeneric(d.ticker)
			cfg := domain.Config{InitialCash: engineCash, Fraction: 1.0, Commission: commission, Lot: d.lot}
			days := map[string]bool{}
			perTicker[ti] = make([][]svc.GapTrade, len(combos))
			for ci, p := range combos {
				if only != nil && !only[ci] {
					continue
				}
				res := domain.Run(binding.Build(p), d.bars, d.daily, nil, cfg)
				kept, dropped := svc.DropExDayTrades(svc.TagGapTrades(d.ticker, res.Trades), d.exDays)
				perTicker[ti][ci] = kept
				for _, day := range dropped {
					days[day] = true
				}
			}
			exHit[ti] = len(days)
		}(ti)
	}
	wg.Wait()

	runs := make([]svc.GapComboRun, len(combos))
	for ci := range combos {
		runs[ci].Params = combos[ci]
		for ti := range data {
			runs[ci].Trades = append(runs[ci].Trades, perTicker[ti][ci]...)
		}
	}
	ex := make(map[string]int, len(data))
	for ti, d := range data {
		ex[d.ticker] = exHit[ti]
	}
	return runs, ex
}
