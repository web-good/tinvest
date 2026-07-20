// Command backtest replays a per-share trading strategy (scalping or reversion)
// over historical candles, simulates a mock portfolio, and writes
// Markdown + CSV reports. It supports a single run (default params or -params) and
// grid calibration (-calibrate). All gRPC/file I/O is here; the engine and metrics
// are pure.
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

	domain "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

const (
	apiAddress     = "invest-public-api.tinkoff.ru:443"
	cacheDir       = "data/candles"
	volWorkers     = 6   // concurrent candle fetches for -volrank
	screenTrendEMA = 200 // volrank: trend EMA period (uptrend context for pullback events)
)

func main() {
	var (
		ticker       = flag.String("ticker", "", "ticker, e.g. RUAL (required)")
		intervalS    = flag.String("interval", "Hour1", "candle timeframe: Minutes15|Minutes30|Hour1|Hour4|Day1|Week1")
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|reversion")
		months       = flag.Int("months", 12, "lookback period in months")
		cash         = flag.Float64("cash", 100000, "starting mock cash")
		fraction     = flag.Float64("fraction", 1.0, "fraction of cash per Buy")
		commission   = flag.Float64("commission", 0.0005, "commission as a fraction of turnover")
		paramsPath   = flag.String("params", "", "path to JSON Params (default: DefaultParams)")
		calibrate    = flag.String("calibrate", "", "path to grid JSON (grid-search mode)")
		metric       = flag.String("metric", "expectancy", "ranking metric: profit_factor|net_pnl|win_rate|max_drawdown|expectancy|sortino")
		minTrades    = flag.Int("min-trades", 15, "calibration: combos with fewer trades sink below qualified ones")
		testMonths   = flag.Int("test-months", 0, "walk-forward: calibrate on the earlier window, report best on the last N months")
		trainMonths  = flag.Int("train-months", 0, "rolling walk-forward (with -calibrate): fixed train window in months; step = -test-months")
		outDir       = flag.String("out", "reports", "report output directory")
		refresh      = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		explain      = flag.String("explain", "", "diagnose one bar: MSK time 'YYYY-MM-DD HH:MM'; prints why the strategy did/didn't enter")
		riskPct      = flag.Float64("risk-pct", 0, "risk-based sizing: risk this %% of equity per trade off the entry stop (requires CatStopATRMult>0; 0 = fixed -fraction sizing)")
		screen       = flag.String("screen", "", "screen mode: comma-separated tickers; ranks them by variance ratio (ignores -ticker/-strategy)")
		volRank      = flag.Bool("volrank", false, "screener mode: rank the liquid RUB universe by Hour1 pullback-in-trend fitness (ignores -ticker/-strategy/-interval)")
		minTurnover  = flag.Float64("min-turnover", 50, "volrank: minimum mean daily turnover in millions of RUB")
		atrPeriod    = flag.Int("atr-period", 14, "volrank: ATR period for the Hour1 reference column (not in Score)")
		topN         = flag.Int("top", 50, "volrank: rows in the report (0 = all)")
		wRecov       = flag.Float64("w-recov", 0.5, "volrank: composite weight on recovery-rate percentile")
		wFreq        = flag.Float64("w-freq", 0.3, "volrank: composite weight on event-frequency percentile")
		wLiq         = flag.Float64("w-liq", 0.2, "volrank: composite weight on turnover percentile")
		rsiPeriodS   = flag.Int("rsi-period", 14, "volrank: RSI period for the pullback trigger")
		rsiOversold  = flag.Float64("rsi-oversold", 30, "volrank: RSI oversold threshold (cross-down trigger)")
		recoverBars  = flag.Int("recover-bars", 24, "volrank: bars allowed for a dip's RSI to recover")
		recoverRSI   = flag.Float64("recover-rsi", 50, "volrank: RSI level marking a recovered dip")
	)
	flag.Parse()
	logger.Init() // candle fetcher logs chunk errors via the package logger

	interval, err := parseInterval(*intervalS)
	if err != nil {
		log.Fatalf("backtest: %v", err)
	}

	if err := run(*ticker, *strategyName, interval, *months, *cash, *fraction, *commission,
		*paramsPath, *calibrate, *metric, *minTrades, *testMonths, *trainMonths, *outDir, *refresh, *explain,
		*riskPct, *screen,
		*volRank, *minTurnover, *atrPeriod, *topN, *wRecov, *wFreq, *wLiq, *rsiPeriodS, *rsiOversold, *recoverBars, *recoverRSI); err != nil {
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
	paramsPath, calibratePath, metric string, minTrades, testMonths, trainMonths int, outDir string, refresh bool, explain string,
	riskPct float64, screenCSV string,
	volRank bool, minTurnoverM float64, atrPeriod, topN int, wRecov, wFreq, wLiq float64, rsiPeriodS int, rsiOversold float64, recoverBars int, recoverRSI float64,
) error {
	if paramsPath != "" && calibratePath != "" {
		return fmt.Errorf("-params and -calibrate are mutually exclusive")
	}
	if trainMonths > 0 && calibratePath == "" {
		return fmt.Errorf("-train-months requires -calibrate (walk-forward re-calibrates each fold from a grid)")
	}
	if screenCSV != "" && (calibratePath != "" || explain != "") {
		return fmt.Errorf("-screen is standalone (not combined with -calibrate/-explain)")
	}
	if volRank && (screenCSV != "" || calibratePath != "" || explain != "") {
		return fmt.Errorf("-volrank is standalone (not combined with -screen/-calibrate/-explain)")
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

	if screenCSV != "" {
		return runScreen(ctx, client, splitTickers(screenCSV), interval, months, outDir, refresh)
	}

	if volRank {
		sp := svc.VolParams{
			ATRPeriod:   atrPeriod,
			EMAPeriod:   screenTrendEMA,
			RSIPeriod:   rsiPeriodS,
			RSIOversold: rsiOversold,
			RecoverRSI:  recoverRSI,
			RecoverBars: recoverBars,
		}
		return runVolRank(ctx, client, months, topN, minTurnoverM, wRecov, wFreq, wLiq, sp, outDir, refresh)
	}

	if ticker == "" {
		return fmt.Errorf("-ticker is required")
	}
	var binding svc.Binding
	switch strategyName {
	case "reversion":
		binding = svc.ReversionLookupOrGeneric(ticker)
	case "scalping":
		binding = svc.LookupOrGeneric(ticker)
	default:
		return fmt.Errorf("unknown strategy %q (want scalping|reversion)", strategyName)
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

	// Reversion's optional 4H HTF trend filter needs real Hour4 candles, loaded with the
	// same year lead-in as the daily series to warm the 4H EMA. Other strategies pass nil.
	var htfCandles []domain.Candle
	if strategyName == "reversion" {
		htfCandles, err = provider.Load(ctx, ticker, share.ID, enum.Hour4, dailyFrom, to, refresh)
		if err != nil {
			return err
		}
	}

	cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot, RiskFractionPct: riskPct}
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
		fmt.Println(domain.Trace(binding.Build(binding.DefaultParams()), candles, dailyCandles, htfCandles, cfg, target))
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	base := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s_%s", ticker, strategyName, interval.String(), stamp))

	if calibratePath != "" {
		return runCalibration(binding, calibratePath, candles, dailyCandles, htfCandles, cfg, metric, minTrades, testMonths, trainMonths, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg), from, to)
	}

	return runSingle(binding, paramsPath, candles, dailyCandles, htfCandles, cfg, periodDays, base,
		metaCommon(ticker, interval, from, to, cfg))
}

func runSingle(b svc.Binding, paramsPath string, candles []domain.Candle, dailyCandles, htfCandles []domain.Candle,
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

	res := domain.Run(b.Build(params), candles, dailyCandles, htfCandles, cfg)
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

func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, dailyCandles, htfCandles []domain.Candle,
	cfg domain.Config, metric string, minTrades, testMonths, trainMonths int, periodDays float64, base string, meta domain.Meta, from, to time.Time,
) error {
	if trainMonths > 0 {
		raw, err := os.ReadFile(gridPath)
		if err != nil {
			return fmt.Errorf("read grid: %w", err)
		}
		phases, err := svc.ParsePhases(raw)
		if err != nil {
			return err
		}
		summary, err := svc.RunWalkForward(b, phases, candles, dailyCandles, htfCandles, cfg, metric, minTrades, from, to, trainMonths, testMonths)
		if err != nil {
			return err
		}
		wfPath := base + "_walkforward.md"
		if err := writeFile(wfPath, svc.RenderWalkForwardMarkdown(meta.Ticker, metric, summary, trainMonths, testMonths)); err != nil {
			return err
		}
		fmt.Printf("walk-forward: %s (folds=%d, pooled PF=%.3f, compounded=%.2f%%)\n",
			wfPath, len(summary.Folds), summary.PooledOOS.ProfitFactor, summary.CompoundedReturnPct*100)
		return nil
	}

	raw, err := os.ReadFile(gridPath)
	if err != nil {
		return fmt.Errorf("read grid: %w", err)
	}
	phases, err := svc.ParsePhases(raw)
	if err != nil {
		return err
	}

	gridCandles, gridDaily, gridHTF := candles, dailyCandles, htfCandles
	bestCandles, bestDaily, bestHTF := candles, dailyCandles, htfCandles
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
		gridHTF, bestHTF = svc.SplitByTime(htfCandles, boundary)
		bestDays = testDays
		gridDays = periodDays - testDays
	}

	// Guard the in-sample (grid) window: shorter than the strategy lookback means the
	// engine returns early for every combo, silently yielding a 0-trade calibration that
	// looks like a result but ranks nothing. Fail loudly instead. The dominant lookback
	// term (e.g. SlowEMA) is not swept here, so the default params are representative.
	if lb := b.Build(b.DefaultParams()).Lookback(); len(gridCandles) < lb {
		return fmt.Errorf("in-sample window has %d candles, fewer than the strategy lookback %d: "+
			"every combo would trade 0 times — the calibration window is too short "+
			"(reduce -test-months or fetch more history; train window starts at the oldest available candle)",
			len(gridCandles), lb)
	}

	results, err := svc.RunPhases(b, phases, gridCandles, gridDaily, gridHTF, cfg, metric, minTrades, gridDays,
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
		res := domain.Run(b.Build(best), bestCandles, bestDaily, bestHTF, cfg)
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

func runScreen(ctx context.Context, client grpcclient.GrpcClient, tickers []string,
	interval enum.Interval, months int, outDir string, refresh bool,
) error {
	if len(tickers) == 0 {
		return fmt.Errorf("-screen: no tickers parsed")
	}
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	byTicker := make(map[string]shareInfo, len(shares))
	for _, s := range shares {
		byTicker[s.Ticker] = shareInfo{ID: s.ID, Lot: s.Lot}
	}
	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)

	var rows []svc.ScreenRow
	for _, ticker := range tickers {
		share, ok := byTicker[ticker]
		if !ok {
			rows = append(rows, svc.ScreenRow{Ticker: ticker, Note: "инструмент не найден"})
			continue
		}
		candles, err := provider.Load(ctx, ticker, share.ID, interval, from, to, refresh)
		if err != nil {
			return fmt.Errorf("%s: load candles: %w", ticker, err)
		}
		closes := make([]float64, len(candles))
		for i, c := range candles {
			closes[i] = c.Close
		}
		rets := domain.SimpleReturns(closes)
		if len(rets) < 9 {
			rows = append(rows, svc.ScreenRow{Ticker: ticker, Note: "мало свечей"})
			continue
		}
		vr2 := domain.VarianceRatio(rets, 2)
		rows = append(rows, svc.ScreenRow{
			Ticker: ticker, VR2: vr2,
			VR4:       domain.VarianceRatio(rets, 4),
			VR8:       domain.VarianceRatio(rets, 8),
			Autocorr1: domain.Autocorr1(rets),
			Verdict:   domain.MeanReversionVerdict(vr2),
		})
		fmt.Printf("screen %s: VR(2)=%.3f autocorr=%.3f\n", ticker, vr2, domain.Autocorr1(rets))
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	path := filepath.Join(outDir, fmt.Sprintf("screen_%s_%s.md", interval.String(), stamp))
	if err := writeFile(path, svc.RenderScreenMarkdown(rows)); err != nil {
		return err
	}
	fmt.Printf("screen report: %s (tickers=%d)\n", path, len(rows))
	return nil
}

// runVolRank ranks the liquid RUB share universe by Hour1 pullback-in-trend
// fitness over the last `months`, writing a markdown report. It fetches Hour1
// candles concurrently (volWorkers) — different tickers hit different cache
// files, so the only shared state guarded is the result slice.
func runVolRank(ctx context.Context, client grpcclient.GrpcClient, months, topN int,
	minTurnoverM, wRecov, wFreq, wLiq float64, sp svc.VolParams, outDir string, refresh bool,
) error {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	var universe []shareInfoT
	for _, s := range shares {
		if strings.EqualFold(s.Currency, "rub") && s.Trading {
			universe = append(universe, shareInfoT{Ticker: s.Ticker, Name: s.Name, ID: s.ID, Lot: s.Lot})
		}
	}
	if len(universe) == 0 {
		return fmt.Errorf("-volrank: no tradable RUB shares found")
	}

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)
	minBars := sp.EMAPeriod + sp.RecoverBars

	sem := semaphore.New(volWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.VolRow
	var done int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u shareInfoT) {
			defer wg.Done()
			defer sem.Release()
			candles, err := provider.Load(ctx, u.Ticker, u.ID, enum.Hour1, from, to, refresh)
			if err != nil {
				fmt.Printf("volrank %s: skip (load: %v)\n", u.Ticker, err)
				return
			}
			st := svc.VolMetrics(candles, u.Lot, sp)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("volrank [%d/%d] %s: recov=%.0f%% freq=%.2f turnover=%.0fM bars=%d\n",
				n, len(universe), u.Ticker, st.RecoveryRate*100, st.EventFreq, st.TurnoverM, st.Bars)
			if st.Bars < minBars || st.TurnoverM < minTurnoverM {
				return
			}
			mu.Lock()
			rows = append(rows, svc.VolRow{
				Ticker: u.Ticker, Name: u.Name,
				RecoveryRate: st.RecoveryRate, EventFreq: st.EventFreq, Events: st.Events,
				MeanATRpct: st.MeanATRpct, LastATRpct: st.LastATRpct, TurnoverM: st.TurnoverM,
				VR2: st.VR2, Autocorr1: st.Autocorr1, Bars: st.Bars,
			})
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	svc.ScoreVolRows(rows, wRecov, wFreq, wLiq)

	meta := svc.VolMeta{
		Months: months, ATRPeriod: sp.ATRPeriod, MinTurnover: minTurnoverM,
		RSIPeriod: sp.RSIPeriod, RSIOversold: sp.RSIOversold, RecoverBars: sp.RecoverBars, RecoverRSI: sp.RecoverRSI,
		WRecov: wRecov, WFreq: wFreq, WLiq: wLiq,
		Scanned: len(universe), Passed: len(rows),
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	path := filepath.Join(outDir, fmt.Sprintf("volatility_Hour1_%s.md", stamp))
	if err := writeFile(path, svc.RenderVolatilityMarkdown(rows, meta, topN)); err != nil {
		return err
	}
	fmt.Printf("volrank report: %s (scanned=%d passed=%d)\n", path, len(universe), len(rows))
	return nil
}

// shareInfoT carries the per-ticker data the volrank worker pool needs.
type shareInfoT struct {
	Ticker string
	Name   string
	ID     string
	Lot    int32
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
