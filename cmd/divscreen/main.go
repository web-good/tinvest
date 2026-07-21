// Command divscreen runs the dividend fundamental screener against the live
// Tinkoff Invest API and prints the ranking to stdout (no Telegram). It exists
// for diagnostics/calibration: unlike the bot command it exposes the raw
// fundamental fields (market cap, payout, net debt, yield) next to the composite
// and pillar scores, so anomalies (e.g. an illiquid thin-float name ranking high)
// can be inspected. All gRPC I/O is here; ranking stays in rank.Rank.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
)

const apiAddress = "invest-public-api.tinkoff.ru:443"

func main() {
	var (
		topN  = flag.Int("top", 20, "rows in the ranked table (0 = all survivors)")
		gated = flag.Bool("gated", false, "also print gated names grouped by reason")
		probe = flag.String("probe", "", "comma-separated tickers: dump full fundamentals + gate decision for each")
	)
	flag.Parse()
	logger.Init()

	if err := run(*topN, *gated, *probe); err != nil {
		log.Fatalf("divscreen: %v", err)
	}
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

// asset bundles the display metadata and fundamentals for one AssetUID.
type asset struct {
	ticker string
	name   string
	fund   *model.Fundamentals
}

func run(topN int, showGated bool, probeCSV string) error {
	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}

	ctx := context.Background()
	instr := client.InstrumentsServiceClient()

	shares, err := instr.Shares(ctx)
	if err != nil {
		return fmt.Errorf("fetch shares: %w", err)
	}

	// Same universe rule as the service: dividend payers with a non-empty asset,
	// grouped by AssetUID. Keep the first instrument per asset for display, and a
	// ticker->UID index (over ALL shares) so -probe can reach non-payers too.
	byUID := make(map[string]*asset)
	uidOrder := make([]string, 0)
	tickerToUID := make(map[string]string, len(shares))
	for _, sh := range shares {
		tickerToUID[strings.ToUpper(sh.Ticker)] = sh.AssetUID
		if !sh.DivYieldFlag || sh.AssetUID == "" {
			continue
		}
		if _, ok := byUID[sh.AssetUID]; !ok {
			byUID[sh.AssetUID] = &asset{ticker: sh.Ticker, name: sh.Name}
			uidOrder = append(uidOrder, sh.AssetUID)
		}
	}

	funds, err := instr.GetAssetFundamentals(ctx, uidOrder)
	if err != nil {
		return fmt.Errorf("fetch fundamentals: %w", err)
	}
	for _, f := range funds {
		if a := byUID[f.AssetUID]; a != nil {
			a.fund = f
		}
	}

	cfg := rank.DefaultConfig()
	scored := rank.Rank(funds, nil, cfg) // TODO(task 5/6): pass real sector map

	printRanked(scored, byUID, cfg, topN)
	if showGated {
		printGated(scored, byUID)
	}
	if probeCSV != "" {
		printProbe(probeCSV, tickerToUID, byUID, cfg)
	}
	return nil
}

func printRanked(scored []rank.ScoredCompany, byUID map[string]*asset, cfg rank.Config, topN int) {
	survivors := make([]rank.ScoredCompany, 0, len(scored))
	for _, sc := range scored {
		if sc.GateReason == "" {
			survivors = append(survivors, sc)
		}
	}
	if topN > 0 && len(survivors) > topN {
		survivors = survivors[:topN]
	}

	fmt.Printf("\n=== Ранжировано: %d (показано %d) ===\n", countSurvivors(scored), len(survivors))
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "#\tTicker\tName\tComp\tBonus\tSust\tSafe\tGrow\tQual\tVal\tMktCap,₽млрд\tFloat%\tPayout%\tND/EBITDA\tYield%")
	for i, sc := range survivors {
		a := byUID[sc.AssetUID]
		if a == nil || a.fund == nil {
			continue
		}
		f := a.fund
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%.0f\t+%d\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.1f\t%.0f\t%.0f\t%.2f\t%.2f\n",
			i+1, a.ticker, trunc(a.name, 22), sc.Composite, bonusFromScore(sc.Composite, cfg),
			sc.Sustainability, sc.Safety, sc.DivGrowth, sc.Quality, sc.Valuation,
			f.MarketCapitalization/1e9, f.FreeFloat*100, f.DividendPayoutRatioFy, f.NetDebtToEbitda, yieldOf(f))
	}
	_ = w.Flush()
}

