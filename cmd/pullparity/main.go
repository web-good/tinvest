// Command pullparity proves that the live rsi_pullback MarketData assembler produces the
// same snapshot — and therefore the same Decide verdict — as the backtest engine's own
// per-bar assembly. It replays cached candles bar by bar and builds each snapshot twice:
// once the way internal/domain/backtest.Run does it, once through the live
// marketdata.Assemble fed by a cache-backed CandleClient that only ever returns candles
// visible at that bar's close. Any divergence is a live/backtest fidelity bug.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/marketdata"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/utils"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
)

const apiAddress = "invest-public-api.tinkoff.ru:443"

// maxDailyHorizonMonths is the largest -months the comparison stays honest for. The live
// assembler's daily fetch (marketdata.dailyFetchDays = 730 calendar days) always covers
// [asOf-730d, asOf]; the reference side has no such cap and sees every daily candle loaded
// for the run. As long as -months <= 24 (~730 days) the live window's lower bound never
// falls after `from`, so both sides draw on the exact same daily candles. Beyond that,
// early bars' reference snapshots see dailies the live window has already aged out of —
// an expected, documented gap in this TOOL, not a bug in the assembler.
const maxDailyHorizonMonths = 24

func main() {
	var (
		tickersCSV = flag.String("tickers", "UGLD,T,GAZP", "comma-separated tickers to check")
		months     = flag.Int("months", 24, "lookback period in months")
		examples   = flag.Int("examples", 10, "max diverging lines printed per ticker")
		cacheDir   = flag.String("cache", "data/candles", "candle cache directory")
	)
	flag.Parse()
	logger.Init()

	if err := run(context.Background(), runCfg{
		tickers:  splitCSV(*tickersCSV),
		months:   *months,
		examples: *examples,
		cacheDir: *cacheDir,
	}); err != nil {
		log.Fatalf("pullparity: %v", err)
	}
}

type runCfg struct {
	tickers          []string
	months, examples int
	cacheDir         string
}

func run(ctx context.Context, cfg runCfg) error {
	if len(cfg.tickers) == 0 {
		return fmt.Errorf("-tickers must not be empty")
	}
	if cfg.months > maxDailyHorizonMonths {
		fmt.Printf("pullparity: -months %d exceeds %d — Daily* divergences on the earliest bars of the run "+
			"are expected (the live assembler's dailyFetchDays window ages out before the reference sees the "+
			"same history), not a defect in the assembler; see maxDailyHorizonMonths\n", cfg.months, maxDailyHorizonMonths)
	}

	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	ids, err := resolveInstrumentIDs(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cfg.cacheDir)

	anyDiff := false
	for _, ticker := range cfg.tickers {
		st, ok := live.StrategyFor(ticker)
		if !ok {
			return fmt.Errorf("%s: not registered in live.StrategyFor — add calibrated params before checking parity", ticker)
		}
		diffs, bars, err := checkTicker(ctx, provider, ticker, ids[ticker], st, from, to)
		if err != nil {
			return fmt.Errorf("%s: %w", ticker, err)
		}
		if len(diffs) > 0 {
			anyDiff = true
		}
		fmt.Printf("%s: %d баров, %d расхождений\n", ticker, bars, len(diffs))
		for i, d := range diffs {
			if i >= cfg.examples {
				break
			}
			fmt.Printf("  %s\n", d)
		}
	}
	if anyDiff {
		return fmt.Errorf("divergences found between the live assembler and the backtest engine")
	}
	return nil
}

