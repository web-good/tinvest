package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const dailyResetDir = "/application/data/state/bybit-density"

func openLock(dir string, dailyReset bool) (*os.File, error) {
	if !dailyReset {
		return os.OpenFile(filepath.Join(dir, "runner.lock"), os.O_CREATE|os.O_RDWR, 0600)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}
	if resolved != dir {
		return nil, fmt.Errorf("bybit daily reset directory must not be a symlink: %q", dir)
	}
	// Lock the directory itself so cleanup can remove all of its entries.
	return os.OpenFile(dir, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
}

// NextReset returns the next 03:00 MSK, or zero for a directory without daily reset.
func (j *Journal) NextReset(now time.Time) time.Time {
	if !j.dailyReset {
		return time.Time{}
	}
	// Moscow is UTC+3, so the reset happens at midnight UTC.
	const day = 24 * time.Hour
	return now.UTC().Truncate(day).Add(day)
}

// Reset clears the daily-reset directory and starts a new simulation session.
// The caller must stop the simulator and outbox before calling Reset.
func (j *Journal) Reset() (err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.dailyReset || j.file == nil || j.isDelivering {
		return errors.New("bybit journal is not ready for daily reset")
	}
	if j.failed != nil {
		return j.failed
	}
	root, err := os.OpenRoot(j.lock.Name())
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	locked, err := j.lock.Stat()
	if err != nil {
		return err
	}
	target, err := root.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(locked, target) {
		return errors.New("bybit daily reset directory changed after startup")
	}

	list, err := root.Open(".")
	if err != nil {
		return err
	}
	names, readErr := list.Readdirnames(-1)
	if err := errors.Join(readErr, list.Close()); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			j.failed = err
		}
	}()
	for _, name := range names {
		if err := root.RemoveAll(name); err != nil {
			return err
		}
	}
	replacement, err := root.OpenFile(journalName, os.O_CREATE|os.O_EXCL|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	old := j.file
	j.file = replacement
	defer func() { err = errors.Join(err, old.Close()) }()
	j.sequence = 0
	j.session = 0
	j.pending = nil
	if err := j.append(record{Kind: sessionStarted, Session: 1}); err != nil {
		return err
	}
	return j.lock.Sync()
}