func printGated(scored []rank.ScoredCompany, byUID map[string]*asset) {
	byReason := make(map[string][]string)
	for _, sc := range scored {
		if sc.GateReason == "" {
			continue
		}
		a := byUID[sc.AssetUID]
		label := sc.AssetUID
		if a != nil {
			label = a.ticker
		}
		byReason[sc.GateReason] = append(byReason[sc.GateReason], label)
	}
	reasons := make([]string, 0, len(byReason))
	for r := range byReason {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	fmt.Printf("\n=== Отсеяно воротами ===\n")
	for _, r := range reasons {
		names := byReason[r]
		sort.Strings(names)
		fmt.Printf("· %s (%d): %s\n", r, len(names), strings.Join(names, ", "))
	}
}

func printProbe(csv string, tickerToUID map[string]string, byUID map[string]*asset, cfg rank.Config) {
	fmt.Printf("\n=== Probe ===\n")
	for _, t := range strings.Split(csv, ",") {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		uid, ok := tickerToUID[t]
		if !ok {
			fmt.Printf("\n%s: тикер не найден среди Shares\n", t)
			continue
		}
		a := byUID[uid]
		if a == nil || a.fund == nil {
			fmt.Printf("\n%s: фундаментал не загружен (не дивидендная бумага или нет данных)\n", t)
			continue
		}
		f := a.fund
		reason, trap := rank.GateDecision(f, cfg)
		verdict := "проходит ворота"
		if reason != "" {
			verdict = "ОТСЕЯН: " + reason
		}
		fmt.Printf("\n%s — %s [%s]\n", t, a.name, verdict)
		if trap {
			fmt.Printf("  ⚠️ yield trap\n")
		}
		fmt.Printf("  MarketCap:        %.1f млрд ₽ (порог %.1f)\n", f.MarketCapitalization/1e9, cfg.MinMarketCap/1e9)
		fmt.Printf("  FreeFloat:        %.1f%%\n", f.FreeFloat*100)
		fmt.Printf("  AvgDailyVol 4w:   %.0f\n", f.AverageDailyVolumeLast4Weeks)
		fmt.Printf("  Yield:            %.2f%% (fwd %.2f / ttm %.2f)\n", yieldOf(f), f.ForwardAnnualDividendYield, f.DividendYieldDailyTtm)
		fmt.Printf("  Payout:           %.2f%%\n", f.DividendPayoutRatioFy)
		fmt.Printf("  NetDebt/EBITDA:   %.2f\n", f.NetDebtToEbitda)
		fmt.Printf("  FCF ttm:          %.0f\n", f.FreeCashFlowTtm)
		fmt.Printf("  ROIC / ROE:       %.2f / %.2f\n", f.Roic, f.Roe)
		fmt.Printf("  EV/EBITDA:        %.2f\n", f.EvToEbitdaMrq)
		fmt.Printf("  DivGrowth 5y:     %.2f\n", f.FiveYearAnnualDividendGrowthRate)
	}
}

func countSurvivors(scored []rank.ScoredCompany) int {
	n := 0
	for _, sc := range scored {
		if sc.GateReason == "" {
			n++
		}
	}
	return n
}

// bonusFromScore mirrors dividend.bonusFromScore (unexported) so the CLI shows
// the same +0..3 band the Golden X integration would apply.
func bonusFromScore(composite float64, cfg rank.Config) int {
	switch {
	case composite >= cfg.BonusScoreT3:
		return 3
	case composite >= cfg.BonusScoreT2:
		return 2
	case composite >= cfg.BonusScoreT1:
		return 1
	default:
		return 0
	}
}

func yieldOf(f *model.Fundamentals) float64 {
	if f.ForwardAnnualDividendYield > 0 {
		return f.ForwardAnnualDividendYield
	}
	return f.DividendYieldDailyTtm
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
