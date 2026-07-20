package detector

import (
	"context"
	"errors"
	"fmt"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
	gxmodel "tinvest/internal/service/trading_strategy/golden_x/model"
	"tinvest/pkg/logger"
)

// DetectAll runs Detect on every successfully fetched share. Shares with
// fetch errors or insufficient history are logged and skipped.
func DetectAll(ctx context.Context, fetched []gxmodel.FetchResult, in dto.Trade, settings gxmodel.Settings, bonusFor func(instrumentID string) int) []gxmodel.DetectResult {
	results := make([]gxmodel.DetectResult, 0, len(fetched))
	for _, fr := range fetched {
		if fr.Err != nil {
			logger.ErrorContext(ctx, fmt.Errorf("get candles for %s: %w", fr.Share.Name, fr.Err).Error())
			continue
		}

		sig, detectErr := Detect(fr.Candles, fr.Share.RSILength, in.Kind, in.UseTrendFilter, settings)
		if errors.Is(detectErr, ErrAdaptiveInsufficientHistory) {
			logger.InfoContext(ctx, "adaptive tiers: insufficient history", "share", fr.Share.Name)
			continue
		}
		if errors.Is(detectErr, ErrInsufficientHistory) {
			logger.InfoContext(ctx, "trend filter: insufficient history", "share", fr.Share.Name)
			continue
		}
		if detectErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("detect signal for %s: %w", fr.Share.Name, detectErr).Error())
			continue
		}

		results = append(results, gxmodel.DetectResult{Share: fr.Share, Signal: sig, FundamentalBonus: bonusFor(fr.Share.ID)})
	}
	return results
}
