// Package accountcoord coordinates continuous and candle strategies on one account.
package accountcoord

import (
	"context"
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
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

var ErrBusy = errors.New("account or instrument is reserved")

type Claim struct {
	Ticker, InstrumentID, Owner, Key, Status string
	Updated                                  time.Time
}
type registry struct {
	Version     int
	AccountHash string
	Claims      map[string]Claim
}
type Coordinator struct {
	root, hash      string
	legacy          statestore.Store
	initErr         error
	guard           func() error
	densitySnapshot func(context.Context) (map[string]DensityPosition, error)
}
type Lease struct {
	coordinator *Coordinator
	file        *os.File
	kind        string
}

func New(root, account string, legacy statestore.Store) *Coordinator {
	sum := sha256.Sum256([]byte(account))
	hash := hex.EncodeToString(sum[:])
	absolute, err := filepath.Abs(root)
	if strings.TrimSpace(root) == "" || strings.TrimSpace(account) == "" {
		err = fmt.Errorf("shared account and state root are required")
	}
	return &Coordinator{root: filepath.Join(absolute, hash), hash: hash, legacy: legacy, initErr: err}
}
func (c *Coordinator) Root() string { return c.root }

// HasDensityBinding reports whether a real Density writer has reserved its journal.
func (c *Coordinator) HasDensityBinding() (bool, error) {
	_, err := os.Stat(filepath.Join(c.root, "density.lock"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// SetGuard installs a read-only binding check before either account participant
// may obtain an entry lease. Configure it before starting workers.
func (c *Coordinator) SetGuard(guard func() error) { c.guard = guard }
func (c *Coordinator) AccountHash() string         { return c.hash }
func (c *Coordinator) MatchesAccount(account string) bool {
	sum := sha256.Sum256([]byte(account))
	return c.hash == hex.EncodeToString(sum[:])
}
func (c *Coordinator) Lock(ctx context.Context) (*Lease, error) {
	if c.guard != nil {
		if err := c.guard(); err != nil {
			return nil, err
		}
	}
	f, err := c.lock(ctx, "entry.lock", false)
	if err != nil {
		return nil, err
	}
	return &Lease{coordinator: c, file: f, kind: "entry"}, nil
}
func (c *Coordinator) TryLock() (*Lease, error) {
	if c.guard != nil {
		if err := c.guard(); err != nil {
			return nil, err
		}
	}
	f, err := c.lock(context.Background(), "entry.lock", true)
	if err != nil {
		return nil, err
	}
	return &Lease{coordinator: c, file: f, kind: "entry"}, nil
}
func (l *Lease) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := errors.Join(syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN), l.file.Close())
	l.file = nil
	return err
}
func (c *Coordinator) valid(l *Lease) bool {
	return l != nil && l.coordinator == c && l.file != nil && l.kind == "entry"
}

// LockDensity excludes journal writers across containers sharing the state volume.
// It is independent of the entry lock, so candle passes never delay safety exits.
func (c *Coordinator) LockDensity(ctx context.Context, journalRoot string) (lease *Lease, err error) {
	f, err := c.lock(ctx, "density.lock", true)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, syscall.Flock(int(f.Fd()), syscall.LOCK_UN), f.Close())
		}
	}()
	absolute, err := filepath.Abs(journalRoot)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(journalRoot) == "" {
		return nil, fmt.Errorf("density journal root required")
	}
	data, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil {
		return nil, err
	}
	var binding struct{ Journal string }
	if len(data) > 0 {
		if len(data) > 8192 || json.Unmarshal(data, &binding) != nil || binding.Journal != absolute {
			return nil, fmt.Errorf("shared density is bound to another journal directory")
		}
	} else {
		binding.Journal = absolute
		data, err = json.Marshal(binding)
		if err != nil {
			return nil, err
		}
		var n int
		n, err = f.WriteAt(data, 0)
		if err != nil {
			return nil, err
		}
		if n != len(data) {
			return nil, io.ErrShortWrite
		}
	}
	if err = f.Sync(); err != nil {
		return nil, err
	}
	if err = syncParents(c.root); err != nil {
		return nil, err
	}
	return &Lease{coordinator: c, file: f, kind: "density"}, nil
}

