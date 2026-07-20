package dividend

import (
	"context"
	"sync"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
)

type service struct {
	client instrumentsClient
	cfg    rank.Config
	ttl    time.Duration

	mu        sync.RWMutex
	bonusByID map[string]int
	ranked    []RankedShare
	stats     Stats
	loadedAt  time.Time
}

func NewService(client instrumentsClient, opts ...Option) *service {
	s := &service{
		client: client,
		cfg:    rank.DefaultConfig(),
		ttl:    defaultTTL,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *service) ensureFresh(ctx context.Context) error {
	s.mu.RLock()
	fresh := !s.loadedAt.IsZero() && time.Since(s.loadedAt) < s.ttl
	s.mu.RUnlock()
	if fresh {
		return nil
	}
	return s.refresh(ctx)
}

func (s *service) refresh(ctx context.Context) error {
	shares, err := s.client.Shares(ctx)
	if err != nil {
		return err
	}

	dividendShares := make([]*model.Share, 0, len(shares))
	uids := make([]string, 0, len(shares))
	shareByAsset := make(map[string]*model.Share, len(shares))
	for _, sh := range shares {
		if !sh.DivYieldFlag || sh.AssetUid == "" {
			continue
		}
		dividendShares = append(dividendShares, sh)
		uids = append(uids, sh.AssetUid)
		shareByAsset[sh.AssetUid] = sh
	}

	funds, err := s.client.GetAssetFundamentals(ctx, uids)
	if err != nil {
		return err
	}

	scored := rank.Rank(funds, s.cfg)

	// Разделить на выживших (по порядку) и посчитать перцентильный ранг.
	survivors := make([]rank.ScoredCompany, 0, len(scored))
	stats := Stats{Universe: len(dividendShares), ByReason: map[string]int{}}
	for _, sc := range scored {
		if sc.GateReason != "" {
			stats.Gated++
			stats.ByReason[sc.GateReason]++
			continue
		}
		survivors = append(survivors, sc)
	}
	stats.Ranked = len(survivors)

	ranked := make([]RankedShare, 0, len(survivors))
	bonusByID := make(map[string]int, len(survivors))
	total := len(survivors)
	for i, sc := range survivors {
		sh := shareByAsset[sc.AssetUid]
		if sh == nil {
			continue
		}
		ranked = append(ranked, RankedShare{Share: sh, Scored: sc})
		bonusByID[sh.ID] = bonusFromRank(i, total)
	}

	s.mu.Lock()
	s.ranked = ranked
	s.bonusByID = bonusByID
	s.stats = stats
	s.loadedAt = time.Now()
	s.mu.Unlock()
	return nil
}

// bonusFromRank: топ-дециль →3, топ-квартиль →2, топ-половина →1, иначе 0.
// idx 0 — лучший из total.
func bonusFromRank(idx, total int) int {
	if total <= 0 {
		return 0
	}
	q := float64(idx) / float64(total)
	switch {
	case q < 0.10:
		return 3
	case q < 0.25:
		return 2
	case q < 0.50:
		return 1
	default:
		return 0
	}
}

func (s *service) RankBonus(instrumentID string) int {
	if err := s.ensureFresh(context.Background()); err != nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bonusByID[instrumentID]
}

func (s *service) Top(ctx context.Context, n int) ([]RankedShare, Stats, error) {
	if err := s.ensureFresh(ctx); err != nil {
		return nil, Stats{}, err
	}
	if n <= 0 {
		n = defaultTopN
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	top := s.ranked
	if len(top) > n {
		top = top[:n]
	}
	out := make([]RankedShare, len(top))
	copy(out, top)
	return out, s.stats, nil
}
