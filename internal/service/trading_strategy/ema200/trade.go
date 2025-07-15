package ema200

import (
	"context"
	"fmt"
	"time"
	"tinvest/internal/service/trading_strategy/ema200/dto/input"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context, dto input.Trade) error {
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		fmt.Println(share)
		_, err5 := s.ema.TechAnalyse(ctx, &share.ID, 4, time.Now().AddDate(0, 0, -20), 200)

		if err5 != nil {
			logger.ErrorContext(ctx, "Error in calculate ema5", err5, share.Name)

			continue
		}
	}
	return nil
}
