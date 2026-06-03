package scalping

import (
	"context"

	"github.com/golang/protobuf/ptypes/timestamp"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

type Scalping interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type instrumentsClient interface {
	Shares(ctx context.Context) ([]*imodel.Share, error)
}

type marketDataClient interface {
	GetCandles(ctx context.Context, instrumentUid *string, interval int32, from *timestamp.Timestamp, to *timestamp.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error)
}

type operationsClient interface {
	GetPortfolio(ctx context.Context, accountID string) ([]*grpcmodel.Position, error)
}

type Option func(*service)

func WithSettings(s model.Settings) Option {
	return func(svc *service) {
		svc.settings = s
	}
}

// WithStrategies overrides the per-share strategy set (used in tests).
func WithStrategies(s []strategy.Strategy) Option {
	return func(svc *service) {
		svc.strategies = s
	}
}

type service struct {
	instrumentsClient instrumentsClient
	marketDataClient  marketDataClient
	operationsClient  operationsClient
	tgClient          telegram.Client
	accountID         string
	settings          model.Settings
	strategies        []strategy.Strategy
}

func NewService(
	instrumentsClient instrumentsClient,
	marketDataClient marketDataClient,
	operationsClient operationsClient,
	tgClient telegram.Client,
	accountID string,
	opts ...Option,
) *service {
	svc := &service{
		instrumentsClient: instrumentsClient,
		marketDataClient:  marketDataClient,
		operationsClient:  operationsClient,
		tgClient:          tgClient,
		accountID:         accountID,
		settings:          model.DefaultSettings(),
		strategies:        defaultStrategies(),
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}
