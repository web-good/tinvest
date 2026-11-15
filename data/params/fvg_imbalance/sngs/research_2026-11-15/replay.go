//go:build ignore

// Run explicitly: go run replay.go -grid ... -out ...; uses production replay engine.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"time"

	d "tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/trading_strategy/scalping/model"
	s "tinvest/internal/service/trading_strategy/scalping/strategy"
)

type observer struct {
	s.Strategy
	Open *s.Position
}

func (o *observer) Decide(md s.MarketData) model.Signal {
	sig := o.Strategy.Decide(md)
	o.Open = md.Position
	if sig.Kind == model.SignalSell {
		o.Open = nil
	}
	return sig
}

type row struct {
	Params          any
	Metrics         map[string]any
	AnnualTrades    [2]int
	AnnualClosedPnL [2]float64
	AnnualPF        [2]string
	Eligible        bool
	OpenPosition    *s.Position
	OpenHours       float64
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func write(path string, v any) {
	b, e := json.MarshalIndent(v, "", "  ")
	must(e)
	must(os.WriteFile(path, append(b, '\n'), 0644))
}
func load(path string, v any) { b, e := os.ReadFile(path); must(e); must(json.Unmarshal(b, v)) }
func slice(a []d.Candle, from, to time.Time) []d.Candle {
	out := []d.Candle{}
	for _, c := range a {
		if !c.Time.Before(from) && c.Time.Before(to) {
			out = append(out, c)
		}
	}
	return out
}
func main() {
	grid := flag.String("grid", "", "full nine-field grid")
	params := flag.String("params", "", "full numeric params, absent means registry")
	out := flag.String("out", "", "report directory")
	start := flag.String("start", "2023-11-15T00:00:00+03:00", "fixed start")
	end := flag.String("end", "2025-11-15T00:00:00+03:00", "fixed end")
	snapshots := flag.String("snapshots", "reports/fvg_imbalance/sngs/research_2026-11-15", "snapshot directory")
	flag.Parse()
	from, e := time.Parse(time.RFC3339, *start)
	must(e)
	to, e := time.Parse(time.RFC3339, *end)
	must(e)
	split := from.AddDate(1, 0, 0)
	var bars, daily []d.Candle
	load(filepath.Join(*snapshots, "snapshot_Minutes30.json"), &bars)
	load(filepath.Join(*snapshots, "snapshot_Day1.json"), &daily)
	bars = slice(bars, from, to)
	daily = slice(daily, from.AddDate(-1, 0, 0), to)
	if len(bars) == 0 {
		panic("no bars")
	}
	b, e := svc.FVGImbalanceLookupOrGeneric("SNGS", enum.Minutes30)
	must(e)
	combos := []any{b.DefaultParams()}
	if *params != "" {
		raw, e := os.ReadFile(*params)
		must(e)
		p, e := b.ParseParams(raw)
		must(e)
		combos = []any{p}
	}
	if *grid != "" {
		var g map[string]json.RawMessage
		load(*grid, &g)
		delete(g, "_comment")
		if len(g) != 9 {
			panic("grid must have all nine fields")
		}
		names := []string{}
		for n := range g {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			var vals []float64
			must(json.Unmarshal(g[n], &vals))
			var next []any
			for _, p := range combos {
				for _, x := range vals {
					v := reflect.New(reflect.TypeOf(p)).Elem()
					v.Set(reflect.ValueOf(p))
					f := v.FieldByName(n)
					if f.Kind() == reflect.Int {
						f.SetInt(int64(x))
					} else {
						f.SetFloat(x)
					}
					next = append(next, v.Interface())
				}
			}
			combos = next
		}
	}
	cfg := d.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 100, EntryAtNextOpen: true}
	must(os.MkdirAll(*out, 0755))
	rows := make([]row, len(combos))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				p := combos[i]
				obs := &observer{Strategy: b.Build(p)}
				res := d.Run(obs, bars, daily, nil, cfg)
				m := d.Compute(res, res.BarsInMarket, len(res.Equity), to.Sub(from).Hours()/24)
				r := row{Params: p, Metrics: map[string]any{}, OpenPosition: obs.Open}
				if obs.Open != nil {
					r.OpenHours = bars[len(bars)-1].Time.Add(30 * time.Minute).Sub(obs.Open.EntryTime).Hours()
				}
				profits, losses := [2]float64{}, [2]float64{}
				for _, t := range res.Trades {
					y := 0
					if !t.EntryTime.Before(split) {
						y = 1
					}
					r.AnnualTrades[y]++
					r.AnnualClosedPnL[y] += t.PnL
					if t.PnL >= 0 {
						profits[y] += t.PnL
					} else {
						losses[y] -= t.PnL
					}
				}
				for y := 0; y < 2; y++ {
					pf := 0.
					if losses[y] > 0 {
						pf = profits[y] / losses[y]
					} else if profits[y] > 0 {
						pf = math.Inf(1)
					}
					r.AnnualPF[y] = fmt.Sprint(pf)
				}
				mv := reflect.ValueOf(m)
				mt := mv.Type()
				for j := 0; j < mv.NumField(); j++ {
					v := mv.Field(j)
					if v.Kind() == reflect.Float64 && (math.IsInf(v.Float(), 0) || math.IsNaN(v.Float())) {
						r.Metrics[mt.Field(j).Name] = fmt.Sprint(v.Float())
					} else {
						r.Metrics[mt.Field(j).Name] = v.Interface()
					}
				}
				r.Eligible = m.TotalTrades >= 20 && r.AnnualTrades[0] >= 10 && r.AnnualTrades[1] >= 10 && m.NetPnL > 0 && m.ProfitFactor > 1 && m.MaxDrawdownPct <= 0.10
				rows[i] = r
				base := filepath.Join(*out, fmt.Sprintf("combo-%04d", i+1))
				must(os.WriteFile(base+"_trades.csv", []byte(d.RenderTradesCSV(res.Trades)), 0644))
				if len(combos) == 1 {
					meta := d.Meta{Ticker: "SNGS", Interval: "Minutes30", From: from, To: to, InitialCash: cfg.InitialCash, Fraction: cfg.Fraction, Commission: cfg.Commission, Params: svc.ParamRows(p), OpenPosition: obs.Open != nil}
					must(os.WriteFile(base+".md", []byte(d.RenderMarkdown(meta, m, res.Trades, res.Equity)), 0644))
					must(os.WriteFile(base+"_equity.csv", []byte(d.RenderEquityCSV(res.Equity)), 0644))
				}
			}
		}()
	}
	for i := range combos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	write(filepath.Join(*out, "results.json"), rows)
	write(filepath.Join(*out, "window.json"), map[string]any{"From": from, "To": to, "Split": split, "M30Count": len(bars), "M30First": bars[0].Time, "M30Last": bars[len(bars)-1].Time, "DailyCount": len(daily), "DailyFirst": daily[0].Time, "DailyLast": daily[len(daily)-1].Time, "Config": cfg})
	n := 0
	for _, r := range rows {
		if r.Eligible {
			n++
		}
	}
	fmt.Printf("%s: %d combinations, %d eligible\n", *out, len(rows), n)
}
