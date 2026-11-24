package bybitdensity

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"
)

const backtestRoot = "reports/bybit-density-backtest"

type backtestVolume struct {
	from, until, referenceDay time.Time
	average, minimum          float64
}

// RunBacktest runs a public-data-only spot replay for one configured instrument.
func RunBacktest(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("bybit-density-backtest", flag.ContinueOnError)
	flags.SetOutput(out)
	ticker := flags.String("ticker", "", "configured spot symbol, e.g. SOL or SOLUSDT")
	fromFlag := flags.String("from", "", "first UTC date (YYYY-MM-DD), defaults to last Monday-Sunday")
	toFlag := flags.String("to", "", "last UTC date, inclusive (YYYY-MM-DD)")
	notional := flags.Float64("notional", 250, "maximum position size in USDT")
	fee := flags.Float64("fee", .001, "fee fraction per side")
	cache := flags.String("cache", filepath.Join(backtestRoot, "cache"), "archive cache directory")
	output := flags.String("out", "", "report directory, defaults to reports/bybit-density-backtest/<symbol>/<date-range>")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments; use -ticker")
	}
	symbol := strings.ToUpper(strings.TrimSpace(*ticker))
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}
	profile, ok := defaultSpotProfiles()[symbol]
	if !ok {
		return fmt.Errorf("use -ticker with a configured spot symbol (see spot.go)")
	}
	if !(*notional > 0) || math.IsInf(*notional, 0) || !(*fee >= 0 && *fee < 1) {
		return fmt.Errorf("notional must be finite and positive, and fee must be in [0, 1)")
	}
	from, to, err := backtestDates(*fromFlag, *toFlag, time.Now())
	if err != nil {
		return err
	}
	market := feed.NewClient("", "")
	instruments, err := market.Instruments(ctx, model.Spot, []string{symbol}, 1)
	if err != nil {
		return fmt.Errorf("instrument %s: %w", symbol, err)
	}
	if len(instruments) != 1 {
		return fmt.Errorf("no spot instrument metadata for %s", symbol)
	}
	instrument := instruments[0]
	end := to.AddDate(0, 0, 1)
	windows, err := loadInPlayWindows(ctx, market, instrument.Key, profile.inPlay, from, end)
	if err != nil {
		return err
	}
	volumes, err := loadBacktestVolumes(ctx, market, instrument.Key, profile, from, end)
	if err != nil {
		return err
	}
	var policy strategy.ZonePolicy
	if profile.zoneFilter.Enabled() {
		policy = strategy.NewNeighborZonePolicy(instrument, profile.zoneWidthUSDT, profile.zoneFilter)
	} else {
		policy = strategy.NewZonePolicy(instrument, profile.zoneWidthUSDT, volumes[0].minimum)
	}
	sim := strategy.NewZoneReplay(instrument, policy, *notional, *fee, time.Duration(profile.bounceConfirmSeconds)*time.Second)
	volumeIndex := 0
	inPlayIndex := -1
	if len(windows) > 0 {
		sim.SetInPlayWindow(strategy.InPlayWindow{})
	}
	apply := func(event model.Event) {
		var at time.Time
		if event.Book != nil {
			at = event.Book.At
		} else if len(event.Trades) > 0 {
			at = event.Trades[0].At
		}
		if len(windows) > 0 && !at.Before(from) && at.Before(end) {
			index := int(at.Sub(from) / strategy.InPlayInterval)
			if index != inPlayIndex {
				sim.SetInPlayWindow(windows[index])
				inPlayIndex = index
			}
		}
		for !profile.zoneFilter.Enabled() && volumeIndex+1 < len(volumes) && !at.Before(volumes[volumeIndex].until) {
			volumeIndex++
			sim.SetMinZoneQuantity(volumes[volumeIndex].minimum)
		}
		sim.Apply(event)
	}
	var source feed.SpotArchiveReplay
	client := &http.Client{Timeout: 5 * time.Minute}
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		date := day.Format(time.DateOnly)
		if _, err := fmt.Fprintf(out, "Replay %s %s UTC\n", symbol, date); err != nil {
			return err
		}
		books, trades, err := spotArchives(ctx, client, filepath.Join(*cache, symbol), symbol, date)
		if err != nil {
			return fmt.Errorf("%s %s: %w", symbol, date, err)
		}
		if err := source.Day(ctx, instrument, day, books, trades, apply); err != nil {
			return fmt.Errorf("%s %s: %w", symbol, date, err)
		}
	}
	dir := *output
	if dir == "" {
		dir = filepath.Join(backtestRoot, symbol, from.Format(time.DateOnly)+"--"+to.Format(time.DateOnly))
	}
	result := sim.Result()
	result.InPlayWindows = len(windows)
	for _, window := range windows {
		if window.Active {
			result.InPlayActiveWindows++
		}
	}
	return writeBacktestReport(out, dir, symbol, from, end, profile, *notional, *fee, result, volumes)
}

func loadBacktestVolumes(ctx context.Context, market marketData, key model.Key, profile spotProfile, from, end time.Time) ([]backtestVolume, error) {
	var volumes []backtestVolume
	allowMissing := profile.inPlay.Enabled() && profile.zoneFilter.Enabled()
	for at := from; at.Before(end); {
		referenceDay, referenceEnd, until := previousSpotVolumeWindow(at)
		average, err := market.AverageVolume(ctx, key, referenceDay, referenceEnd)
		if err != nil {
			if !allowMissing || ctx.Err() != nil {
				return nil, fmt.Errorf("M5 volume for %s on %s MSK: %w", key.Symbol, referenceDay.Format(time.DateOnly), err)
			}
			average = 0
		}
		if !allowMissing && !(average > 0) {
			return nil, fmt.Errorf("no positive M5 volume for %s on %s MSK", key.Symbol, referenceDay.Format(time.DateOnly))
		}
		if end.Before(until) {
			until = end
		}
		volumes = append(volumes, backtestVolume{from: at, until: until, referenceDay: referenceDay, average: average, minimum: average * profile.m5VolumeMultiplier})
		at = until
	}
	return volumes, nil
}

func backtestDates(first, last string, now time.Time) (from, to time.Time, err error) {
	now = now.UTC().Truncate(24 * time.Hour)
	if first == "" && last == "" {
		monday := now.AddDate(0, 0, -int((now.Weekday()+6)%7))
		return monday.AddDate(0, 0, -7), monday.AddDate(0, 0, -1), nil
	}
	from, err = time.Parse(time.DateOnly, first)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("-from must be a UTC date (YYYY-MM-DD)")
	}
	to, err = time.Parse(time.DateOnly, last)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("-to must be a UTC date (YYYY-MM-DD)")
	}
	if to.Before(from) || !to.Before(now) {
		return time.Time{}, time.Time{}, fmt.Errorf("require -from <= -to and complete UTC days")
	}
	return from, to, nil
}
