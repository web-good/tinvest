package live

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

// CheckSharedBinding runs synchronously before candle decisions. It never opens
// a writer or contacts the broker. The last complete append frame is sufficient:
// SharedRoot is fixed before the first real order, even during a later append.
func CheckSharedBinding(journalDir, account, coordinatorRoot string) error {
	dirs := map[string]bool{journalDir: true}
	if cache, err := os.UserCacheDir(); err == nil {
		data, readErr := os.ReadFile(filepath.Join(cache, "tinvest", "density-live", AccountHash(account)+".lock"))
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			return readErr
		}
		if len(data) > 0 {
			var binding struct{ Dir string }
			if json.Unmarshal(data, &binding) != nil || binding.Dir == "" {
				return fmt.Errorf("invalid known density journal binding")
			}
			dirs[binding.Dir] = true
		}
	}
	for dir := range dirs {
		state, exists, err := lastJournalState(filepath.Join(dir, "events.jsonl"))
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if (state.Version != 1 && state.Version != 2) || state.AccountHash != AccountHash(account) {
			return fmt.Errorf("density journal account binding mismatch")
		}
		if state.SharedRoot != "" && state.SharedRoot != coordinatorRoot {
			return fmt.Errorf("existing density ownership belongs to another shared state root; candle decisions blocked")
		}
		if state.SharedRoot == "" && (state.Episode != nil || state.ReleasePending != nil || len(state.Episodes) > 0 || len(state.Releases) > 0) {
			return fmt.Errorf("unresolved standalone density episode must be reconciled before shared candle decisions")
		}
	}
	return nil
}

// ReadPositions returns a durable snapshot, or nil if there is no real journal.
func ReadPositions(journalDir string) (map[string]accountcoord.DensityPosition, error) {
	state, exists, err := lastJournalState(filepath.Join(journalDir, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	state, err = migrateState(state)
	if err != nil {
		return nil, err
	}
	rememberOwnership(&state)
	return state.Ownership, nil
}

// CheckSimulationState prevents a mode switch from concealing unfinished real trades.
// It only reads real journals; paper state lives in its own subdirectory.
func CheckSimulationState(root string) error {
	dirs := map[string]bool{}
	entries, err := os.ReadDir(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "paper" {
			continue
		}
		dirs[filepath.Join(root, entry.Name())] = true
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	bindingDir := filepath.Join(cache, "tinvest", "density-live")
	bindings, err := os.ReadDir(bindingDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, entry := range bindings {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".lock" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(bindingDir, entry.Name()))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if len(data) == 0 {
			continue
		}
		var binding struct{ Dir string }
		if len(data) > 8192 || json.Unmarshal(data, &binding) != nil || binding.Dir == "" {
			return fmt.Errorf("invalid known density journal binding")
		}
		dirs[binding.Dir] = true
	}
	for dir := range dirs {
		state, exists, err := lastJournalState(filepath.Join(dir, "events.jsonl"))
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		state, err = migrateState(state)
		if err != nil {
			return err
		}
		if len(state.Episodes) > 0 || len(state.Releases) > 0 {
			return fmt.Errorf("unfinished real density trades: restore real management before starting simulation")
		}
	}
	return nil
}

func lastJournalState(path string) (state State, exists bool, retErr error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, false, nil
	}
	if err != nil {
		return state, false, err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return state, true, err
	}
	size := info.Size()
	if size == 0 {
		return state, true, fmt.Errorf("existing density journal has no complete state")
	}
	// Expand only when the last complete frame (or a partial append) exceeds
	// the initial tail window. Journal growth must not disable account guards.
	var data []byte
	var begin, end int
	for window := int64(64 << 10); ; window *= 2 {
		start := max(int64(0), size-window)
		data = make([]byte, size-start)
		n, readErr := f.ReadAt(data, start)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return state, true, readErr
		}
		data = data[:n]
		end = bytes.LastIndexByte(data, '\n')
		if end >= 0 {
			begin = bytes.LastIndexByte(data[:end], '\n') + 1
			if begin > 0 || start == 0 {
				break
			}
		}
		if start == 0 {
			return state, true, fmt.Errorf("density journal has no complete frame")
		}
	}
	var record journalRecord
	if err = json.Unmarshal(data[begin:end], &record); err != nil {
		return state, true, fmt.Errorf("invalid density journal tail: %w", err)
	}
	return record.State, true, nil
}