func (c *Coordinator) lock(ctx context.Context, name string, try bool) (*os.File, error) {
	if c.initErr != nil {
		return nil, c.initErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.root, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(c.root, name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	timer := time.NewTicker(20 * time.Millisecond)
	defer timer.Stop()
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			_ = f.Close()
			return nil, err
		}
		if try {
			_ = f.Close()
			return nil, ErrBusy
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Coordinator) transaction(ctx context.Context, fn func(map[string]Claim) (bool, error)) (retErr error) {
	lock, err := c.lock(ctx, "registry.lock", false)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, syscall.Flock(int(lock.Fd()), syscall.LOCK_UN), lock.Close()) }()
	path := filepath.Join(c.root, "owners.json")
	data, err := os.ReadFile(path)
	r := registry{Version: 2, AccountHash: c.hash, Claims: map[string]Claim{}}
	created := false
	if errors.Is(err, os.ErrNotExist) {
		info, statErr := lock.Stat()
		if statErr != nil {
			return statErr
		}
		if info.Size() != 0 {
			return fmt.Errorf("shared ownership registry is missing; automatic reconstruction forbidden")
		}
		created = true
	} else if err != nil {
		return err
	} else if err = json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("invalid ownership registry: %w", err)
	}
	if (r.Version != 1 && r.Version != 2) || r.AccountHash != c.hash || r.Claims == nil {
		return fmt.Errorf("ownership registry binding/schema mismatch")
	}
	for ticker, claim := range r.Claims {
		if !validClaim(claim) || ticker != claim.Ticker && ticker != claimKey(claim) {
			return fmt.Errorf("invalid ownership record")
		}
	}
	migrated := r.Version == 1
	r.Version = 2
	normalized := map[string]Claim{}
	for _, claim := range r.Claims {
		if _, exists := normalized[claimKey(claim)]; exists {
			return fmt.Errorf("duplicate ownership bucket")
		}
		normalized[claimKey(claim)] = claim
	}
	r.Claims = normalized
	changed, err := fn(r.Claims)
	if err != nil {
		return err
	}
	if changed || created || migrated {
		data, err = json.Marshal(r)
		if err != nil {
			return err
		}
		if err = atomicWrite(path, data); err != nil {
			return err
		}
	}
	// A separate persistent marker makes deletion of owners.json a hard error.
	if _, err = lock.WriteAt([]byte("initialized\n"), 0); err != nil {
		return err
	}
	if err = lock.Sync(); err != nil {
		return err
	}
	return syncParents(c.root)
}
func validClaim(claim Claim) bool {
	return claim.Ticker != "" && claim.Ticker == strings.ToUpper(strings.TrimSpace(claim.Ticker)) && claim.InstrumentID != "" && claim.Owner != "" && claim.Key != "" && (claim.Status == "active" || claim.Status == "closed")
}

