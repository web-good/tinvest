// Package backtest wires the pure backtest engine to real candle data: a
// file-cached, chunked Tinkoff fetcher plus a strategy registry and grid
// calibration. All gRPC/file I/O lives here; the engine itself stays pure.
package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// fetchPause throttles requests to respect API limits.
const fetchPause = 300 * time.Millisecond

// chunkDaysFor bounds each GetCandles request window per interval, since the API
// caps the span per request by interval (see the CandleInterval doc comments in
// internal/pb/v1/marketdata.pb.go): 5-min candles span up to 1 week, 15/30-min up
// to ~3 weeks, hour-and-up months. We stay at/under the cap to fetch long ranges
// in pieces. Values are in days.
func chunkDaysFor(i enum.Interval) int {
	switch i {
	case enum.Minutes5:
		return 7 // API cap: 5-min candles span up to 1 week per request
	case enum.Minutes15, enum.Minutes30:
		return 14 // API cap is ~3 weeks for 15/30-min candles
	default:
		return 30 // hour-and-up allow months; 30 is comfortable
	}
}

// candleFetcher is the slice of the gRPC market-data client the provider needs.
// The real grpc.MarketDataServiceClient satisfies it.
type candleFetcher interface {
	GetCandles(ctx context.Context, instrumentUID *string, interval int32,
		from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool,
	) ([]*model.CandleItemTechAnalyse, error)
}

// CandleProvider loads candles for (ticker, interval), caching them in one JSON
// file per pair and topping up the tail on demand.
type CandleProvider struct {
	client candleFetcher
	dir    string // cache directory, e.g. data/candles
}

func NewCandleProvider(client candleFetcher, dir string) *CandleProvider {
	return &CandleProvider{client: client, dir: dir}
}

// Load returns oldest-first candles in [from, to]. With a warm cache it reads
// the file and only fetches the missing tail; refresh forces a full refetch.
func (p *CandleProvider) Load(ctx context.Context, ticker, instrumentID string,
	interval enum.Interval, from, to time.Time, refresh bool,
) ([]backtest.Candle, error) {
	path := p.cachePath(ticker, interval)

	if refresh {
		fetched, err := p.fetchRange(ctx, instrumentID, interval, from, to)
		if err != nil {
			return nil, err
		}
		if err := p.writeCache(path, fetched); err != nil {
			return nil, err
		}
		return sliceWindow(fetched, from, to), nil
	}

	cached, err := p.readCache(path)
	if err != nil {
		return nil, err
	}
	if len(cached) == 0 {
		fetched, ferr := p.fetchRange(ctx, instrumentID, interval, from, to)
		if ferr != nil {
			return nil, ferr
		}
		if werr := p.writeCache(path, fetched); werr != nil {
			return nil, werr
		}
		return sliceWindow(fetched, from, to), nil
	}

	// The head and the tail are topped up symmetrically. Topping up only the tail let a warm
	// but SHORT cache return a truncated window with no error: a run asking for 24 months on a
	// 12-month cache would quietly calibrate on half the history. A missing head is NOT fatal,
	// though — an instrument may simply not have traded that far back, so we warn and continue
	// with what exists instead of failing the run.
	dirty := false
	if first := cached[0].Time; first.After(from) {
		head, ferr := p.fetchRange(ctx, instrumentID, interval, from, first)
		if ferr != nil {
			logger.Warn(fmt.Sprintf("backtest: %s (%s) has no candles before %s — the window starts where its history does, not at %s",
				ticker, interval.String(), first, from))
		} else {
			cached = mergeCandles(cached, head)
			dirty = true
		}
	}
	if last := cached[len(cached)-1].Time; last.Before(to) {
		tail, ferr := p.fetchRange(ctx, instrumentID, interval, last, to)
		if ferr != nil {
			return nil, ferr
		}
		cached = mergeCandles(cached, tail)
		dirty = true
	}
	if dirty {
		if werr := p.writeCache(path, cached); werr != nil {
			return nil, werr
		}
	}
	return sliceWindow(cached, from, to), nil
}

// fetchRange pulls [from, to] in per-interval windows, converting and merging.
func (p *CandleProvider) fetchRange(ctx context.Context, instrumentID string,
	interval enum.Interval, from, to time.Time,
) ([]backtest.Candle, error) {
	var all []backtest.Candle
	id := instrumentID
	num := interval.ToNumberInvestAPI()
	chunk := time.Duration(chunkDaysFor(interval)) * 24 * time.Hour
	for winFrom := from; winFrom.Before(to); {
		winTo := winFrom.Add(chunk)
		if winTo.After(to) {
			winTo = to
		}
		items, err := p.client.GetCandles(ctx, &id, num,
			timestamppb.New(winFrom), timestamppb.New(winTo), nil, true)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("backtest: candle chunk %s-%s failed: %v", winFrom, winTo, err))
		} else {
			all = mergeCandles(all, convertCandles(items))
		}
		winFrom = winTo
		time.Sleep(fetchPause)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("backtest: no candles fetched for %s in [%s, %s]", instrumentID, from, to)
	}
	return all, nil
}

func (p *CandleProvider) cachePath(ticker string, interval enum.Interval) string {
	return filepath.Join(p.dir, fmt.Sprintf("%s_%s.json", ticker, interval.String()))
}

func (p *CandleProvider) readCache(path string) ([]backtest.Candle, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backtest: read cache %s: %w", path, err)
	}
	var out []backtest.Candle
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("backtest: parse cache %s: %w", path, err)
	}
	return out, nil
}

func (p *CandleProvider) writeCache(path string, candles []backtest.Candle) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("backtest: mkdir cache dir: %w", err)
	}
	data, err := json.MarshalIndent(candles, "", "  ")
	if err != nil {
		return fmt.Errorf("backtest: marshal cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("backtest: write cache %s: %w", path, err)
	}
	return nil
}

// convertCandles maps complete API candles to domain candles (drops the
// still-forming last bar).
func convertCandles(items []*model.CandleItemTechAnalyse) []backtest.Candle {
	out := make([]backtest.Candle, 0, len(items))
	for _, c := range items {
		if !c.IsComplete {
			continue
		}
		out = append(out, backtest.Candle{
			Time:   c.Time,
			Open:   utils.CombinePrice(c.Open.Units, c.Open.Nano),
			High:   utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:    utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close:  utils.CombinePrice(c.Close.Units, c.Close.Nano),
			Volume: c.Volume,
		})
	}
	return out
}

// mergeCandles concatenates two series, dedups by Time (first occurrence wins)
// and returns them sorted oldest-first.
func mergeCandles(a, b []backtest.Candle) []backtest.Candle {
	seen := make(map[int64]struct{}, len(a)+len(b))
	var out []backtest.Candle
	for _, src := range [][]backtest.Candle{a, b} {
		for _, c := range src {
			key := c.Time.UnixNano()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// sliceWindow returns the candles whose Time is within [from, to] inclusive.
func sliceWindow(candles []backtest.Candle, from, to time.Time) []backtest.Candle {
	var out []backtest.Candle
	for _, c := range candles {
		if c.Time.Before(from) || c.Time.After(to) {
			continue
		}
		out = append(out, c)
	}
	return out
}
