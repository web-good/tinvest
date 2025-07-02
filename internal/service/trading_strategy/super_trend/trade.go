package super_trend

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	atr "tinvest/internal/domain/atr"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context) error {
	var (
		shares []model.Share
		atrs   map[string]atr.ItemTechAnalyse
	)
	atrs = make(map[string]atr.ItemTechAnalyse)
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		dayMCD, err := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -4)), timestamppb.New(time.Now()), 9)

		if err != nil {
			logger.ErrorContext(ctx, "cannot get day macd", err, share.Name)

			continue
		}

		macDSp := specification.MacDSpecification{}

		if macDSp.IsSatisfiedBy(dayMCD) == false {
			continue
		}

		ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), 5)

		if err5 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema5", err5, share.Name)

			continue
		}

		ema35, err35 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -7), 35)

		if err35 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema35", err35, share.Name)

			continue
		}

		sp := specification.SuperTrendIntersection{}

		if sp.IsSatisfiedBy(ema5, ema35) == false {
			continue
		}

		logger.InfoContext(ctx, "Entered the condition ema intersection", share.Name)
		macDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 11, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 9)
		greenMacDSp := specification.GreenMacD{}

		if greenMacDSp.IsSatisfiedBy(macDModel) == false {
			continue
		}

		logger.InfoContext(ctx, "Entered the condition macd intersection", share.Name)
		shares = append(shares, *share)
		atrTechItem, atrErr := s.atr.TechAnalyse(ctx, &share.ID)

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
