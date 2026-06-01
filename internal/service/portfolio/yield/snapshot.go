package yield

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Snapshot captures the total portfolio value at a point in time.
type Snapshot struct {
	Date       time.Time `json:"date"`        // when the snapshot was taken
	TotalValue float64   `json:"total_value"` // total portfolio value (RUB)
}

type snapshotStore struct {
	path string
}

func newSnapshotStore(path string) *snapshotStore {
	return &snapshotStore{path: path}
}

// load reads all snapshots from s.path. If the file does not exist it returns
// an empty slice and nil error. Malformed JSON is returned as an error.
func (s *snapshotStore) load() ([]Snapshot, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Snapshot{}, nil
	}
	if err != nil {
		return nil, err
	}

	var snaps []Snapshot
	if err := json.Unmarshal(data, &snaps); err != nil {
		return nil, err
	}
	return snaps, nil
}

// append loads existing snapshots, appends snap, and writes the full array back
// atomically (write to a temp file, then rename over s.path). Parent directories
// are created if they do not exist.
func (s *snapshotStore) append(snap Snapshot) error {
	snaps, err := s.load()
	if err != nil {
		return err
	}
	snaps = append(snaps, snap)

	data, err := json.MarshalIndent(snaps, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

// valueAtYearStart determines the portfolio value at the start of the given
// calendar year using the following precedence:
//  1. Last snapshot whose Date is on or before Jan 1 of year (closing value of prior year).
//  2. Earliest snapshot within the first 7 days of January of year.
//  3. manualOverride if > 0.
//  4. (0, false, nil) — no data available.
func (s *snapshotStore) valueAtYearStart(year int, manualOverride float64) (value float64, ok bool, err error) {
	snaps, err := s.load()
	if err != nil {
		return 0, false, err
	}

	target := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	tolerance := target.AddDate(0, 0, 7) // 7-day window into January
	// Only consider snapshots from within the three months before target as
	// "prior year closing value". Snapshots older than that are not a reliable
	// starting point for a year-to-date calculation.
	lookbackStart := target.AddDate(0, -3, 0)

	// Sort ascending by date so we can scan cleanly.
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Date.Before(snaps[j].Date)
	})

	// Pass 1: find the LAST snapshot on or before target that is within the
	// 3-month lookback window.
	lastOnOrBefore := -1
	for i, snap := range snaps {
		if !snap.Date.Before(lookbackStart) && !snap.Date.After(target) {
			lastOnOrBefore = i
		}
	}
	if lastOnOrBefore >= 0 {
		return snaps[lastOnOrBefore].TotalValue, true, nil
	}

	// Pass 2: find the EARLIEST snapshot strictly after target but within tolerance.
	for _, snap := range snaps {
		if snap.Date.After(target) && !snap.Date.After(tolerance) {
			return snap.TotalValue, true, nil
		}
	}

	// Pass 3: manual override.
	if manualOverride > 0 {
		return manualOverride, true, nil
	}

	return 0, false, nil
}
