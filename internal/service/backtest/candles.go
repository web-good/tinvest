// Package backtest wires the pure backtest engine to real candle data: a
// file-cached, chunked Tinkoff fetcher plus a strategy registry and grid
// calibration. All gRPC/file I/O lives here; the engine itself stays pure.
package backtest

import (
	"context"
	"encoding/json"
	"errors"
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
		// refresh — это «забыть всё, что мы знали», включая вывод о начале истории:
		// брокер мог добить архив назад, и без сброса мы бы никогда об этом не узнали.
		_ = os.Remove(metaPath(path))
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
	// 12-month cache would quietly calibrate on half the history. A head that comes back EMPTY
	// is NOT fatal — an instrument may simply not have traded that far back, so we warn,
	// remember it and continue with what exists. A head that FAILED is fatal, exactly like a
	// failing tail: treating an API outage as "no history" would silently shorten the run
	// window, and the report would say nothing about it.
	dirty := false
	if first := cached[0].Time; first.After(from) && !p.historyStartsAt(path, first) {
		head, ferr := p.fetchRange(ctx, instrumentID, interval, from, first)
		switch {
		case errors.Is(ferr, errNoHistory):
			logger.Warn(fmt.Sprintf("backtest: %s (%s) has no candles before %s — the window starts where its history does, not at %s",
				ticker, interval.String(), first, from))
			// Запоминаем, иначе каждый следующий прогон снова долбит API чанками с
			// паузой за периодом, которого не существует.
			p.rememberHistoryStart(path, first)
		case ferr != nil:
			return nil, fmt.Errorf("backtest: %s (%s) head fetch [%s, %s]: %w",
				ticker, interval.String(), from, first, ferr)
		default:
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
	var failed error
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
			if failed == nil {
				failed = err // первая настоящая причина; остальные уже в логе
			}
		} else {
			all = mergeCandles(all, convertCandles(items))
		}
		winFrom = winTo
		time.Sleep(fetchPause)
	}
	if len(all) == 0 {
		// Пустой ответ без единого сбоя и пустой ответ из-за упавшего API — разные вещи:
		// первое означает «инструмент столько не торговался» и допускает деградацию,
		// второе обязано ронять загрузку. Различает их errNoHistory у вызывающего.
		if failed != nil {
			return nil, fmt.Errorf("backtest: no candles fetched for %s in [%s, %s]: %w", instrumentID, from, to, failed)
		}
		return nil, fmt.Errorf("backtest: %s has no candles in [%s, %s]: %w", instrumentID, from, to, errNoHistory)
	}
	return all, nil
}

// errNoHistory marks an empty fetch that hit no API errors at all — the exchange simply has
// nothing for that instrument in that window.
var errNoHistory = errors.New("no candle history in range")

// historyMeta is the sidecar next to a candle cache file: what a previous run learned about
// where the instrument's history actually begins.
type historyMeta struct {
	// EarliestKnown is the oldest bar the exchange returned anything for. A head top-up
	// asking for a window that starts before it is known to be pointless.
	EarliestKnown time.Time `json:"earliestKnown"`
}

func metaPath(cachePath string) string { return cachePath + ".meta" }

// historyStartsAt reports whether a previous run already established that nothing exists
// before first, so the head top-up can be skipped entirely.
func (p *CandleProvider) historyStartsAt(cachePath string, first time.Time) bool {
	raw, err := os.ReadFile(metaPath(cachePath))
	if err != nil {
		return false
	}
	var m historyMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	return !m.EarliestKnown.IsZero() && !first.After(m.EarliestKnown)
}

// rememberHistoryStart records that the instrument has nothing before first. Failures are
// swallowed: the sidecar is an optimisation, and losing it only costs one extra fetch.
func (p *CandleProvider) rememberHistoryStart(cachePath string, first time.Time) {
	raw, err := json.Marshal(historyMeta{EarliestKnown: first})
	if err != nil {
		return
	}
	_ = os.WriteFile(metaPath(cachePath), raw, 0o600)
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
