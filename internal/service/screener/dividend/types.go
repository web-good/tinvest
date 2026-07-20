package dividend

import (
	"context"
	"time"

	"tinvest/internal/model"
	"tinvest/internal/service/screener/dividend/rank"
	"tinvest/pkg/client/telegram"
)

const defaultTTL = 24 * time.Hour
const defaultTopN = 15

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*model.Share, error)
	GetAssetFundamentals(ctx context.Context, assetUIDs []string) ([]*model.Fundamentals, error)
}

// Screener — потребитель бот-команды.
type Screener interface {
	Send(ctx context.Context, tg telegram.Client) error
}

// RankProvider — узкий потребитель Golden X.
type RankProvider interface {
	RankBonus(instrumentID string) int
}

type RankedShare struct {
	Share  *model.Share
	Scored rank.ScoredCompany
}

type Stats struct {
	Universe int
	Ranked   int
	Gated    int
	ByReason map[string]int
}

type Option func(*service)

func WithConfig(c rank.Config) Option { return func(s *service) { s.cfg = c } }
func WithTTL(d time.Duration) Option  { return func(s *service) { s.ttl = d } }
