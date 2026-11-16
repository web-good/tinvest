package live

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Store struct {
	Dir        string
	file, lock *os.File
	state      State
	failed     error
}
type journalRecord struct {
	Event string
	State State
}

func AccountHash(account string) string {
	sum := sha256.Sum256([]byte(account))
	return hex.EncodeToString(sum[:])
}

func OpenStore(root, account string) (s *Store, err error) {
	if strings.TrimSpace(account) == "" {
		return nil, fmt.Errorf("explicit account is required")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	lockDir := filepath.Join(cache, "tinvest", "density-live")
	if err = os.MkdirAll(lockDir, 0700); err != nil {
		return nil, err
	}
	hash := AccountHash(account)
	lock, err := os.OpenFile(filepath.Join(lockDir, hash+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("another density-live process owns this account: %w", err)
	}
	s = &Store{Dir: filepath.Join(root, hash), lock: lock, state: State{Version: 2, AccountHash: hash, Cooldown: map[string]time.Time{}, PnLComplete: true}}
	opened := s
	defer func() {
		if err != nil {
			err = errors.Join(err, opened.Close())
			s = nil
		}
	}()
	s.Dir, err = filepath.Abs(s.Dir)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(lock, 8193))
	if err != nil {
		return nil, err
	}
	newBinding := len(data) == 0
	if !newBinding {
		var binding struct{ Dir string }
		if len(data) > 8192 || json.Unmarshal(data, &binding) != nil || binding.Dir != s.Dir {
			return nil, fmt.Errorf("account already bound to another state directory; reuse the original absolute -state-dir for recovery")
		}
	} else {
		binding, marshalErr := json.Marshal(struct{ Dir string }{s.Dir})
		if marshalErr != nil {
			return nil, marshalErr
		}
		if _, err = lock.Write(binding); err != nil {
			return nil, err
		}
		if err = lock.Sync(); err != nil {
			return nil, err
		}
	}
	if err = os.MkdirAll(s.Dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(s.Dir, "events.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	s.file = f
	reader := bufio.NewReader(f)
	records := 0
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			if line[len(line)-1] != '\n' {
				return nil, fmt.Errorf("incomplete live journal; manual reconciliation required")
			}
			var rec journalRecord
			if err = json.Unmarshal(line, &rec); err != nil {
				return nil, fmt.Errorf("invalid live journal: %w", err)
			}
			if (rec.State.Version != 1 && rec.State.Version != 2) || rec.State.AccountHash != hash || rec.State.Cooldown == nil {
				return nil, fmt.Errorf("live journal account/schema mismatch")
			}
			s.state = rec.State
			records++
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	oldVersion := s.state.Version
	s.state, err = migrateState(s.state)
	if err != nil {
		return nil, err
	}
	if records > 0 && oldVersion == 1 {
		if err = s.Save(s.state, "journal_migrated_v2"); err != nil {
			return nil, err
		}
	}
	if records == 0 {
		if !newBinding {
			return nil, fmt.Errorf("previous live journal is missing or empty; manual reconciliation required")
		}
		if err = s.Save(s.state, "journal_created"); err != nil {
			return nil, err
		}
	}
	// File fsync alone does not make newly-created directory entries durable.
	if err = lock.Sync(); err != nil {
		return nil, err
	}
	if err = f.Sync(); err != nil {
		return nil, err
	}
	if err = syncDirectoryChain(lockDir); err != nil {
		return nil, err
	}
	if err = syncDirectoryChain(s.Dir); err != nil {
		return nil, err
	}
	return s, nil
}

func syncDirectoryChain(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		dir, err := os.Open(absolute)
		if err != nil {
			return err
		}
		info, statErr := dir.Stat()
		if statErr != nil || !info.IsDir() {
			return errors.Join(fmt.Errorf("durability path is not a directory"), statErr, dir.Close())
		}
		err = errors.Join(dir.Sync(), dir.Close())
		if err != nil {
			return fmt.Errorf("sync live journal directory: %w", err)
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return nil
		}
		absolute = parent
	}
}
func (s *Store) State() State { return s.state }
func (s *Store) Save(state State, event string) error {
	if s.failed != nil {
		return s.failed
	}
	if s.file == nil {
		return fmt.Errorf("live journal is closed")
	}
	if state.Version != 2 || state.AccountHash != s.state.AccountHash {
		return fmt.Errorf("live journal binding changed")
	}
	state.LastEvent = event
	state.Updated = time.Now().UTC()
	data, err := json.Marshal(journalRecord{Event: event, State: state})
	if err == nil {
		data = append(data, '\n')
		var n int
		n, err = s.file.Write(data)
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
	}
	if err == nil {
		err = s.file.Sync()
	}
	if err != nil {
		s.failed = fmt.Errorf("live journal: %w", err)
		return s.failed
	}
	s.state = state
	return nil
}
func (s *Store) Close() error {
	var errs []error
	if s.file != nil {
		errs = append(errs, s.file.Close())
		s.file = nil
	}
	if s.lock != nil {
		errs = append(errs, syscall.Flock(int(s.lock.Fd()), syscall.LOCK_UN), s.lock.Close())
		s.lock = nil
	}
	return errors.Join(errs...)
}
