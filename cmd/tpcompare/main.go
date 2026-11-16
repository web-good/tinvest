// Command tpcompare compares RSI_PULLBACK TP fill assumptions using local caches only.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	domain "tinvest/internal/domain/backtest"
	svc "tinvest/internal/service/backtest"
	live "tinvest/internal/service/trading_strategy/rsi_pullback/live"
	core "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

type input struct {
	params         core.Params
	candles, daily []domain.Candle
	hashes         [2]string
}

func loadWindow(path string, from, to time.Time) ([]domain.Candle, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var all []domain.Candle
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, "", err
	}
	var out []domain.Candle
	for i, c := range all {
		if i > 0 && !c.Time.After(all[i-1].Time) {
			return nil, "", fmt.Errorf("%s: unsorted or duplicate candles", path)
		}
		if !c.Time.Before(from) && c.Time.Before(to) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, "", fmt.Errorf("%s: empty requested window", path)
	}
	return out, fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}

func prepare(dir, ticker string, from, to time.Time) (input, error) {
	p, ok := live.ParamsFor(ticker)
	if !ok {
		return input{}, fmt.Errorf("%s: no current live literal", ticker)
	}
	in := input{params: p}
	var err error
	in.candles, in.hashes[0], err = loadWindow(filepath.Join(dir, ticker+"_Minutes30.json"), from, to)
	if err != nil {
		return input{}, err
	}
	in.daily, in.hashes[1], err = loadWindow(filepath.Join(dir, ticker+"_Day1.json"), from.AddDate(-1, 0, 0), to)
	if err != nil {
		return input{}, err
	}
	// A weekend may separate the final session from the requested boundary.
	if to.Sub(in.candles[len(in.candles)-1].Time) > 7*24*time.Hour {
		return input{}, fmt.Errorf("%s: M30 cache stale (last %s)", ticker, in.candles[len(in.candles)-1].Time.Format(time.RFC3339))
	}
	if to.Sub(in.daily[len(in.daily)-1].Time) > 7*24*time.Hour {
		return input{}, fmt.Errorf("%s: daily cache stale", ticker)
	}
	if len(in.candles) < core.NewWithParams(ticker, p).Lookback() {
		return input{}, fmt.Errorf("%s: insufficient M30 lookback", ticker)
	}
	// The first Decide must have enough completed weekday dailies for ATR.
	// Otherwise a recent but truncated daily cache silently suppresses entries.
	decision := in.candles[core.NewWithParams(ticker, p).Lookback()-1].Time.In(time.FixedZone("MSK", 3*3600))
	day := time.Date(decision.Year(), decision.Month(), decision.Day(), 0, 0, 0, 0, decision.Location())
	warmup := 0
	for _, c := range in.daily {
		d := c.Time.In(day.Location())
		if d.Before(day) && d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			warmup++
		}
	}
	if warmup < p.DailyATRPeriod+1 {
		return input{}, fmt.Errorf("%s: insufficient daily warmup at first decision (%d weekday candles, need %d)", ticker, warmup, p.DailyATRPeriod+1)
	}
	return in, nil
}