// Claims returns ticker keys, with density:TICKER for a second owner.
// Trading participants should use the owner-specific views instead.
func (c *Coordinator) Claims(ctx context.Context) (map[string]Claim, error) {
	out := map[string]Claim{}
	err := c.transaction(ctx, func(m map[string]Claim) (bool, error) {
		for k, v := range m {
			out[k] = v
		}
		return false, nil
	})
	for k, v := range out {
		if v.Owner == "density" {
			if _, exists := out[v.Ticker]; !exists {
				out[v.Ticker] = v
				delete(out, k)
			}
		}
	}
	return out, err
}
func claimKey(v Claim) string {
	if v.Owner == "density" {
		return "density:" + v.Ticker
	}
	return v.Ticker
}
func (c *Coordinator) CanEnter(l *Lease, ticker, owner string) (bool, error) {
	if !c.valid(l) {
		return false, fmt.Errorf("entry lease required")
	}
	if owner != "density" && c.legacy != nil {
		state, err := c.legacy.Load()
		if err != nil {
			return false, err
		}
		if _, ok := state[ticker]; ok {
			return false, nil
		}
	}
	claims, err := c.Claims(context.Background())
	if err != nil {
		return false, err
	}
	for _, claim := range claims {
		if claim.Ticker == ticker && (claim.Owner == "density") == (owner == "density") && claim.Status == "active" {
			return false, nil
		}
	}
	return true, nil
}
func (c *Coordinator) Reserve(l *Lease, claim Claim) error {
	if !c.valid(l) {
		return fmt.Errorf("entry lease required")
	}
	claim.Status = "active"
	claim.Updated = time.Now().UTC()
	if !validClaim(claim) {
		return fmt.Errorf("invalid ownership reservation")
	}
	return c.transaction(context.Background(), func(m map[string]Claim) (bool, error) {
		for _, old := range m {
			if old.Status == "active" && old.InstrumentID == claim.InstrumentID && old.Ticker != claim.Ticker && (old.Owner == "density") == (claim.Owner == "density") {
				return false, ErrBusy
			}
		}
		if old, ok := m[claimKey(claim)]; ok && old.Status == "active" {
			if old.Key == claim.Key && old.Owner == claim.Owner && old.InstrumentID == claim.InstrumentID {
				return false, nil
			}
			return false, ErrBusy
		}
		m[claimKey(claim)] = claim
		return true, nil
	})
}
func (c *Coordinator) Verify(ctx context.Context, want Claim) error {
	claims, err := c.ownerClaims(ctx, want.Owner == "density")
	if err != nil {
		return err
	}
	got, ok := claims[want.Ticker]
	if !ok || got.Status != "active" || got.Key != want.Key || got.Owner != want.Owner || got.InstrumentID != want.InstrumentID {
		return fmt.Errorf("shared ownership does not match unresolved episode")
	}
	return nil
}
func (c *Coordinator) MarkClosed(ctx context.Context, want Claim) error {
	return c.transaction(ctx, func(m map[string]Claim) (bool, error) {
		old, ok := m[claimKey(want)]
		if !ok || old.Key != want.Key || old.Owner != want.Owner {
			return false, nil
		}
		old.Status = "closed"
		old.Updated = time.Now().UTC()
		m[claimKey(want)] = old
		return true, nil
	})
}
func (c *Coordinator) Remove(l *Lease, want Claim) error {
	if !c.valid(l) {
		return fmt.Errorf("entry lease required")
	}
	return c.transaction(context.Background(), func(m map[string]Claim) (bool, error) {
		old, ok := m[claimKey(want)]
		if !ok {
			return false, nil
		}
		if old.Key != want.Key || old.Owner != want.Owner {
			return false, ErrBusy
		}
		delete(m, claimKey(want))
		return true, nil
	})
}

func atomicWrite(path string, data []byte) (retErr error) {
	f, err := os.CreateTemp(filepath.Dir(path), "ownership-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	n, err := f.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	return syncParents(filepath.Dir(path))
}
func syncParents(path string) error {
	for {
		d, err := os.Open(path)
		if err != nil {
			return err
		}
		err = errors.Join(d.Sync(), d.Close())
		if err != nil {
			return err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

// DensityPosition is a durable read-only view of density inventory and its orders.
type DensityPosition struct {
	InstrumentID      string
	Quantity          int64
	StopIDs, OrderIDs []string
}

func (c *Coordinator) SetDensitySnapshot(f func(context.Context) (map[string]DensityPosition, error)) {
	c.densitySnapshot = f
}
func (c *Coordinator) DensitySnapshot(ctx context.Context) (map[string]DensityPosition, error) {
	if c.densitySnapshot == nil {
		return nil, nil
	}
	return c.densitySnapshot(ctx)
}
func (c *Coordinator) LegacyState() (map[string]statestore.Entry, error) {
	if c.legacy == nil {
		return map[string]statestore.Entry{}, nil
	}
	return c.legacy.Load()
}
func (c *Coordinator) ownerClaims(ctx context.Context, density bool) (map[string]Claim, error) {
	out := map[string]Claim{}
	err := c.transaction(ctx, func(m map[string]Claim) (bool, error) {
		for _, v := range m {
			if (v.Owner == "density") == density {
				out[v.Ticker] = v
			}
		}
		return false, nil
	})
	return out, err
}
func (c *Coordinator) DensityClaims(ctx context.Context) (map[string]Claim, error) {
	return c.ownerClaims(ctx, true)
}
func (c *Coordinator) CandleClaims(ctx context.Context) (map[string]Claim, error) {
	return c.ownerClaims(ctx, false)
}
