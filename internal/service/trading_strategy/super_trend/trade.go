package super_trend

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	notification2 "tinvest/internal/domain/notification"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context) error {
	var (
		n []notification2.SuperTrend
	)

	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		hMacD, err := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -8)), timestamppb.New(time.Now()), 12)

		if err != nil {
			logger.ErrorContext(ctx, "cannot get day macd", err, share.Name)

			continue
		}

		rsiModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()))

		if err != nil {
			logger.ErrorContext(ctx, "Failed to get rsi", share.Name)
		}

		macDSp := specification.GreenMacD{}

		if macDSp.IsSatisfiedBy(hMacD) == false {
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

		rsiS := specification.RsiSpecification{}

		if rsiS.IsSatisfiedBy(rsiModel) != true {
			continue
		}

		greenMacDSp := specification.GreenMacD{}
		macDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 11, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 9)
		greenMacDSp.IsSatisfiedBy(macDModel)
		notif := notification2.SuperTrend{
			Share: *share,
		}

		if greenMacDSp.IsSatisfiedBy(macDModel) == true {
			notif.Indicator = notification2.Yellow
		}

		dMacDModel, _ := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, 5, timestamppb.New(time.Now().AddDate(0, 0, -20)), timestamppb.New(time.Now()), 9)

		if greenMacDSp.IsSatisfiedBy(dMacDModel) == false {
			continue
		}

		notif.Indicator = notification2.Green
		var atrErr error
		notif.Atr, atrErr = s.atr.TechAnalyse(ctx, &share.ID)

		if atrErr != nil {
			logger.ErrorContext(ctx, "Failed to get ATR", share.Name)
		}

		n = append(n, notif)
	}

	if len(n) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(n))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}