func main() {
	dir := flag.String("cache", "data/candles", "local candle cache, including shares.json")
	tickers := flag.String("tickers", "", "comma-separated current live tickers (required)")
	end := flag.String("end", "", "exclusive end RFC3339 (required, fixes the experiment window)")
	months := flag.Int("months", 24, "requested window in months")
	out := flag.String("out", "reports/rsi_pullback_tp", "output directory")
	cash := flag.Float64("cash", 100000, "independent starting cash per ticker and scenario")
	fraction := flag.Float64("fraction", 1, "cash fraction per entry")
	commission := flag.Float64("commission", 0.0005, "commission fraction per side")
	flag.Parse()
	runtime.GOMAXPROCS(2)
	to, err := time.Parse(time.RFC3339, *end)
	if err != nil || *tickers == "" || *months <= 0 || math.IsNaN(*cash) || math.IsInf(*cash, 0) || *cash <= 0 || math.IsNaN(*fraction) || *fraction <= 0 || *fraction > 1 || math.IsNaN(*commission) || *commission < 0 || *commission >= 1 {
		fmt.Fprintln(os.Stderr, "invalid arguments: require -tickers, RFC3339 -end, positive months/cash, fraction in (0,1], commission in [0,1)")
		os.Exit(1)
	}
	if err := compare(*dir, *tickers, to.AddDate(0, -*months, 0), to, *out, *cash, *fraction, *commission); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func compare(dir, tickerCSV string, from, to time.Time, out string, cash, fraction, commission float64) (retErr error) {
	raw, err := os.ReadFile(filepath.Join(dir, "shares.json"))
	if err != nil {
		return err
	}
	var shares map[string]struct{ Lot int32 }
	if err := json.Unmarshal(raw, &shares); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(out, "comparison.csv"))
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); retErr == nil {
			retErr = err
		}
	}()
	w := csv.NewWriter(file)
	write := func(row []string) error { return w.Write(row) }
	if err := write([]string{"ticker", "mode", "slippage_bps", "trades", "tp_exits", "net_pnl", "profit_factor", "max_drawdown_pct", "first_m30", "last_m30", "first_daily", "lot", "short_history", "open_position"}); err != nil {
		return err
	}
	var summary strings.Builder
	fmt.Fprintf(&summary, "# RSI_PULLBACK TP comparison\n\nRequested window [%s, %s).\nIndependent accounts: cash %.2f; fraction %g; commission %g per side.\n", from.Format(time.RFC3339), to.Format(time.RFC3339), cash, fraction, commission)
	summary.WriteString("\nSignals and ticker literals unchanged. touch models target/gap fills; bar-close approximates delayed live market execution. 0/5/10 bps adverse TP slippage are illustrative assumptions, not measured costs. Intrabar touch order is unknown; existing strategy stop priorities are preserved. No recalibration. This is a retrospective execution comparison, not out-of-sample validation. Independent account PnLs are not a portfolio return. Cached lots are used as-is; historical lot changes and broker tick rounding are not modeled. Daily warmup limitations and stale caches are listed below.\n")
	manifest := map[string]any{"from": from, "to_exclusive": to, "cash": cash, "fraction": fraction, "commission": commission, "arguments": os.Args[1:], "go_version": runtime.Version(), "shares_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "inputs": map[string]any{}}
	inputs := manifest["inputs"].(map[string]any)
	names := strings.Split(strings.ToUpper(tickerCSV), ",")
	sort.Strings(names)
	seen := map[string]bool{}
	completed := 0
	for _, name := range names {
		ticker := strings.TrimSpace(name)
		if ticker == "" || seen[ticker] {
			continue
		}
		seen[ticker] = true
		in, err := prepare(dir, ticker, from, to)
		if err != nil {
			fmt.Fprintf(&summary, "\n- SKIPPED %s: %s\n", ticker, err)
			fmt.Printf("skip %s: %v\n", ticker, err)
			continue
		}
		lot := shares[ticker].Lot
		if lot <= 0 {
			fmt.Fprintf(&summary, "\n- SKIPPED %s: missing lot size\n", ticker)
			continue
		}
		short := in.candles[0].Time.Sub(from) > 7*24*time.Hour
		inputs[ticker] = map[string]any{"m30_sha256": in.hashes[0], "daily_sha256": in.hashes[1], "params": in.params, "lot": lot, "first_m30": in.candles[0].Time, "last_m30": in.candles[len(in.candles)-1].Time, "first_daily": in.daily[0].Time}
		fmt.Fprintf(&summary, "\n- %s: M30 %s — %s; first daily %s; lot %d; short M30 history: %t.\n", ticker, in.candles[0].Time.Format(time.RFC3339), in.candles[len(in.candles)-1].Time.Format(time.RFC3339), in.daily[0].Time.Format(time.RFC3339), lot, short)
		for _, mode := range []string{"touch", "bar-close"} {
			for _, bps := range []float64{0, 5, 10} {
				cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: lot, TPExecution: domain.TPExecution{Mode: mode, SlippageBPS: bps}}
				res := domain.Run(core.NewWithParams(ticker, in.params), in.candles, in.daily, nil, cfg)
				m := domain.Compute(res, res.BarsInMarket, len(res.Equity), to.Sub(from).Hours()/24)
				nTP := 0
				for _, tr := range res.Trades {
					if tr.Reason == "TP" {
						nTP++
					}
				}
				heldBars := 0
				for _, tr := range res.Trades {
					heldBars += tr.BarsHeld
				}
				open := res.BarsInMarket > heldBars
				base := filepath.Join(out, fmt.Sprintf("%s_%s_%gbps", ticker, mode, bps))
				meta := domain.Meta{Ticker: ticker, Interval: "Minutes30", From: from, To: to, InitialCash: cash, Fraction: fraction, Commission: commission, ExecutionAssumptions: cfg.TPExecution.Description(), Params: svc.ParamRows(in.params), OpenPosition: open}
				for suffix, content := range map[string]string{".md": domain.RenderMarkdown(meta, m, res.Trades, res.Equity), "_trades.csv": domain.RenderTradesCSV(res.Trades), "_equity.csv": domain.RenderEquityCSV(res.Equity)} {
					if err := os.WriteFile(base+suffix, []byte(content), 0644); err != nil {
						return err
					}
				}
				row := []string{ticker, mode, num(bps), strconv.Itoa(m.TotalTrades), strconv.Itoa(nTP), num(m.NetPnL), num(m.ProfitFactor), num(m.MaxDrawdownPct * 100), in.candles[0].Time.Format(time.RFC3339), in.candles[len(in.candles)-1].Time.Format(time.RFC3339), in.daily[0].Time.Format(time.RFC3339), strconv.Itoa(int(lot)), strconv.FormatBool(short), strconv.FormatBool(open)}
				if err := write(row); err != nil {
					return err
				}
				fmt.Printf("%s %s %gbps trades=%d TP=%d net=%.2f PF=%.3f DD=%.2f%%\n", ticker, mode, bps, m.TotalTrades, nTP, m.NetPnL, m.ProfitFactor, m.MaxDrawdownPct*100)
			}
		}
		completed++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	if completed == 0 {
		return fmt.Errorf("no eligible ticker histories; see skipped inputs")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "manifest.json"), data, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "README.md"), []byte(summary.String()), 0644)
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }
