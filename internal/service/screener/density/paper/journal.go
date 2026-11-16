package paper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Journal struct {
	Dir  string
	file *os.File
}

func NewJournal(root string) (*Journal, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(root, time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Journal{Dir: dir, file: f}, nil
}
func (j *Journal) Record(r Record) error { return j.write(r) }
func (j *Journal) Session(paper Config, detector any, instruments any) error {
	return j.write(struct {
		At                    time.Time
		Kind                  string
		Paper                 Config
		Detector, Instruments any
	}{time.Now().UTC(), "session_start", paper, detector, instruments})
}
func (j *Journal) write(v any) error {
	if j.file == nil {
		return fmt.Errorf("journal closed")
	}
	if err := json.NewEncoder(j.file).Encode(v); err != nil {
		return err
	}
	return j.file.Sync()
}
func (j *Journal) WriteSummary(s Summary) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(j.Dir, ".summary-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	_, writeErr := f.Write(append(data, '\n'))
	syncErr := f.Sync()
	closeErr := f.Close()
	if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(j.Dir, "summary.json"))
}
func (j *Journal) Close() error {
	if j.file == nil {
		return nil
	}
	err := j.file.Close()
	j.file = nil
	return err
}
