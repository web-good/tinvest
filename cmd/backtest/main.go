// Command backtest replays a per-share trading strategy (scalping or levels) over
// historical candles, simulates a mock portfolio, and writes Markdown + CSV reports.
// It supports a single run (default params or -params) and grid calibration
// (-calibrate). All gRPC/file I/O is here; the engine and metrics are pure.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"

	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
)

const (
	apiAddress = "invest-public-api.tinkoff.ru:443"
	cacheDir   = "data/candles"
)

func main() {
	var (
		ticker       = flag.String("ticker", "", "ticker, e.g. RUAL (required)")
		intervalS    = flag.String("interval", "Hour1", "candle timeframe: Minutes15|Minutes30|Hour1|Hour4|Day1|Week1")
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|levels|momentum")
		months       = flag.Int("months", 12, "lookback period in months")
		cash         = flag.Float64("cash", 100000, "starting mock cash")
		fraction     = flag.Float64("fraction", 1.0, "fraction of cash per Buy")
		commission   = flag.Float64("commission", 0.0005, "commission as a fraction of turnover")
		paramsPath   = flag.String("params", "", "path to JSON Params (default: DefaultParams)")
		calibrate    = flag.String("calibrate", "", "path to grid JSON (grid-search mode)")
		metric       = flag.String("metric", "expectancy", "ranking metric: profit_factor|net_pnl|win_rate|max_drawdown|expectancy|sortino")
		minTrades    = flag.Int("min-trades", 15, "calibration: combos with fewer trades sink below qualified ones")
		testMonths   = flag.Int("test-months", 0, "walk-forward: calibrate on the earlier window, report best on the last N months")
		outDir       = flag.String("out", "reports", "report output directory")
		refresh      = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		explain      = flag.String("explain", "", "diagnose one bar: MSK time 'YYYY-MM-DD HH:MM'; prints why the strategy did/didn't enter")
		basket  = flag.String("basket", "", "basket mode: comma-separated tickers; calibrates each on the early window and pools OOS trades (ignores -ticker)")
		gridDir = flag.String("grid-dir", "data/params", "basket mode: directory holding <lower-ticker>/momentum_grid.json")
	)
	flag.Parse()
	logger.Init() // candle fetcher logs chunk errors via the package logger

	interval, err := parseInterval(*intervalS)
	if err != nil {
		log.Fatalf("backtest: %v", err)
	}

	if err := run(*ticker, *strategyName, interval, *months, *cash, *fraction, *commission,
		*paramsPath, *calibrate, *metric, *minTrades, *testMonths, *outDir, *refresh, *explain,
		*basket, *gridDir); err != nil {
		log.Fatalf("backtest: %v", err)
	}
}

// parseInterval maps a timeframe flag string to the enum the candle API uses.
func parseInterval(s string) (enum.Interval, error) {
	switch s {
	case "Minutes15":
		return enum.Minutes15, nil
	case "Minutes30":
		return enum.Minutes30, nil
	case "Hour1":
		return enum.Hour1, nil
	case "Hour4":
		return enum.Hour4, nil
	case "Day1":
		return enum.Day1, nil
	case "Week1":
		return enum.Week1, nil
	default:
		return 0, fmt.Errorf("unknown interval %q (want Minutes15|Hour1|Hour4|Day1|Week1)", s)
	}
}

