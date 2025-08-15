package macd_rsi

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	"tinvest/internal/domain/atr"
	enum2 "tinvest/internal/enum"
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
	//rsiSpecification := specification.RsiTrade{Value: 30}
	emaSpecification := specification.EmaIntersection{}

	for _, share := range t {
		dateFrom := timestamppb.New(time.Now().AddDate(0, 0, -2))
		dateTo := timestamppb.New(time.Now())

		if in.LocalInterval == enum.Hour1 {
			dateFrom = timestamppb.New(time.Now().AddDate(0, 0, -2))
		}

		if in.LocalInterval == enum.Hour4 {
			dateFrom = timestamppb.New(time.Now().AddDate(0, 0, -4))
		}

		rsiModel, errR := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, share.ID, int(in.LocalInterval), dateFrom, dateTo, in.RSILength)

		if errR != nil {
			logger.ErrorContext(ctx, "Failed to get rsi", share.Name)

			continue
		}

		if len(rsiModel) == 0 {
			logger.InfoContext(ctx, "Rsi can not get 0 len", share.Name)

			continue
		}

		ema5, err5 := s.ema.TechAnalyse(ctx, &share.ID, int32(in.GlobalInterval), time.Now().AddDate(0, 0, -20), time.Now(), 5)

		if err5 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema5 4h :%w,  %s", err5, share.Name).Error())

			continue
		}

		emaG20, errG20 := s.ema.TechAnalyse(ctx, &share.ID, int32(in.GlobalInterval), time.Now().AddDate(0, 0, -20), time.Now(), 20)

		if errG20 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema50 4h :%w,  %s", emaG20, share.Name).Error())

			continue
		}

		if emaSpecification.IsSatisfiedBy(ema5, emaG20) != true {
			continue
		}

		ema20, err20 := s.ema.TechAnalyse(ctx, &share.ID, int32(in.LocalInterval), time.Now().AddDate(0, 0, -10), time.Now(), 20)

		if err20 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema5 4h :%w,  %s", err20, share.Name).Error())

			continue
		}

		ema50, err50 := s.ema.TechAnalyse(ctx, &share.ID, int32(in.LocalInterval), time.Now().AddDate(0, 0, -20), time.Now(), 50)

		if err50 != nil {
			logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema50 4h :%w,  %s", err50, share.Name).Error())

			continue
		}

		if emaSpecification.IsSatisfiedBy(ema20, ema50) != true {
			continue
		}

		shares = append(shares, *share)
		atrTechItem, atrErr := s.atrInstrument.TechAnalyse(ctx, &share.ID, enum2.Hour1)

		if atrErr != nil {
			logger.ErrorContext(ctx, "Failed to get ATR", share.Name)
		} else {
			atrs[share.ID] = atrTechItem
		}
	}

	if len(shares) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(shares, atrs, in.LocalInterval))
		shares = []model.Share{}

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}
