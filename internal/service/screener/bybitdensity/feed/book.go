package feed

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

type bookState struct {
	bids map[int64]float64
	asks map[int64]float64
	id   int64
	seq  int64
	at   millis
}

func (b *bookState) apply(symbol string, tick decimal, depth int, message streamMessage, received time.Time) (*model.Book, error) {
	var data struct {
		Symbol string     `json:"s"`
		ID     int64      `json:"u"`
		Seq    int64      `json:"seq"`
		Bids   [][]string `json:"b"`
		Asks   [][]string `json:"a"`
	}
	if err := json.Unmarshal(message.Data, &data); err != nil {
		return nil, fmt.Errorf("decode orderbook: %w", err)
	}
	if data.Symbol != symbol || data.ID <= 0 || data.Bids == nil || data.Asks == nil {
		return nil, errors.New("invalid orderbook identity or sides")
	}
	at := cmp.Or(message.CTS, message.TS)
	if at <= 0 {
		return nil, errors.New("missing orderbook timestamp")
	}

	switch message.Type {
	case "snapshot":
		// Level-one books repeat the same update ID as a response to inactivity.
		if depth == 1 && data.ID == b.id && data.Seq == b.seq {
			return nil, nil
		}
		b.bids = make(map[int64]float64, len(data.Bids))
		b.asks = make(map[int64]float64, len(data.Asks))
	case "delta":
		if b.bids == nil {
			return nil, nil
		}
		if data.ID == b.id && data.Seq == b.seq {
			return nil, nil
		}
		if data.ID <= b.id || (data.Seq > 0 && data.Seq < b.seq) || at < b.at {
			return nil, errors.New("out-of-order orderbook delta")
		}
	default:
		return nil, fmt.Errorf("unknown orderbook update type %q", message.Type)
	}

	bidsChanged, err := applyLevels(b.bids, data.Bids, tick)
	if err != nil {
		return nil, fmt.Errorf("bids: %w", err)
	}
	asksChanged, err := applyLevels(b.asks, data.Asks, tick)
	if err != nil {
		return nil, fmt.Errorf("asks: %w", err)
	}
	b.id, b.seq, b.at = data.ID, data.Seq, at
	snapshot := message.Type == "snapshot"
	if !snapshot && !bidsChanged && !asksChanged {
		return nil, nil
	}

	if len(b.bids) > depth || len(b.asks) > depth {
		return nil, fmt.Errorf("orderbook exceeded subscribed depth %d", depth)
	}
	bids := sortedLevels(b.bids, true)
	asks := sortedLevels(b.asks, false)
	if len(bids) == 0 || len(asks) == 0 {
		return nil, errors.New("one-sided orderbook")
	}
	if bids[0].Ticks >= asks[0].Ticks {
		return nil, errors.New("crossed orderbook")
	}

	return &model.Book{Bids: bids, Asks: asks, At: time.UnixMilli(int64(at)), Received: received, Snapshot: snapshot}, nil
}

func applyLevels(levels map[int64]float64, changes [][]string, tick decimal) (bool, error) {
	changed := false
	for _, entry := range changes {
		if len(entry) != 2 {
			return false, fmt.Errorf("expected level price and quantity, got %d values", len(entry))
		}
		ticks, err := tick.ticks(entry[0])
		if err != nil {
			return false, err
		}
		qty, err := nonnegative(entry[1])
		if err != nil {
			return false, err
		}
		if levels[ticks] == qty {
			continue
		}
		changed = true
		if qty == 0 {
			delete(levels, ticks)
		} else {
			levels[ticks] = qty
		}
	}

	return changed, nil
}

func sortedLevels(levels map[int64]float64, descending bool) []model.Level {
	result := make([]model.Level, 0, len(levels))
	for ticks, qty := range levels {
		result = append(result, model.Level{Ticks: ticks, Qty: qty})
	}
	slices.SortFunc(result, func(a, b model.Level) int {
		if descending {
			return cmp.Compare(b.Ticks, a.Ticks)
		}

		return cmp.Compare(a.Ticks, b.Ticks)
	})

	return result
}