func run(ticker, strategyName string, interval enum.Interval, months int, cash, fraction, commission float64,
	paramsPath, calibratePath, metric string, minTrades, testMonths int, outDir string, refresh bool, explain string,
	basketCSV, gridDir string,
) error {
	if paramsPath != "" && calibratePath != "" {
		return fmt.Errorf("-params and -calibrate are mutually exclusive")
	}
	if basketCSV != "" && calibratePath != "" {
		return fmt.Errorf("-basket and -calibrate are mutually exclusive (-basket uses per-ticker grids from -grid-dir)")
	}

	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	ctx := context.Background()

	if basketCSV != "" {
		if strategyName != "momentum" {
			return fmt.Errorf("-basket currently supports -strategy momentum only")
		}
		return runBasket(ctx, client, splitTickers(basketCSV), interval, months,
			cash, fraction, commission, metric, minTrades, testMonths, gridDir, outDir, refresh)
	}

	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
	var binding svc.Binding
	switch strategyName {
	case "levels":
		binding = svc.LevelsLookupOrGeneric(ticker)
	case "momentum":
		binding = svc.MomentumLookupOrGeneric(ticker)
	case "scalping":
		binding = svc.LookupOrGeneric(ticker)
	default:
		return fmt.Errorf("unknown strategy %q (want scalping|levels|momentum)", strategyName)
	}

	share, err := resolveShare(ctx, client, ticker)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -months, 0)

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	candles, err := provider.Load(ctx, ticker, share.ID, interval, from, to, refresh)
	if err != nil {
		return err
	}

	dailyFrom := from.AddDate(-1, 0, 0) // ~250 trading days of lead-in to warm the daily EMA
	dailyCandles, err := provider.Load(ctx, ticker, share.ID, enum.Day1, dailyFrom, to, refresh)
	if err != nil {
		return err
	}

	cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot}
	periodDays := to.Sub(from).Hours() / 24

	if explain != "" {
		mskLoc, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			mskLoc = time.UTC
		}
		target, err := time.ParseInLocation("2006-01-02 15:04", explain, mskLoc)
		if err != nil {
			return fmt.Errorf("parse -explain time %q (want 'YYYY-MM-DD HH:MM' MSK): %w", explain, err)
		}
		fmt.Println(domain.Trace(binding.Build(binding.DefaultParams()), candles, dailyCandles, cfg, target))
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	base := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s_%s", ticker, strategyName, interval.String(), stamp))

	if calibratePath != "" {
		return runCalibration(binding, calibratePath, candles, dailyCandles, cfg, metric, minTrades, testMonths, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg), to)
	}

	return runSingle(binding, paramsPath, candles, dailyCandles, cfg, periodDays, base,
		metaCommon(ticker, interval, from, to, cfg))
}

func runSingle(b svc.Binding, paramsPath string, candles []domain.Candle, dailyCandles []domain.Candle,
	cfg domain.Config, periodDays float64, base string, meta domain.Meta,
) error {
	params := b.DefaultParams()
	if paramsPath != "" {
		raw, err := os.ReadFile(paramsPath)
		if err != nil {
			return fmt.Errorf("read params: %w", err)
		}
		params, err = b.ParseParams(raw)
		if err != nil {
			return err
		}
	}

	if len(candles) < b.Build(params).Lookback() {
		fmt.Printf("⚠️ not enough candles (%d) for lookback; empty report\n", len(candles))
	}

	res := domain.Run(b.Build(params), candles, dailyCandles, cfg)
	m := domain.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)

	meta.Params = svc.ParamRows(params)
	meta.OpenPosition = openPosition(res)

	if err := writeFile(base+".md", domain.RenderMarkdown(meta, m, res.Trades, res.Equity)); err != nil {
		return err
	}
	if err := writeFile(base+"_trades.csv", domain.RenderTradesCSV(res.Trades)); err != nil {
		return err
	}
	if err := writeFile(base+"_equity.csv", domain.RenderEquityCSV(res.Equity)); err != nil {
		return err
	}
	fmt.Printf("report: %s.md (trades=%d, net=%.2f, PF=%.3f)\n", base, m.TotalTrades, m.NetPnL, m.ProfitFactor)
	return nil
}

