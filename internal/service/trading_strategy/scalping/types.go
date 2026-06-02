package scalping

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain"
	domainatr "tinvest/internal/domain/atr"
	domainema "tinvest/internal/domain/ema"
	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
)

type Scalping interface {
	Trade(ctx context.Context, in dto.Trade) error
}

type emaInstrument interface {
	TechAnalyse(ctx context.Context, instrumentUid *string, interval int32, from time.Time, to time.Time, period int) ([]domainema.ItemTechAnalyse, error)
}

type rsiInstrument interface {
	CalculateRSI(ctx context.Context, instrumentUid string, interval enum.Interval, dateFrom *timestamppb.Timestamp, dateTo *timestamppb.Timestamp, length int32) ([]*domain.RSIItemTechAnalyse, error)
}

type atrInstrument interface {
	TechAnalyse(ctx context.Context, instrumentUid *string, interval enum.Interval, dateNow time.Time) (domainatr.ItemTechAnalyse, error)
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

type service struct {
	ema               emaInstrument
	rsi               rsiInstrument
	atr               atrInstrument
	instrumentsClient instrumentsClient
	marketDataClient  marketDataClient
	operationsClient  operationsClient
	tgClient          telegram.Client
	accountID         string
	settings          model.Settings
}

func NewService(
	ema emaInstrument,
	rsi rsiInstrument,
	atr atrInstrument,
	instrumentsClient instrumentsClient,
	marketDataClient marketDataClient,
	operationsClient operationsClient,
	tgClient telegram.Client,
	accountID string,
	opts ...Option,
) *service {
	svc := &service{
		ema:               ema,
		rsi:               rsi,
		atr:               atr,
		instrumentsClient: instrumentsClient,
		marketDataClient:  marketDataClient,
		operationsClient:  operationsClient,
		tgClient:          tgClient,
		accountID:         accountID,
		settings:          model.DefaultSettings(),
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}