// checkTicker replays m30 bar by bar from lookback-1 onward, building the MarketData
// snapshot (and the resulting Decide verdict) twice per bar — once via the engine's own
// AssembleMarketData/TodayExtent, once via marketdata.Assemble fed by a cache-backed
// client that only ever sees candles visible at that bar's close. bars is the number of
// bars actually replayed (len(m30) - lookback + 1).
func checkTicker(ctx context.Context, provider *svc.CandleProvider, ticker, instrumentUID string,
	st *core.Strategy, from, to time.Time) (diffs []string, bars int, err error) {

	lookback := st.Lookback()
	m30, err := provider.Load(ctx, ticker, instrumentUID, enum.Minutes30, from, to, false)
	if err != nil {
		return nil, 0, fmt.Errorf("load 30m: %w", err)
	}
	day, err := provider.Load(ctx, ticker, instrumentUID, enum.Day1, from, to, false)
	if err != nil {
		return nil, 0, fmt.Errorf("load daily: %w", err)
	}
	if len(m30) < lookback {
		return nil, 0, fmt.Errorf("only %d cached 30m candles, need at least lookback %d", len(m30), lookback)
	}

	for i := lookback - 1; i < len(m30); i++ {
		cur := m30[i].Time
		bars++

		want := backtest.AssembleMarketData(m30[i-lookback+1:i+1], day, nil, cur)
		want.TodayHigh, want.TodayLow = backtest.TodayExtent(m30, i)

		// asOf is the instant bar i has just closed: the moment a live runner would wake
		// up and act on it.
		asOf := cur.Add(30 * time.Minute)
		fake := newCacheClient(m30, day, asOf)
		got, aerr := marketdata.Assemble(ctx, fake, instrumentUID, lookback, asOf)
		if aerr != nil {
			// The engine built a snapshot on this bar; the live path refusing to is itself
			// a divergence worth reporting, not a reason to skip the bar.
			diffs = append(diffs, fmt.Sprintf("%s: Assemble: %v", cur, aerr))
			continue
		}

		lines := diffMarketData(want, got)
		lines = append(lines, diffSignal(st.Decide(want), st.Decide(got))...)
		for _, l := range lines {
			diffs = append(diffs, fmt.Sprintf("%s: %s", cur, l))
		}
	}
	return diffs, bars, nil
}

// cacheClient implements candles.CandleClient over in-memory slices already loaded by the
// candle provider. It returns exactly what a live GetCandles call would have returned had
// it been made at asOf: bars whose Time falls in the requested [from, to] AND strictly
// before asOf, marked complete. Feeding it the SAME m30/day slices the reference assembly
// reads means any divergence traces to the assembly logic, not to different underlying data.
type cacheClient struct {
	m30, day []backtest.Candle
	asOf     time.Time
}

var _ candles.CandleClient = (*cacheClient)(nil)

func newCacheClient(m30, day []backtest.Candle, asOf time.Time) *cacheClient {
	return &cacheClient{m30: m30, day: day, asOf: asOf}
}

func (c *cacheClient) GetCandles(_ context.Context, _ *string, interval int32,
	from, to *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {

	var src []backtest.Candle
	switch interval {
	case enum.Minutes30.ToNumberInvestAPI():
		src = c.m30
	case enum.Day1.ToNumberInvestAPI():
		src = c.day
	default:
		return nil, fmt.Errorf("pullparity: cacheClient: unexpected interval %d", interval)
	}

	fromT, toT := from.AsTime(), to.AsTime()
	out := make([]*imodel.CandleItemTechAnalyse, 0, len(src))
	for _, cd := range src {
		if cd.Time.Before(fromT) || cd.Time.After(toT) || !cd.Time.Before(c.asOf) {
			continue
		}
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time:       cd.Time,
			Open:       toQuotation(cd.Open),
			Close:      toQuotation(cd.Close),
			Low:        toQuotation(cd.Low),
			High:       toQuotation(cd.High),
			Volume:     cd.Volume,
			IsComplete: true,
		})
	}
	return out, nil
}

func toQuotation(price float64) imodel.Quotation {
	units, nano := utils.SplitPrice(price)
	return imodel.Quotation{Units: units, Nano: nano}
}

// diffMarketData compares every field Decide can read, returning one line per divergence.
// Length mismatches on a slice short-circuit that slice's element-wise comparison (there is
// nothing meaningful to pair up index-for-index) but do not stop the rest of the fields
// from being checked.
func diffMarketData(want, got strategy.MarketData) []string {
	var out []string
	out = append(out, diffFloat("Price", want.Price, got.Price)...)
	out = append(out, diffFloats("Closes", want.Closes, got.Closes)...)
	out = append(out, diffFloats("Highs", want.Highs, got.Highs)...)
	out = append(out, diffFloats("Lows", want.Lows, got.Lows)...)
	out = append(out, diffInts("Volumes", want.Volumes, got.Volumes)...)
	out = append(out, diffTimes("Times", want.Times, got.Times)...)
	out = append(out, diffFloats("DailyCloses", want.DailyCloses, got.DailyCloses)...)
	out = append(out, diffFloats("DailyHighs", want.DailyHighs, got.DailyHighs)...)
	out = append(out, diffFloats("DailyLows", want.DailyLows, got.DailyLows)...)
	out = append(out, diffTimes("DailyTimes", want.DailyTimes, got.DailyTimes)...)
	out = append(out, diffFloat("TodayHigh", want.TodayHigh, got.TodayHigh)...)
	out = append(out, diffFloat("TodayLow", want.TodayLow, got.TodayLow)...)
	return out
}