func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, dailyCandles []domain.Candle,
	cfg domain.Config, metric string, minTrades, testMonths int, periodDays float64, base string, meta domain.Meta, to time.Time,
) error {
	raw, err := os.ReadFile(gridPath)
	if err != nil {
		return fmt.Errorf("read grid: %w", err)
	}
	phases, err := svc.ParsePhases(raw)
	if err != nil {
		return err
	}

	gridCandles, gridDaily := candles, dailyCandles
	bestCandles, bestDaily := candles, dailyCandles
	bestDays := periodDays
	gridDays := periodDays
	var boundary time.Time
	if testMonths > 0 {
		boundary = to.AddDate(0, -testMonths, 0)
		testDays := to.Sub(boundary).Hours() / 24
		if testDays >= periodDays {
			return fmt.Errorf("test-months window (%.0f days) must be smaller than the backtest window (%.0f days)", testDays, periodDays)
		}
		gridCandles, bestCandles = svc.SplitByTime(candles, boundary)
		gridDaily, bestDaily = svc.SplitByTime(dailyCandles, boundary)
		bestDays = testDays
		gridDays = periodDays - testDays
	}

	results, err := svc.RunPhases(b, phases, gridCandles, gridDaily, cfg, metric, minTrades, gridDays,
		func(p svc.PhaseProgress) {
			fmt.Printf("phase %s: %d combos -> kept %d (best %s=%.4g)\n", p.Name, p.Combos, p.Kept, metric, p.BestMetric)
		})
	if err != nil {
		return err
	}
	calibPath := base + "_calibration.md"
	if err := writeFile(calibPath, svc.RenderCalibrationMarkdown(metric, results, 20)); err != nil {
		return err
	}

	// Also emit the full single-run report for the best combination.
	if len(results) > 0 {
		best := results[0].Params
		res := domain.Run(b.Build(best), bestCandles, bestDaily, cfg)
		m := domain.Compute(res, res.BarsInMarket, len(res.Equity), bestDays)
		bestMeta := meta
		bestMeta.Params = svc.ParamRows(best)
		bestMeta.OpenPosition = openPosition(res)
		if testMonths > 0 {
			bestMeta.From = boundary
		}
		if err := writeFile(base+"_best.md", domain.RenderMarkdown(bestMeta, m, res.Trades, res.Equity)); err != nil {
			return err
		}
	}
	fmt.Printf("calibration: %s (combos=%d, test_months=%d)\n", calibPath, len(results), testMonths)
	return nil
}

func metaCommon(ticker string, interval enum.Interval, from, to time.Time, cfg domain.Config) domain.Meta {
	return domain.Meta{
		Ticker:      ticker,
		Interval:    interval.String(),
		From:        from,
		To:          to,
		InitialCash: cfg.InitialCash,
		Fraction:    cfg.Fraction,
		Commission:  cfg.Commission,
	}
}

func openPosition(res domain.Result) bool {
	if len(res.Equity) == 0 {
		return false
	}
	var closedBars int
	for _, t := range res.Trades {
		closedBars += t.BarsHeld
	}
	return res.BarsInMarket > closedBars
}

func resolveShare(ctx context.Context, client grpcclient.GrpcClient, ticker string) (shareInfo, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return shareInfo{}, fmt.Errorf("load shares: %w", err)
	}
	for _, s := range shares {
		if s.Ticker == ticker {
			return shareInfo{ID: s.ID, Lot: s.Lot}, nil
		}
	}
	return shareInfo{}, fmt.Errorf("ticker %q not found in Shares()", ticker)
}

type shareInfo struct {
	ID  string
	Lot int32
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

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// splitTickers parses a comma-separated ticker list, trimming blanks.
func splitTickers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// runBasket calibrates each ticker on the early window, runs the winner on the shared
// OOS tail, and pools the tail trades across tickers into one statistically meaningful
// sample. Each ticker keeps its own calibrated params; the pool is validation only.
func runBasket(ctx context.Context, client grpcclient.GrpcClient, tickers []string, interval enum.Interval,
	months int, cash, fraction, commission float64, metric string, minTrades, testMonths int,
	gridDir, outDir string, refresh bool,
) error {
	if testMonths <= 0 {
		return fmt.Errorf("-basket requires -test-months > 0 (walk-forward OOS window)")
	}
	if len(tickers) == 0 {
		return fmt.Errorf("-basket: no tickers parsed")
	}

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)
	boundary := to.AddDate(0, -testMonths, 0)
	dailyFrom := from.AddDate(-1, 0, 0)
	periodDays := to.Sub(from).Hours() / 24
	testDays := to.Sub(boundary).Hours() / 24
	gridDays := periodDays - testDays
	if testDays >= periodDays {
		return fmt.Errorf("-test-months window (%.0f days) must be smaller than the -months window (%.0f days)", testDays, periodDays)
	}

	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	byTicker := make(map[string]shareInfo, len(shares))
	for _, s := range shares {
		byTicker[s.Ticker] = shareInfo{ID: s.ID, Lot: s.Lot}
	}

	var pooled []domain.Trade
	var summary svc.BasketSummary

	for _, ticker := range tickers {
		entry := svc.BasketEntry{Ticker: ticker}

		gridPath := filepath.Join(gridDir, strings.ToLower(ticker), "momentum_grid.json")
		raw, err := os.ReadFile(gridPath)
		if err != nil {
			entry.Skipped, entry.Note = true, fmt.Sprintf("нет грида (%s)", gridPath)
			summary.Entries = append(summary.Entries, entry)
			continue
		}
		phases, err := svc.ParsePhases(raw)
		if err != nil {
			entry.Skipped, entry.Note = true, fmt.Sprintf("грид невалиден: %v", err)
			summary.Entries = append(summary.Entries, entry)
			continue
		}

		share, ok := byTicker[ticker]
		if !ok {
			entry.Skipped, entry.Note = true, "инструмент не найден в Shares()"
			summary.Entries = append(summary.Entries, entry)
			continue
		}

		candles, err := provider.Load(ctx, ticker, share.ID, interval, from, to, refresh)
		if err != nil {
			return fmt.Errorf("%s: load candles: %w", ticker, err)
		}
		dailyCandles, err := provider.Load(ctx, ticker, share.ID, enum.Day1, dailyFrom, to, refresh)
		if err != nil {
			return fmt.Errorf("%s: load daily: %w", ticker, err)
		}

		binding := svc.MomentumLookupOrGeneric(ticker)
		cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot}
		gridCandles, bestCandles := svc.SplitByTime(candles, boundary)
		gridDaily, bestDaily := svc.SplitByTime(dailyCandles, boundary)

		results, err := svc.RunPhases(binding, phases, gridCandles, gridDaily, cfg, metric, minTrades, gridDays, nil)
		if err != nil {
			return fmt.Errorf("%s: calibrate: %w", ticker, err)
		}
		if len(results) == 0 {
			entry.Skipped, entry.Note = true, "калибровка не дала комбинаций"
			summary.Entries = append(summary.Entries, entry)
			continue
		}

		best := results[0].Params
		res := domain.Run(binding.Build(best), bestCandles, bestDaily, cfg)
		m := domain.Compute(res, res.BarsInMarket, len(res.Equity), testDays)

		entry.Trades = m.TotalTrades
		entry.ProfitFactor = m.ProfitFactor
		entry.NetPnL = m.NetPnL
		entry.NetPnLPct = m.NetPnLPct
		entry.MaxDrawdownPct = m.MaxDrawdownPct
		entry.WinRate = m.WinRate
		entry.Params = svc.ParamRows(best)
		if m.TotalTrades == 0 {
			entry.Note = "нет OOS-сделок"
		}
		summary.Entries = append(summary.Entries, entry)
		pooled = append(pooled, res.Trades...)
		fmt.Printf("basket %s: OOS trades=%d PF=%.3f net=%.2f\n", ticker, m.TotalTrades, m.ProfitFactor, m.NetPnL)
	}

	summary.Pooled = svc.PooledMetrics(pooled)

	if len(pooled) == 0 {
		skipped := 0
		for _, e := range summary.Entries {
			if e.Skipped {
				skipped++
			}
		}
		if skipped == len(summary.Entries) {
			return fmt.Errorf("basket: every ticker was skipped (no grids / instruments not found); nothing to report")
		}
	}

	basketDir := filepath.Join(outDir, "basket")
	if err := os.MkdirAll(basketDir, 0o755); err != nil {
		return fmt.Errorf("mkdir basket dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	base := filepath.Join(basketDir, fmt.Sprintf("basket_momentum_%s", stamp))
	if err := writeFile(base+".md", svc.RenderBasketMarkdown(metric, summary, boundary, to)); err != nil {
		return err
	}
	if err := writeFile(base+"_trades.csv", domain.RenderTradesCSV(pooled)); err != nil {
		return err
	}
	fmt.Printf("basket report: %s.md (pooled trades=%d, PF=%.3f)\n", base, summary.Pooled.TotalTrades, summary.Pooled.ProfitFactor)
	return nil
}
