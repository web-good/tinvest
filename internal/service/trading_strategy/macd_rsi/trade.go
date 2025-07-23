package macd_rsi

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/enum"
	"tinvest/internal/service/trading_strategy/macd_rsi/notification"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	var (
		shares []model.Share
		atrs   map[string]atr.ItemTechAnalyse
	)
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	atrs = make(map[string]atr.ItemTechAnalyse)
	macdSpecification := specification.Macd{SearchArea: in.SearchArea}
	rsiSpecification := specification.RsiTrade{SearchArea: in.SearchArea, Value: 50}
	rsi2Specification := specification.RsiTrade{SearchArea: in.SearchArea, Value: 70}

	for _, share := range t {
		dateFrom := timestamppb.New(time.Now().AddDate(0, 0, -1))
		dateTo := timestamppb.New(time.Now())

		if in.Interval == enum.Hour1 {
			dateFrom = timestamppb.New(time.Now().AddDate(0, 0, -1))
		}

		if in.Interval == enum.Hour4 {
			dateFrom = timestamppb.New(time.Now().AddDate(0, 0, -3))
		}

		macDModel, errM := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, int(in.Interval), dateFrom, dateTo, in.MACDLength)

		if errM != nil {
			logger.ErrorContext(ctx, "Failed to get macd", share.Name)

			continue
		}

		rsiModel, errR := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, int(in.Interval), dateFrom, dateTo, in.RSILength)

		if errR != nil {
			logger.ErrorContext(ctx, "Failed to get rsi", share.Name)

			continue
		}

		rsi2Model, errR := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, int(in.Interval), dateFrom, dateTo, in.RSIFastLength)

		if errR != nil {
			logger.ErrorContext(ctx, "Failed to get rsi", share.Name)

			continue
		}

		if macdSpecification.IsSatisfiedBy(macDModel) != true || rsiSpecification.IsSatisfiedBy(rsiModel) != true || rsi2Specification.IsSatisfiedBy(rsi2Model) != true {
			continue
		}

		shares = append(shares, *share)
		atrTechItem, atrErr := s.atrInstrument.TechAnalyse(ctx, &share.ID)

		if atrErr != nil {
			logger.ErrorContext(ctx, "Failed to get ATR", share.Name)
		} else {
			atrs[share.ID] = atrTechItem
		}
	}

	if len(shares) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(shares, atrs, in.Interval))
		shares = []model.Share{}

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}
