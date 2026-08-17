// Command tmp_measure prints instrument statistics used to build rsi_pullback calibration grids.
// Temporary: deleted right after the measurements are written into the grid comments.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"tinvest/pkg/indicators"
)

type candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

func main() {
	ticker := os.Args[1]

	m30 := load("data/candles/" + ticker + "_Minutes30.json")
	day := load("data/candles/" + ticker + "_Day1.json")

	now := time.Now()
	from := now.AddDate(0, -36, 0)
	m30 = window(m30, from)
	day = window(day, from)

	fmt.Printf("=== %s ===\n", ticker)
	fmt.Printf("30m bars: %d (%s .. %s), %.1f months\n", len(m30),
		m30[0].Time.Format("2006-01-02"), m30[len(m30)-1].Time.Format("2006-01-02"),
		m30[len(m30)-1].Time.Sub(m30[0].Time).Hours()/24/30.44)

	weekday := make([]candle, 0, len(m30))
	for _, c := range m30 {
		if wd := c.Time.Weekday(); wd != time.Saturday && wd != time.Sunday {
			weekday = append(weekday, c)
		}
	}
	fmt.Printf("weekday 30m bars: %d\n", len(weekday))

	fmt.Println("\n-- RSI cross-down counts (all / weekday) --")
	closesAll := closes(m30)
	closesWd := closes(weekday)
	levels := []float64{10, 15, 20, 25, 30, 35, 40, 45, 50}
	fmt.Printf("%-8s", "period")
	for _, l := range levels {
		fmt.Printf("%14.0f", l)
	}
	fmt.Println()
	for p := 4; p <= 7; p++ {
		rAll := indicators.RSISeries(closesAll, p)
		rWd := indicators.RSISeries(closesWd, p)
		fmt.Printf("RSI(%d)  ", p)
		for _, l := range levels {
			fmt.Printf("%8d(%4d)", crossDown(rAll, l, p), crossDown(rWd, l, p))
		}
		fmt.Println()
	}

	fmt.Println("\n-- daily ATR(14) as % of close --")
	atr := dailyATR(day, 14)
	pct := make([]float64, 0, len(atr))
	for i, a := range atr {
		if a > 0 && day[i].Close > 0 {
			pct = append(pct, a/day[i].Close*100)
		}
	}
	sort.Float64s(pct)
	fmt.Printf("median %.2f%%  p10 %.2f%%  p90 %.2f%%  n=%d\n",
		quantile(pct, 0.5), quantile(pct, 0.1), quantile(pct, 0.9), len(pct))

	fmt.Println("\n-- daily range (high-low) as % of close --")
	rng := make([]float64, 0, len(day))
	for _, d := range day {
		if d.Close > 0 {
			rng = append(rng, (d.High-d.Low)/d.Close*100)
		}
	}
	sort.Float64s(rng)
	fmt.Printf("median %.2f%%  p25 %.2f%%  p75 %.2f%%\n",
		quantile(rng, 0.5), quantile(rng, 0.25), quantile(rng, 0.75))

	fmt.Println("\n-- turnover (mln RUB/day) --")
	turn := make([]float64, 0, len(day))
	for _, d := range day {
		turn = append(turn, d.Volume*d.Close/1e6)
	}
	sort.Float64s(turn)
	fmt.Printf("median %.0f  p10 %.0f  p90 %.0f (volume in lots, not shares)\n",
		quantile(turn, 0.5), quantile(turn, 0.1), quantile(turn, 0.9))

	fmt.Println("\n-- regime --")
	split := now.AddDate(0, -6, 0)
	trainFirst, trainLast, holdFirst, holdLast := 0.0, 0.0, 0.0, 0.0
	var peak, trough float64
	var peakT, troughT time.Time
	for _, c := range m30 {
		if peak == 0 || c.High > peak {
			peak, peakT = c.High, c.Time
		}
		if trough == 0 || c.Low < trough {
			trough, troughT = c.Low, c.Time
		}
		if c.Time.Before(split) {
			if trainFirst == 0 {
				trainFirst = c.Close
			}
			trainLast = c.Close
		} else {
			if holdFirst == 0 {
				holdFirst = c.Close
			}
			holdLast = c.Close
		}
	}
	fmt.Printf("train  %.2f -> %.2f  (%+.1f%%)\n", trainFirst, trainLast, (trainLast/trainFirst-1)*100)
	fmt.Printf("holdout %.2f -> %.2f  (%+.1f%%)\n", holdFirst, holdLast, (holdLast/holdFirst-1)*100)
	fmt.Printf("full   %.2f -> %.2f  (%+.1f%%)\n", trainFirst, holdLast, (holdLast/trainFirst-1)*100)
	fmt.Printf("peak %.2f (%s), trough %.2f (%s), max drawdown %.1f%%\n",
		peak, peakT.Format("2006-01-02"), trough, troughT.Format("2006-01-02"), (trough/peak-1)*100)

	fmt.Println("\n-- half-year returns --")
	for k := 6; k >= 1; k-- {
		a, b := now.AddDate(0, -6*k, 0), now.AddDate(0, -6*(k-1), 0)
		var first, last float64
		for _, c := range m30 {
			if c.Time.Before(a) || !c.Time.Before(b) {
				continue
			}
			if first == 0 {
				first = c.Close
			}
			last = c.Close
		}
		if first > 0 {
			fmt.Printf("%s..%s  %+.1f%%\n", a.Format("2006-01"), b.Format("2006-01"), (last/first-1)*100)
		}
	}

	fmt.Println("\n-- day state --")
	dayStats(m30, day)

	fmt.Println("\n-- RSI cross-up counts (weekday), exit band --")
	upLevels := []float64{55, 60, 65, 70, 75, 80, 85}
	fmt.Printf("%-8s", "period")
	for _, l := range upLevels {
		fmt.Printf("%8.0f", l)
	}
	fmt.Println()
	for p := 4; p <= 7; p++ {
		rWd := indicators.RSISeries(closesWd, p)
		fmt.Printf("RSI(%d)  ", p)
		for _, l := range upLevels {
			fmt.Printf("%8d", crossUp(rWd, l, p))
		}
		fmt.Println()
	}

	fmt.Println("\n-- volume slot gate: share of weekday bars clearing VolMult --")
	volStats(weekday)

	fmt.Println("\n-- EMA distance: share of weekday bars with EMAfast>EMAslow --")
	for _, pair := range [][2]int{
		{5, 50}, {5, 100}, {5, 150}, {5, 200},
		{10, 50}, {10, 70}, {10, 100}, {10, 120}, {10, 150}, {10, 170}, {10, 200},
		{20, 50}, {20, 100}, {20, 150}, {20, 200},
		{30, 100}, {30, 150}, {30, 200}, {40, 150}, {40, 200},
	} {
		f := ema(closesWd, pair[0])
		s := ema(closesWd, pair[1])
		up := 0
		n := 0
		for i := pair[1]; i < len(closesWd); i++ {
			n++
			if f[i] > s[i] {
				up++
			}
		}
		fmt.Printf("EMA %d/%d: %.1f%% of %d bars\n", pair[0], pair[1], float64(up)/float64(n)*100, n)
	}
}

