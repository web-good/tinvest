package macd_rsi

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/macd_rsi/notification"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context) error {
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	var (
		shares []model.Share
		atrs   map[string]atr.ItemTechAnalyse
	)
	atrs = make(map[string]atr.ItemTechAnalyse)

	for _, share := range t {
		//EMA
		//y, _ := s.marketDataServiceGrpcClient.GetTechAnalyseEma(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 200)
		//ema := specification.EmaSpecification{}
		//ema.IsSatisfiedBy(y)
		macDModel4h, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 11, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 9)
		greenMacDSp := specification.GreenMacD{}

		if greenMacDSp.IsSatisfiedBy(macDModel4h) == false {
			continue
		}

		rsiModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()))

		if err != nil {
			logger.ErrorContext(ctx, "Failed to get rsi", share.Name)
		}

		rsiS := specification.RsiSpecification{}
		macDModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 9)

		if err != nil {
			logger.ErrorContext(ctx, "Failed to get macd", share.Name)
		}

		macDSpecification := specification.MacDSpecification{}

		if macDSpecification.IsSatisfiedBy(macDModel) != true || rsiS.IsSatisfiedBy(rsiModel) != true {
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
		err := s.tgClient.SendMessage(notification.Trade(shares, atrs))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}
