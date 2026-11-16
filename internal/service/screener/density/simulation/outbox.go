package simulation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"tinvest/internal/service/screener/density/paper"
)

type message struct {
	ID, Text string
	Fill     *paper.Record `json:",omitempty"`
}
type Outbox struct {
	mu      sync.Mutex
	path    string
	pending []message
	wake    chan struct{}
}

func OpenOutbox(path string) (*Outbox, error) {
	q := &Outbox{path: path, wake: make(chan struct{}, 1)}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(data, &q.pending); err != nil {
			return nil, fmt.Errorf("simulation outbox: %w", err)
		}
	}
	return q, nil
}

func (q *Outbox) save(pending []message) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(q.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(q.path), ".outbox-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	encodeErr := json.NewEncoder(f).Encode(pending)
	syncErr := f.Sync()
	closeErr := f.Close()
	if err = errors.Join(encodeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), q.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(q.path))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

// Record durably queues fills; market processing never waits for Telegram.
func (q *Outbox) Record(r paper.Record) error {
	text := FormatFill(r)
	if text == "" {
		return nil
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%x", sha256.Sum256(data))
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, m := range q.pending {
		if m.ID == id {
			return nil
		}
	}
	snapshot := r
	if r.Order != nil {
		order := *r.Order
		snapshot.Order = &order
	}
	pending := append(append([]message(nil), q.pending...), message{ID: id, Text: text, Fill: &snapshot})
	if err = q.save(pending); err != nil {
		return err
	}
	q.pending = pending
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return nil
}

func (q *Outbox) next() (message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return message{}, false
	}
	return q.pending[0], true
}
func (q *Outbox) ack(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 || q.pending[0].ID != id {
		return nil
	}
	pending := append([]message(nil), q.pending[1:]...)
	if err := q.save(pending); err != nil {
		return err
	}
	q.pending = pending
	return nil
}

// Run retries pending alerts, including alerts restored from previous sessions.
func (q *Outbox) Run(ctx context.Context, notify func(context.Context, string) error, warn func(error)) {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		for {
			m, ok := q.next()
			if !ok || ctx.Err() != nil || notify == nil {
				break
			}
			sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := notify(sendCtx, m.Text)
			cancel()
			if err == nil {
				err = q.ack(m.ID)
			}
			if err != nil {
				if warn != nil {
					warn(err)
				}
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
		case <-timer.C:
		}
	}
}
