package ema200

import (
	"context"
	"fmt"
	"time"
	"tinvest/internal/service/trading_strategy/ema200/dto/input"
	"tinvest/internal/service/trading_strategy/ema200/specification"
	"tinvest/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) Trade(ctx context.Context, dto input.Trade) error {
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		fmt.Println(share)

		rsiModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, int(dto.Interval), timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 14)

		if err != nil {
			logger.ErrorContext(ctx, "Failed to get rsi", share.Name)
		}

		rsiS := specification.RsiSpecification{}

		if !rsiS.IsSatisfiedBy(rsiModel) {
			continue
		}

		macDModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseMacD(ctx, share.ID, int(dto.Interval), timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 9)

		if err != nil {
			logger.ErrorContext(ctx, "Failed to get macd", share.Name)
		}

		macDSpecification := specification.MacDSpecification{}

		if !macDSpecification.IsSatisfiedBy(macDModel) {
			continue
		}

		ema200, err200 := s.ema.TechAnalyse(ctx, &share.ID, int32(dto.Interval), time.Now().AddDate(0, 0, -20), time.Now(), 200)

		if err200 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema200", err200, share.Name)

			continue
		}

		ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -2), time.Now(), 5)

		if err5 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema200", err5, share.Name)

			continue
		}

		ov := specification.Over200ema{}

		if !ov.IsSatisfiedBy(ema5, ema200) {
			continue
		}

	}
	return nil
}
