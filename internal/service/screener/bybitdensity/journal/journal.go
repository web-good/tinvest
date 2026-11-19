// Package journal persists Bybit simulation sessions, fills and their notifications.
package journal

import (
	"bufio"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"syscall"
	"time"
	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	sessionStarted = "session_started"
	sessionEnded   = "session_ended"
	fillRecorded   = "fill"
	messageSent    = "message_sent"
)

type position struct {
	Entry     model.Fill `json:"entry"`
	Remaining float64    `json:"remaining"`
}

type record struct {
	ID       uint64      `json:"id"`
	Kind     string      `json:"kind"`
	At       time.Time   `json:"at"`
	Session  uint64      `json:"session,omitempty"`
	Reason   string      `json:"reason,omitempty"`
	Open     []position  `json:"open,omitempty"`
	Fill     *model.Fill `json:"fill,omitempty"`
	Message  string      `json:"message,omitempty"`
	Delivery uint64      `json:"delivery,omitempty"`
}

type notification struct {
	id      uint64
	message string
}

type Journal struct {
	mu           sync.Mutex
	file         *os.File
	lock         *os.File
	sequence     uint64
	session      uint64
	positions    map[model.Key]position
	pending      []notification
	wake         chan struct{}
	done         chan struct{}
	isDelivering bool
	failed       error
}

// Open exclusively owns dir, recovers its journal, and starts a fresh session.
func Open(dir string) (_ *Journal, err error) {
	if dir == "" {
		return nil, errors.New("bybit journal: state directory is required")
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "runner.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, errors.Join(fmt.Errorf("lock bybit journal %q: %w", dir, err), lock.Close())
	}
	j := &Journal{lock: lock, wake: make(chan struct{}, 1), done: make(chan struct{})}
	defer func() {
		if err != nil {
			err = errors.Join(err, j.release())
		}
	}()
	j.file, err = os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	if err := j.replay(); err != nil {
		return nil, err
	}
	if err := syncDirectories(dir); err != nil {
		return nil, fmt.Errorf("sync bybit journal directory: %w", err)
	}
	if j.session != 0 {
		if err := j.endSession("interrupted"); err != nil {
			return nil, err
		}
	}
	if err := j.append(record{Kind: sessionStarted, Session: j.sequence + 1}); err != nil {
		return nil, err
	}
	return j, nil
}

func syncDirectories(dir string) error {
	for {
		f, err := os.Open(dir)
		if err != nil {
			return err
		}
		if err := errors.Join(f.Sync(), f.Close()); err != nil {
			return err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func (j *Journal) replay() error {
	reader := bufio.NewReader(j.file)
	var offset int64
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
			// Every commit includes its newline in the same write. A partial
			// final row could not have been acknowledged by the caller.
			if err := j.file.Truncate(offset); err != nil {
				return fmt.Errorf("truncate incomplete bybit journal row: %w", err)
			}
			return j.file.Sync()
		}
		if err != nil {
			return fmt.Errorf("read bybit journal: %w", err)
		}
		var r record
		if err := json.Unmarshal(line, &r); err != nil {
			return fmt.Errorf("decode bybit journal row %d: %w", j.sequence+1, err)
		}
		if err := j.validate(r); err != nil {
			return fmt.Errorf("invalid bybit journal row %d: %w", j.sequence+1, err)
		}
		j.apply(r)
		offset += int64(len(line))
	}
}

// Record commits a fill and its optional outbox message in the same journal row.
func (j *Journal) Record(fill model.Fill, message string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.append(record{Kind: fillRecorded, Session: j.session, Fill: &fill, Message: message}); err != nil {
		return err
	}
	if message != "" {
		select {
		case j.wake <- struct{}{}:
		default:
		}
	}
	return nil
}

func (j *Journal) append(r record) error {
	if j.file == nil {
		return errors.New("bybit journal is closed")
	}
	if j.failed != nil {
		return j.failed
	}
	r.ID = j.sequence + 1
	r.At = time.Now().UTC()
	if err := j.validate(r); err != nil {
		return fmt.Errorf("record bybit journal %s: %w", r.Kind, err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode bybit journal %s: %w", r.Kind, err)
	}
	if _, err = j.file.Write(append(data, '\n')); err == nil {
		err = j.file.Sync()
	}
	if err != nil {
		j.failed = fmt.Errorf("persist bybit journal %s: %w", r.Kind, err)
		return j.failed
	}
	j.apply(r)
	return nil
}

func (j *Journal) validate(r record) error {
	if r.ID != j.sequence+1 {
		return errors.New("record id is out of sequence")
	}
	switch r.Kind {
	case sessionStarted:
		if j.session != 0 || r.Session != r.ID {
			return errors.New("unexpected session start")
		}
	case sessionEnded:
		if j.session == 0 || r.Session != j.session {
			return errors.New("unexpected session end")
		}
	case fillRecorded:
		if j.session == 0 || r.Session != j.session || r.Fill == nil || !(r.Fill.Qty > 0) {
			return errors.New("invalid fill")
		}
		current, exists := j.positions[r.Fill.Key]
		if r.Fill.Entry == exists || (exists && current.Entry.Side != r.Fill.Side) {
			return fmt.Errorf("unexpected fill for %s:%s", r.Fill.Key.Market, r.Fill.Key.Symbol)
		}
	case messageSent:
		if len(j.pending) == 0 || r.Delivery != j.pending[0].id {
			return errors.New("unexpected delivery acknowledgement")
		}
	default:
		return fmt.Errorf("unknown record kind %q", r.Kind)
	}
	return nil
}

func (j *Journal) apply(r record) {
	j.sequence = r.ID
	switch r.Kind {
	case sessionStarted:
		j.session = r.Session
		j.positions = make(map[model.Key]position)
	case sessionEnded:
		j.session = 0
		j.positions = nil
	case fillRecorded:
		fill := *r.Fill
		switch {
		case fill.Entry:
			j.positions[fill.Key] = position{Entry: fill, Remaining: fill.Qty}
		case fill.Partial:
			current := j.positions[fill.Key]
			current.Remaining -= fill.Qty
			j.positions[fill.Key] = current
		default:
			delete(j.positions, fill.Key)
		}
		if r.Message != "" {
			j.pending = append(j.pending, notification{id: r.ID, message: r.Message})
		}
	case messageSent:
		j.pending[0] = notification{}
		j.pending = j.pending[1:]
		if len(j.pending) == 0 {
			j.pending = nil
		}
	}
}

func (j *Journal) endSession(reason string) error {
	open := make([]position, 0, len(j.positions))
	for _, p := range j.positions {
		open = append(open, p)
	}
	slices.SortFunc(open, func(a, b position) int {
		return cmp.Or(cmp.Compare(a.Entry.Key.Market, b.Entry.Key.Market), cmp.Compare(a.Entry.Key.Symbol, b.Entry.Key.Symbol))
	})
	return j.append(record{Kind: sessionEnded, Session: j.session, Reason: reason, Open: open})
}

// Close records unresolved positions and releases the directory lock.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	close(j.done)
	return errors.Join(j.endSession("shutdown"), j.release())
}

func (j *Journal) release() error {
	var fileErr error
	if j.file != nil {
		fileErr = j.file.Close()
		j.file = nil
	}
	return errors.Join(fileErr, j.lock.Close())
}