// dayStats prints how the day-state gate would see this instrument: the daily range in units of
// the previous day's ATR, and the share of weekday 30m bars falling into each branch of the gate.
func dayStats(m30, day []candle) {
	atr := dailyATR(day, 14)
	prevATR := map[string]float64{}
	for i := 1; i < len(day); i++ {
		if atr[i-1] > 0 {
			prevATR[day[i].Time.Format("2006-01-02")] = atr[i-1]
		}
	}

	ratios := make([]float64, 0, len(day))
	for _, d := range day {
		if a := prevATR[d.Time.Format("2006-01-02")]; a > 0 {
			ratios = append(ratios, (d.High-d.Low)/a)
		}
	}
	sort.Float64s(ratios)
	fmt.Printf("daily range / prev-day ATR: median %.2f  p25 %.2f  p75 %.2f  n=%d\n",
		quantile(ratios, 0.5), quantile(ratios, 0.25), quantile(ratios, 0.75), len(ratios))
	for _, th := range []float64{0.6, 0.8, 0.9, 1.0, 1.25, 1.3, 1.5} {
		n := 0
		for _, r := range ratios {
			if r >= th {
				n++
			}
		}
		fmt.Printf("  days reaching %.2f ATR: %.1f%%\n", th, float64(n)/float64(len(ratios))*100)
	}

	// Per-bar view: used = TodayHigh-TodayLow up to and including the bar, as dayStateOK sees it.
	var curDay string
	var hi, lo float64
	var secondBarRatio []float64
	fresh := map[float64]int{0.3: 0, 0.4: 0, 0.5: 0}
	spent := map[float64]int{0.6: 0, 0.8: 0, 0.9: 0, 1.0: 0, 1.25: 0, 1.3: 0, 1.5: 0}
	total := 0
	barIdx := 0
	for _, c := range m30 {
		if wd := c.Time.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		d := c.Time.Format("2006-01-02")
		if d != curDay {
			curDay, hi, lo, barIdx = d, c.High, c.Low, 0
		}
		if c.High > hi {
			hi = c.High
		}
		if c.Low < lo {
			lo = c.Low
		}
		barIdx++
		a := prevATR[d]
		if a <= 0 {
			continue
		}
		used := (hi - lo) / a
		if barIdx == 2 {
			secondBarRatio = append(secondBarRatio, used)
		}
		total++
		for th := range fresh {
			if used <= th {
				fresh[th]++
			}
		}
		for th := range spent {
			if used >= th {
				spent[th]++
			}
		}
	}
	sort.Float64s(secondBarRatio)
	fmt.Printf("used at 2nd bar of day: median %.2f ATR\n", quantile(secondBarRatio, 0.5))
	for _, th := range []float64{0.3, 0.4, 0.5} {
		fmt.Printf("  fresh branch (used <= %.1f ATR): %.1f%% of %d weekday bars\n", th, float64(fresh[th])/float64(total)*100, total)
	}
	for _, th := range []float64{0.6, 0.8, 0.9, 1.0, 1.25, 1.3, 1.5} {
		fmt.Printf("  spent branch (used >= %.2f ATR): %.1f%%\n", th, float64(spent[th])/float64(total)*100)
	}
}