// diffSignal compares the fields a fill/risk decision actually depends on: whether the
// live and backtest snapshots would place the same order at the same stop/target. Fields
// such as Price/RSI/Level are derived display data, not trade-relevant.
func diffSignal(want, got model.Signal) []string {
	var out []string
	if want.Kind != got.Kind {
		out = append(out, fmt.Sprintf("Kind: want %v, got %v", want.Kind, got.Kind))
	}
	if want.Reason != got.Reason {
		out = append(out, fmt.Sprintf("Reason: want %q, got %q", want.Reason, got.Reason))
	}
	if want.StopLoss != got.StopLoss {
		out = append(out, fmt.Sprintf("StopLoss: want %.4f, got %.4f", want.StopLoss, got.StopLoss))
	}
	if want.TakeProfit != got.TakeProfit {
		out = append(out, fmt.Sprintf("TakeProfit: want %.4f, got %.4f", want.TakeProfit, got.TakeProfit))
	}
	if want.ATR != got.ATR {
		out = append(out, fmt.Sprintf("ATR: want %.4f, got %.4f", want.ATR, got.ATR))
	}
	return out
}

func diffFloat(name string, want, got float64) []string {
	if want != got {
		return []string{fmt.Sprintf("%s: want %.4f, got %.4f", name, want, got)}
	}
	return nil
}

func diffFloats(name string, want, got []float64) []string {
	if len(want) != len(got) {
		return []string{fmt.Sprintf("%s: length want %d, got %d", name, len(want), len(got))}
	}
	var out []string
	for i := range want {
		if want[i] != got[i] {
			out = append(out, fmt.Sprintf("%s[%d]: want %.4f, got %.4f", name, i, want[i], got[i]))
		}
	}
	return out
}

func diffInts(name string, want, got []int64) []string {
	if len(want) != len(got) {
		return []string{fmt.Sprintf("%s: length want %d, got %d", name, len(want), len(got))}
	}
	var out []string
	for i := range want {
		if want[i] != got[i] {
			out = append(out, fmt.Sprintf("%s[%d]: want %d, got %d", name, i, want[i], got[i]))
		}
	}
	return out
}

func diffTimes(name string, want, got []time.Time) []string {
	if len(want) != len(got) {
		return []string{fmt.Sprintf("%s: length want %d, got %d", name, len(want), len(got))}
	}
	var out []string
	for i := range want {
		if !want[i].Equal(got[i]) {
			out = append(out, fmt.Sprintf("%s[%d]: want %s, got %s", name, i, want[i], got[i]))
		}
	}
	return out
}

// contains reports whether s contains substr. A thin wrapper so callers outside this file
// need not import strings just to grep a diff line.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// resolveInstrumentIDs looks up the instrument UID for every requested ticker in one
// Shares() call, erroring out (rather than silently skipping) if any ticker is not found —
// the whole point of this tool is a trustworthy comparison, not a best-effort one.
func resolveInstrumentIDs(ctx context.Context, client grpcclient.GrpcClient, tickers []string) (map[string]string, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("load shares: %w", err)
	}
	byTicker := make(map[string]string, len(shares))
	for _, s := range shares {
		byTicker[strings.ToUpper(s.Ticker)] = s.ID
	}
	out := make(map[string]string, len(tickers))
	for _, t := range tickers {
		id, ok := byTicker[strings.ToUpper(t)]
		if !ok {
			return nil, fmt.Errorf("%s: instrument not found", t)
		}
		out[t] = id
	}
	return out, nil
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
