package dividend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
	"tinvest/pkg/client/telegram"
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
		return fmt.Errorf("dividend screener: fetch shares: %w", err)
	}

	sharesByAsset := make(map[string][]*model.Share, len(shares))
	for _, sh := range shares {
		if !sh.DivYieldFlag || sh.AssetUID == "" {
			continue
		}
		sharesByAsset[sh.AssetUID] = append(sharesByAsset[sh.AssetUID], sh)
	}

	uids := make([]string, 0, len(sharesByAsset))
	for assetUID := range sharesByAsset {
		uids = append(uids, assetUID)
	}

	funds, err := s.client.GetAssetFundamentals(ctx, uids)
	if err != nil {
		return fmt.Errorf("dividend screener: fetch fundamentals: %w", err)
	}

	scored := rank.Rank(funds, nil, s.cfg) // TODO(task 5/6): pass real sector map

	// Разделить на выживших (по порядку) и посчитать перцентильный ранг.
	survivors := make([]rank.ScoredCompany, 0, len(scored))
	stats := Stats{Universe: len(uids), ByReason: map[string]int{}}
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
	for _, sc := range survivors {
		instruments := sharesByAsset[sc.AssetUID]
		if len(instruments) == 0 {
			continue
		}
		ranked = append(ranked, RankedShare{Share: instruments[0], Scored: sc})
		bonus := bonusFromScore(sc.Composite, s.cfg)
		for _, sh := range instruments {
			bonusByID[sh.ID] = bonus
		}
	}

	s.mu.Lock()
	s.ranked = ranked
	s.bonusByID = bonusByID
	s.stats = stats
	s.loadedAt = time.Now()
	s.mu.Unlock()
	return nil
}

// bonusFromScore отображает композит (0..100) в фундаментальный бонус Golden X.
// Пороги — точки калибровки (см. live-шаг); полосы абсолютны, чтобы бонус не
// зависел от состава вселенной.
func bonusFromScore(composite float64, cfg rank.Config) int {
	switch {
	case composite >= cfg.BonusScoreT3:
		return 3
	case composite >= cfg.BonusScoreT2:
		return 2
	case composite >= cfg.BonusScoreT1:
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

// Top returns the top-n ranked shares plus gate stats.
// n<=0 falls back to defaultTopN. If fewer than n shares are ranked, all are returned.
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

// Send рендерит топ-N дивидендных акций и отправляет в Telegram.
func (s *service) Send(ctx context.Context, tg telegram.Client) error {
	ranked, stats, err := s.Top(ctx, defaultTopN)
	if err != nil {
		return err
	}
	return tg.SendMessage(Render(ranked, stats))
}