// volStats reproduces the core's slot baseline: for every weekday bar, the average volume of its
// own 30-minute slot over the last VolBaseDays COMPLETED weekday days (the current day excluded,
// slots with fewer than two observations falling back to a flat average). Printed is the share of
// bars whose own volume clears VolMult times that baseline — the single-bar view, not the
// three-bar lookback the gate actually uses, so the numbers are a lower bound on the pass rate.
func volStats(weekday []candle) {
	byDay := map[string][]candle{}
	dayKeys := make([]string, 0)
	for _, c := range weekday {
		k := c.Time.Format("2006-01-02")
		if _, ok := byDay[k]; !ok {
			dayKeys = append(dayKeys, k)
		}
		byDay[k] = append(byDay[k], c)
	}

	for _, baseDays := range []int{3, 5, 10, 14} {
		ratios := make([]float64, 0, len(weekday))
		for di := baseDays; di < len(dayKeys); di++ {
			sums := map[int]float64{}
			counts := map[int]int{}
			var flatSum float64
			var flatCount int
			for k := di - baseDays; k < di; k++ {
				for _, c := range byDay[dayKeys[k]] {
					if c.Volume <= 0 {
						continue
					}
					sl := c.Time.Hour()*60 + c.Time.Minute()
					sums[sl] += c.Volume
					counts[sl]++
					flatSum += c.Volume
					flatCount++
				}
			}
			if flatCount == 0 {
				continue
			}
			flat := flatSum / float64(flatCount)
			for _, c := range byDay[dayKeys[di]] {
				if c.Volume <= 0 {
					continue
				}
				sl := c.Time.Hour()*60 + c.Time.Minute()
				base := flat
				if counts[sl] >= 2 {
					base = sums[sl] / float64(counts[sl])
				}
				if base > 0 {
					ratios = append(ratios, c.Volume/base)
				}
			}
		}
		sorted := append([]float64(nil), ratios...)
		sort.Float64s(sorted)
		fmt.Printf("base %2d days (n=%d): median %.2f  p75 %.2f  p90 %.2f", baseDays, len(sorted),
			quantile(sorted, 0.5), quantile(sorted, 0.75), quantile(sorted, 0.9))
		for _, th := range []float64{1.0, 1.2, 1.5, 2.0, 2.5, 3.0} {
			n := 0
			for _, r := range sorted {
				if r >= th {
					n++
				}
			}
			fmt.Printf("  |%.1f: %.1f%%", th, float64(n)/float64(len(sorted))*100)
		}
		fmt.Println()
	}
}

func crossUp(rsi []float64, level float64, period int) int {
	n := 0
	for i := period + 1; i < len(rsi); i++ {
		if rsi[i-1] <= level && rsi[i] > level {
			n++
		}
	}
	return n
}

func closes(cs []candle) []float64 {
	out := make([]float64, len(cs))
	for i, c := range cs {
		out[i] = c.Close
	}
	return out
}

func crossDown(rsi []float64, level float64, period int) int {
	n := 0
	for i := period + 1; i < len(rsi); i++ {
		if rsi[i-1] >= level && rsi[i] < level {
			n++
		}
	}
	return n
}

func dailyATR(day []candle, period int) []float64 {
	out := make([]float64, len(day))
	if len(day) <= period {
		return out
	}
	tr := make([]float64, len(day))
	for i := 1; i < len(day); i++ {
		h, l, pc := day[i].High, day[i].Low, day[i-1].Close
		m := h - l
		if v := abs(h - pc); v > m {
			m = v
		}
		if v := abs(l - pc); v > m {
			m = v
		}
		tr[i] = m
	}
	var sum float64
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	out[period] = sum / float64(period)
	for i := period + 1; i < len(day); i++ {
		out[i] = (out[i-1]*float64(period-1) + tr[i]) / float64(period)
	}
	return out
}

func ema(v []float64, period int) []float64 {
	out := make([]float64, len(v))
	if len(v) < period {
		return out
	}
	k := 2.0 / float64(period+1)
	var sum float64
	for i := 0; i < period; i++ {
		sum += v[i]
	}
	out[period-1] = sum / float64(period)
	for i := period; i < len(v); i++ {
		out[i] = v[i]*k + out[i-1]*(1-k)
	}
	return out
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(q * float64(len(sorted)-1))
	return sorted[i]
}

func window(cs []candle, from time.Time) []candle {
	out := make([]candle, 0, len(cs))
	for _, c := range cs {
		if !c.Time.Before(from) {
			out = append(out, c)
		}
	}
	return out
}

func load(path string) []candle {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var out []candle
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
