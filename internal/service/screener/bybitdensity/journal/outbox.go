package journal

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const (
	initialRetryDelay = 250 * time.Millisecond
	maxRetryDelay     = 30 * time.Second
	sendTimeout       = 5 * time.Second
)

// Deliver sends queued messages in order and retries transport failures.
// Cancel the context and wait for Deliver before closing the journal.
func (j *Journal) Deliver(ctx context.Context, send func(context.Context, string) error) error {
	if send == nil {
		return errors.New("bybit outbox: sender is required")
	}
	j.mu.Lock()
	if j.isDelivering {
		j.mu.Unlock()
		return errors.New("bybit outbox: delivery worker is already running")
	}
	j.isDelivering = true
	j.mu.Unlock()
	defer func() {
		j.mu.Lock()
		j.isDelivering = false
		j.mu.Unlock()
	}()
	backoff := initialRetryDelay
	for ctx.Err() == nil {
		msg, ok, err := j.next()
		if err != nil {
			return err
		}
		if !ok {
			select {
			case <-ctx.Done():
				return nil
			case <-j.done:
				return nil
			case <-j.wake:
			}
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
		err = send(sendCtx, msg.message)
		cancel()
		if err == nil {
			if err := j.acknowledge(msg.id); err != nil {
				return err
			}
			backoff = initialRetryDelay
			continue
		}
		if ctx.Err() != nil {
			return nil
		}
		slog.WarnContext(ctx, "bybit simulation notification failed", "id", msg.id, "retry_in", backoff, "error", err)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-j.done:
			timer.Stop()
			return nil
		case <-timer.C:
			backoff = min(2*backoff, maxRetryDelay)
		}
	}
	return nil
}

func (j *Journal) next() (notification, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil || j.failed != nil || len(j.pending) == 0 {
		return notification{}, false, j.failed
	}
	return j.pending[0], true, nil
}

func (j *Journal) acknowledge(id uint64) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	return j.append(record{Kind: messageSent, Delivery: id})
}
