// Package statestore persists per-ticker reversion entry-state (EntryATR,
// MaxFavorablePrice, etc.) that the broker does not return. The state file is the
// primary source for the protective/profit exits; reconstruct is the fallback.
package statestore

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Entry is the persisted entry-state for one open position.
type Entry struct {
	Ticker     string    `json:"ticker"`
	EntryTime  time.Time `json:"entryTime"`
	EntryPrice float64   `json:"entryPrice"`
	EntryATR   float64   `json:"entryATR"`
	MaxFav     float64   `json:"maxFav"`
	Quantity   int64     `json:"quantity"`

	// TakeProfit — цель, замороженная на входе. Бэктест хранит её на позиции
	// (strategy.Position.TakeProfit), а живой раннер обязан пережить её через перезапуск,
	// иначе close-модель TP не с чем сравнивать high бара. reversion её не пишет —
	// omitempty оставляет формат файла прежним.
	TakeProfit float64 `json:"takeProfit,omitempty"`

	StopOrderID string  `json:"stopOrderId,omitempty"` // активная биржевая стоп-заявка ("" = нет)
	StopPrice   float64 `json:"stopPrice,omitempty"`   // уровень выставленной заявки
	StopReason  string  `json:"stopReason,omitempty"`  // компонент уровня: SL | ATRSL | TRAIL
}

// Store loads and saves the full per-ticker entry-state map.
type Store interface {
	Load() (map[string]Entry, error)
	Save(map[string]Entry) error
}

// FileStore persists the map as a single JSON file with atomic writes.
type FileStore struct {
	path string
}

// New returns a FileStore backed by path.
func New(path string) *FileStore { return &FileStore{path: path} }

// Load reads the state map; a missing file yields an empty map (not an error).
func (s *FileStore) Load() (map[string]Entry, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]Entry{}, nil
		}
		return nil, err
	}
	out := map[string]Entry{}
	if len(b) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Save writes the map atomically: marshal to a temp file in the same dir, then rename.
func (s *FileStore) Save(m map[string]Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path)
}
