package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/utils"
)

// CandleFetcher returns the full weekly-candle history for a share. The
// backtest cache calls it on misses (and on every call when --refresh).
type CandleFetcher interface {
	Fetch(ctx context.Context, shareID string) ([]*model.CandleItemTechAnalyse, error)
}

// Cache stores weekly candles per share as JSON on disk. JSON encodes
// float64 prices (already converted via utils.CombinePrice) rather than
// the Quotation{Units, Nano} pair, to keep the file readable.
type Cache struct {
	dir     string
	fetcher CandleFetcher
	refresh bool
}

func NewCache(dir string, fetcher CandleFetcher, refresh bool) *Cache {
	return &Cache{dir: dir, fetcher: fetcher, refresh: refresh}
}

type diskCandle struct {
	Date   time.Time `json:"date"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume int64     `json:"volume"`
}

func (c *Cache) path(shareID string) string {
	return filepath.Join(c.dir, shareID+"_W.json")
}

func (c *Cache) Get(ctx context.Context, shareID string) ([]*model.CandleItemTechAnalyse, error) {
	if !c.refresh {
		if candles, ok, err := c.readDisk(shareID); err != nil {
			return nil, err
		} else if ok {
			return candles, nil
		}
	}
	fetched, err := c.fetcher.Fetch(ctx, shareID)
	if err != nil {
		return nil, fmt.Errorf("fetch candles for %s: %w", shareID, err)
	}
	if err := c.writeDisk(shareID, fetched); err != nil {
		return nil, fmt.Errorf("write cache for %s: %w", shareID, err)
	}
	return fetched, nil
}

func (c *Cache) readDisk(shareID string) ([]*model.CandleItemTechAnalyse, bool, error) {
	data, err := os.ReadFile(c.path(shareID))
	if errIsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var disk []diskCandle
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, false, err
	}
	out := make([]*model.CandleItemTechAnalyse, len(disk))
	for i, d := range disk {
		oU, oN := utils.SplitPrice(d.Open)
		hU, hN := utils.SplitPrice(d.High)
		lU, lN := utils.SplitPrice(d.Low)
		cU, cN := utils.SplitPrice(d.Close)
		out[i] = &model.CandleItemTechAnalyse{
			Time:   d.Date,
			Open:   model.Quotation{Units: oU, Nano: oN},
			High:   model.Quotation{Units: hU, Nano: hN},
			Low:    model.Quotation{Units: lU, Nano: lN},
			Close:  model.Quotation{Units: cU, Nano: cN},
			Volume: d.Volume,
		}
	}
	return out, true, nil
}

func (c *Cache) writeDisk(shareID string, candles []*model.CandleItemTechAnalyse) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	disk := make([]diskCandle, len(candles))
	for i, k := range candles {
		disk[i] = diskCandle{
			Date:   k.Time,
			Open:   utils.CombinePrice(k.Open.Units, k.Open.Nano),
			High:   utils.CombinePrice(k.High.Units, k.High.Nano),
			Low:    utils.CombinePrice(k.Low.Units, k.Low.Nano),
			Close:  utils.CombinePrice(k.Close.Units, k.Close.Nano),
			Volume: k.Volume,
		}
	}
	data, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(shareID), data, 0o644)
}

func errIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }
