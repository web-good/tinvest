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
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x"
	"tinvest/internal/service/trading_strategy/golden_x/backtest"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/service/trading_strategy/golden_x/shares"
	"tinvest/internal/utils"
	"tinvest/pkg/client/grpc"
	"tinvest/pkg/collection"
)

func main() {
	var (
		kindStr   = flag.String("kind", "Dividend", "Dividend | Growth")
		fromStr   = flag.String("from", "2022-01-01", "inclusive start date (MSK)")
		toStr     = flag.String("to", "", "inclusive end date (default: today)")
		sharesCSV = flag.String("shares", "", "comma-separated share IDs (default: all)")
		refresh   = flag.Bool("refresh", false, "force re-fetch from Tinkoff API")
		outDir    = flag.String("out-dir", "cache/backtests", "report output directory")
		cacheDir  = flag.String("cache-dir", "cache/candles", "candle cache directory")
	)
	flag.Parse()

	kind, err := parseKind(*kindStr)
	if err != nil {
		log.Fatal(err)
	}
	msk, _ := time.LoadLocation("Europe/Moscow")
	from, err := time.ParseInLocation("2006-01-02", *fromStr, msk)
	if err != nil {
		log.Fatalf("parse --from: %v", err)
	}
	to := time.Now().In(msk)
	if *toStr != "" {
		to, err = time.ParseInLocation("2006-01-02", *toStr, msk)
		if err != nil {
			log.Fatalf("parse --to: %v", err)
		}
	}

	list := selectShareList(kind, *sharesCSV)
	if len(list.All()) == 0 {
		log.Fatalf("no shares selected")
	}

	// Token only needed if any share misses cache or --refresh is set.
	_ = godotenv.Load("env/local.env")
	_ = godotenv.Load("env/token.env")
	token := os.Getenv("T_BANK")
	var grpcClient grpc.GrpcClient
	needsFetch := *refresh || anyCacheMiss(*cacheDir, list)
	if needsFetch {
		if token == "" {
			log.Fatalf("T_BANK env var required (no cache and not --refresh)")
		}
		grpcClient, err = grpc.NewClientGrpc("invest-public-api.tinkoff.ru:443", token)
		if err != nil {
			log.Fatalf("grpc dial: %v", err)
		}
	}

	ctx := context.Background()
	report := backtest.Report{Kind: kind, From: from, To: to, PerShare: map[string]backtest.Stats{}}
	useTrendFilter := kind == dto.StrategyKindGrowth

	fetcher := newGrpcFetcher(grpcClient, to)
	cache := backtest.NewCache(*cacheDir, fetcher, *refresh)
	settings := golden_x.DefaultSettings()

	for _, instr := range list.All() {
		raw, ferr := cache.Get(ctx, instr.ID)
		if ferr != nil {
			if *refresh {
				log.Fatalf("fetch %s: %v", instr.Name, ferr)
			}
			log.Printf("WARN %s: %v (skipping; rerun with --refresh to retry)", instr.Name, ferr)
			continue
		}
		closed := filterClosed(raw, msk)
		closed = trimByDateRange(closed, from, to)
		startIdx := chooseStartIdx(instr.RSILength)
		if len(closed) <= startIdx {
			continue
		}
		trades := backtest.Replay(instr.ID, closed,
			func(c []*model.CandleItemTechAnalyse, period int, k dto.StrategyKind, uft bool, s dto.Settings) (dto.Signal, error) {
				return golden_x.Detect(c, period, k, uft, s)
			},
			backtest.ReplayConfig{
				Kind: kind, RSIPeriod: instr.RSILength, StartIdx: startIdx,
				MaxWeeks: 52, UseTrendFilter: useTrendFilter, Settings: settings,
			})
		report.Trades = append(report.Trades, trades...)
		report.PerShare[instr.ID] = backtest.AggregateStats(trades)
	}
	report.Overall = backtest.AggregateStats(report.Trades)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	stamp := time.Now().Format("2006-01-02_1504")
	path := filepath.Join(*outDir, fmt.Sprintf("%s_%s.md", stamp, kindLabel(kind)))
	if err := os.WriteFile(path, []byte(backtest.RenderMarkdown(report)), 0o644); err != nil {
		log.Fatalf("write report: %v", err)
	}

	// Console summary.
	fmt.Println(consoleSummary(report))
	fmt.Printf("\nReport written to: %s\n", path)
}

