//go:build ignore

// Research-only fixed-snapshot replay. Uses the production engine and strategy unchanged.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	d "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	s "tinvest/internal/service/backtest"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
)

type Cohort struct {
	Trades int
	Net    float64
	PF     float64
}
type Row struct {
	Params       core.Params
	Metrics      d.Metrics
	Years        [2]Cohort
	Eligible     bool
	OpenPosition bool
	OpenBars     int
	ClosedPnL    float64
	Trades       []d.Trade
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(p string, v any) { b, e := os.ReadFile(p); must(e); must(json.Unmarshal(b, v)) }
func write(p string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(p, append(b, '\n'), 0644))
}
func main() {
	snapshot := flag.String("snapshot", "reports/fvg_imbalance/NLMK/research_2026-11-15/snapshot", "completed-candle snapshot")
	grid := flag.String("grid", "", "full nine-field grid")
	params := flag.String("params", "", "full parameter object")
	out := flag.String("out", "", "report directory")
	final := flag.Bool("final", false, "reserved final window, only after lock")
	flag.Parse()
	runtime.GOMAXPROCS(2)
	var bars, daily []d.Candle
	read(filepath.Join(*snapshot, "Minutes30.json"), &bars)
	read(filepath.Join(*snapshot, "Day1.json"), &daily)
	loc, e := time.LoadLocation("Europe/Moscow")
	must(e)
	from := time.Date(2023, 11, 15, 0, 0, 0, 0, loc)
	to := from.AddDate(2, 0, 0)
	if *final {
		from = to
		to = from.AddDate(1, 0, 0)
	}
	for i, c := range bars {
		if c.Time.Before(from) || !c.Time.Before(to) {
			panic(fmt.Sprintf("bar %d outside locked window", i))
		}
		if i > 0 && !c.Time.After(bars[i-1].Time) {
			panic("unsorted bars")
		}
	}
	cfg := d.Config{InitialCash: 100000, Fraction: 1, Commission: .0005, Lot: 10, EntryAtNextOpen: true}
	binding, e := s.FVGImbalanceLookupOrGeneric("NLMK", enum.Minutes30)
	must(e)
	var combos []core.Params
	if *params != "" {
		var p core.Params
		read(*params, &p)
		combos = append(combos, p)
	} else {
		raw, e := os.ReadFile(*grid)
		must(e)
		ph, e := s.ParsePhases(raw)
		must(e)
		if len(ph) != 1 {
			panic("only single-phase independent grids")
		}
		// Reflection-free expansion; preserve all nine explicit fields, never inherit defaults.
		g := ph[0].Grid
		keys := []string{"ATRPeriod", "MinGapATR", "MaxWaitBars", "StochPeriod", "UseStochExit", "StopATR", "TakeProfitATR", "HTFTrendEMA", "RequireEntryAboveFVGLower"}
		if len(g) != 9 {
			panic("grid requires all nine fields")
		}
		var expand func(int, map[string]float64)
		expand = func(i int, v map[string]float64) {
			if i == len(keys) {
				raw, e := json.Marshal(v)
				must(e)
				var p core.Params
				must(json.Unmarshal(raw, &p))
				must(core.ValidateParams(p))
				combos = append(combos, p)
				return
			}
			k := keys[i]
			if len(g[k]) == 0 {
				panic("missing " + k)
			}
			for _, x := range g[k] {
				v[k] = x
				expand(i+1, v)
			}
		}
		expand(0, map[string]float64{})
	}
	must(os.MkdirAll(*out, 0755))
	rows := make([]Row, len(combos))
	start := time.Now()
	for i, p := range combos {
		must(core.ValidateParams(p))
		r := d.Run(binding.Build(p), bars, daily, nil, cfg)
		m := d.Compute(r, r.BarsInMarket, len(r.Equity), to.Sub(from).Hours()/24)
		// JSON has no infinity; preserve profitable no-loss PF using maximum finite float.
		if math.IsInf(m.ProfitFactor, 1) {
			m.ProfitFactor = math.MaxFloat64
		}
		if math.IsInf(m.Sortino, 1) {
			m.Sortino = math.MaxFloat64
		}
		row := Row{Params: p, Metrics: m, Trades: r.Trades}
		closedBars := 0
		var gp, gl [2]float64
		for _, t := range r.Trades {
			j := 0
			if !t.EntryTime.Before(from.AddDate(1, 0, 0)) {
				j = 1
			}
			row.Years[j].Trades++
			row.Years[j].Net += t.PnL
			row.ClosedPnL += t.PnL
			closedBars += t.BarsHeld
			if t.PnL > 0 {
				gp[j] += t.PnL
			} else {
				gl[j] -= t.PnL
			}
		}
		for j := range row.Years {
			if gl[j] > 0 {
				row.Years[j].PF = gp[j] / gl[j]
			} else if gp[j] > 0 {
				row.Years[j].PF = math.MaxFloat64
			}
		}
		row.OpenBars = r.BarsInMarket - closedBars
		row.OpenPosition = row.OpenBars > 0
		row.Eligible = m.TotalTrades >= 20 && row.Years[0].Trades >= 10 && row.Years[1].Trades >= 10 && m.NetPnL > 0 && m.ProfitFactor > 1 && m.MaxDrawdownPct <= .10
		if *final {
			row.Eligible = m.TotalTrades >= 10 && m.NetPnL > 0 && m.ProfitFactor > 1 && m.MaxDrawdownPct <= .10
		}
		rows[i] = row
		if len(combos) == 1 {
			meta := d.Meta{Ticker: "NLMK", Interval: "Minutes30", From: from, To: to, InitialCash: cfg.InitialCash, Fraction: 1, Commission: cfg.Commission, Params: s.ParamRows(p), OpenPosition: row.OpenPosition}
			must(os.WriteFile(filepath.Join(*out, "report.md"), []byte(d.RenderMarkdown(meta, m, r.Trades, r.Equity)), 0644))
			must(os.WriteFile(filepath.Join(*out, "trades.csv"), []byte(d.RenderTradesCSV(r.Trades)), 0644))
			must(os.WriteFile(filepath.Join(*out, "equity.csv"), []byte(d.RenderEquityCSV(r.Equity)), 0644))
		}
		if (i+1)%10 == 0 {
			fmt.Printf("%d/%d elapsed %s\n", i+1, len(combos), time.Since(start).Round(time.Second))
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Metrics.Expectancy > rows[j].Metrics.Expectancy })
	write(filepath.Join(*out, "results.json"), rows)
	results := make([]s.CalibResult, len(rows))
	eligible := 0
	for i, r := range rows {
		results[i] = s.CalibResult{Params: r.Params, Metrics: r.Metrics, RecentTrades: r.Years[1].Trades}
		if r.Eligible {
			eligible++
		}
	}
	must(os.WriteFile(filepath.Join(*out, "calibration.md"), []byte(s.RenderCalibrationMarkdown("expectancy", results, 0)), 0644))
	fmt.Printf("DONE %d combos, eligible %d, elapsed %s\n", len(rows), eligible, time.Since(start).Round(time.Second))
}
