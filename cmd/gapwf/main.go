// Command gapwf validates the gap_fade strategy on a whole ticker universe at once: one set of
// parameters for every ticker, chosen per walk-forward fold by the pooled profit factor of all
// tickers' training trades, scored on the pooled out-of-sample trades and judged by the spec's
// six-condition gate. Every grid combination runs once per ticker over the full window; folds
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
	"time"

	"github.com/joho/godotenv"

	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/logger"
)

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
		tickers: dedupeTickers(tickers), months: *months, trainMonths: *trainMonths, testMonths: *testMonths,
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

// dedupeTickers drops repeated tickers (case-insensitive), keeping the first spelling and the
// order: a duplicate would run twice and double its weight in the pool.
func dedupeTickers(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, t := range in {
		k := strings.ToUpper(t)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t)
	}
	return out
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

	if !svc.IsGapCanonicalCommission(cfg.commission) {
		fmt.Printf("gapwf: ВНИМАНИЕ: издержки %.4f на сторону вместо 0.0005 — пороги гейта заданы не для них, вердикт неканоничный\n", cfg.commission)
	}
	to := time.Now()
	from := windowStart(to, cfg.months)
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

	res, err := evaluate(data, combos, cfg, from, to)
	if err != nil {
		return err
	}

	loaded := make([]string, len(data))
	for i, d := range data {
		loaded[i] = d.ticker
	}
	md := svc.RenderGapWFMarkdown(svc.GapReport{
		Generated: to, From: from, To: to, Last12From: res.last12From,
		TrainMonths: cfg.trainMonths, TestMonths: cfg.testMonths, MinTrades: cfg.minTrades, Commission: cfg.commission,
		Universe: loaded, Skipped: skipped, Runs: res.runs, Folds: res.folds, Sensitivity: res.sens, ExDays: res.exDays, Verdict: res.verdict,
	})
	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("gapwf_%s.md", to.Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	fmt.Printf("gapwf: гейт %s, pooled OOS %d сделок; отчёт %s\n",
		svc.GapVerdictStatus(res.verdict, cfg.commission), len(svc.PooledGapOOS(res.folds)), path)
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
	return tickerData{ticker: u.Ticker, lot: u.Lot, bars: bars, daily: daily, exDays: dividendExDays(divs, bars)}, nil
}

// dividendExDays maps every non-cancelled dividend to its ex-day: the first trading bar's date
// after last_buy_date (svc.DividendExDaysFromBars). A dividend without last_buy_date falls back
// to its record date, which under T+1 settlement is itself the ex-day.
func dividendExDays(divs []*pkgmodel.Dividend, bars []domain.Candle) map[string]bool {
	barTimes := make([]time.Time, len(bars))
	for i, b := range bars {
		barTimes[i] = b.Time
	}
	var lastBuy []time.Time
	var recordEx []time.Time
	for _, d := range divs {
		switch {
		case d == nil || d.DividendType == "Cancelled":
		case !d.LastBuyDate.IsZero():
			lastBuy = append(lastBuy, d.LastBuyDate)
		case !d.RecordDate.IsZero():
			recordEx = append(recordEx, d.RecordDate)
		}
	}
	out := svc.DividendExDaysFromBars(lastBuy, barTimes)
	for _, r := range recordEx {
		out[svc.GapDay(r)] = true
	}
	return out
}