func parseKind(s string) (dto.StrategyKind, error) {
	switch strings.ToLower(s) {
	case "dividend", "gold":
		return dto.StrategyKindDividend, nil
	case "growth":
		return dto.StrategyKindGrowth, nil
	default:
		return 0, fmt.Errorf("unknown --kind %q (expected Dividend|Growth)", s)
	}
}

func kindLabel(k dto.StrategyKind) string {
	if k == dto.StrategyKindGrowth {
		return "Growth"
	}
	return "Dividend"
}

func selectShareList(kind dto.StrategyKind, csv string) *collection.InstrumentCollection {
	base := shares.Dividend()
	if kind == dto.StrategyKindGrowth {
		base = shares.Growth()
	}
	if csv == "" {
		return base
	}
	want := map[string]bool{}
	for _, id := range strings.Split(csv, ",") {
		want[strings.TrimSpace(id)] = true
	}
	out := collection.NewCollection()
	for _, instr := range base.All() {
		if want[instr.ID] {
			out.Add(instr)
		}
	}
	return out
}

func anyCacheMiss(dir string, list *collection.InstrumentCollection) bool {
	for _, instr := range list.All() {
		if _, err := os.Stat(filepath.Join(dir, instr.ID+"_W.json")); os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// grpcFetcher implements backtest.CandleFetcher using the existing Tinkoff
// gRPC client. It pulls up to ~5 years of weekly candles in 300-candle
// chunks (Tinkoff's per-call cap for weekly is 300).
type grpcFetcher struct {
	client grpc.GrpcClient
	to     time.Time
}

func newGrpcFetcher(client grpc.GrpcClient, to time.Time) backtest.CandleFetcher {
	return &grpcFetcher{client: client, to: to}
}

func (g *grpcFetcher) Fetch(ctx context.Context, shareID string) ([]*model.CandleItemTechAnalyse, error) {
	// One call is enough at weekly interval: 300 candles × 7 days ≈ 5.7 years.
	limit := int32(300)
	from := utils.TimeStampPbGenerator(g.to, -300, enum.Week1)
	return g.client.MarketDataServiceClient().GetCandles(
		ctx, &shareID, enum.Week1.ToNumberInvestApi(),
		from, timestamppb.New(g.to), &limit, true,
	)
}

func filterClosed(candles []*model.CandleItemTechAnalyse, loc *time.Location) []*model.CandleItemTechAnalyse {
	now := time.Now().In(loc)
	// Reuse the same Monday-00:00 cutoff prod uses.
	weekday := (int(now.Weekday()) + 6) % 7
	y, m, d := now.Date()
	cutoff := time.Date(y, m, d-weekday, 0, 0, 0, 0, loc)
	out := make([]*model.CandleItemTechAnalyse, 0, len(candles))
	for _, c := range candles {
		if c != nil && c.Time.Before(cutoff) {
			out = append(out, c)
		}
	}
	return out
}

func trimByDateRange(candles []*model.CandleItemTechAnalyse, from, to time.Time) []*model.CandleItemTechAnalyse {
	out := make([]*model.CandleItemTechAnalyse, 0, len(candles))
	for _, c := range candles {
		if c.Time.Before(from) || c.Time.After(to) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// chooseStartIdx leaves enough head-room for the adaptive RSI window plus
// the EMA200 warmup the trend filter needs (≈200 closed weeks for Growth).
func chooseStartIdx(rsiPeriod int) int {
	const adaptiveWindowMin = 100
	const emaWarmup = 200
	if emaWarmup > rsiPeriod+adaptiveWindowMin {
		return emaWarmup
	}
	return rsiPeriod + adaptiveWindowMin
}

func consoleSummary(r backtest.Report) string {
	var b strings.Builder
	b.WriteString("## Overall\n\n")
	// Inline format mirroring report.writeStatsTable.
	s := r.Overall
	fmt.Fprintf(&b, "Count=%d Wins=%d Losses=%d Open=%d WinRate=%.2f%% Cumulative=%.2f%% MaxDD=%.2f%% AvgWeeks=%.1f\n\n",
		s.Count, s.Wins, s.Losses, s.Open, s.WinRate*100, s.CumulativeReturn, s.MaxDrawdown, s.AvgWeeksHeld)
	b.WriteString("## Exit reasons\n\n")
	counts := map[backtest.ExitReason]int{}
	sums := map[backtest.ExitReason]float64{}
	for _, tr := range r.Trades {
		counts[tr.ExitReason]++
		sums[tr.ExitReason] += tr.ReturnPct
	}
	for reason, n := range counts {
		fmt.Fprintf(&b, "%-10s count=%d avg=%.2f%%\n", reason, n, sums[reason]/float64(n))
	}
	return b.String()
}
