package macd_rsi

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sync"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/notification"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

//Trade Условия покупки
/*	1) пересечение MACD ниже отметки 0
	2) цена находится над EMA 200
	3) объёмы выше среднего
	4) RSI ниже 70
*/
func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	info := domain.NewInfo()
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	semaphore := semaphore.New(3)
	var wg sync.WaitGroup

	for _, share := range t {
		wg.Add(1)
		go func(share *model.Share) {
			fmt.Println(share.Name)
			defer func() {
				wg.Done()
				semaphore.Release()
			}()
			semaphore.Acquire()
			result, err := s.do(ctx, in, share)

			if err != nil || result.MACD == nil {
				return
			}

			info.WriteToMap(share.ID, result)
		}(share)
	}

	wg.Wait()

	if len(info.Items()) > 0 {
		err := s.tgClient.SendMessage(notification.Trade(info))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)

			return err
		}
	}

	return nil
}

func (s *service) do(ctx context.Context, in dto.Trade, share *model.Share) (domain.Item, error) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	emaSp := specification.EmaSpecification{}
	interval := int32(3)
	rsiSp := specification.RsiTrade{}
	macDSp := specification.Macd{}
	volumeSp := specification.Volume{}
	ema, errEma := s.ema.TechAnalyse(ctx, &share.ID, int32(in.Interval), utils.TimeGenerator(dateNow, -350, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 200)

	if errEma != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema :%w", errEma).Error())

		return domain.Item{}, errEma
	}

	priceNow, errPriceN := s.marketDataServiceGrpcClient.GetCandles(ctx, &share.ID, int32(in.Interval), timestamppb.New(dateNow), timestamppb.New(dateNow), &interval, true)

	if errPriceN != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate priceNow :%w", errPriceN).Error())

		return domain.Item{}, errPriceN
	}

	if len(priceNow) == 0 {
		logger.ErrorContext(ctx, fmt.Errorf("empty priceNow").Error())

		return domain.Item{}, errPriceN
	}

	if !emaSp.IsSatisfiedBy(ema[len(ema)-2], *priceNow[0]) {
		return domain.Item{}, nil
	}

	rsi, errRsi := s.rsi.CalculateRSI(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -25, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 12)

	if errRsi != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errRsi).Error())

		return domain.Item{}, errRsi
	}

	if len(rsi) == 0 {
		logger.ErrorContext(ctx, fmt.Errorf("empty rsi").Error())

		return domain.Item{}, fmt.Errorf("empty rsi")
	}

	if !rsiSp.IsSatisfiedBy(rsi[0]) {
		return domain.Item{}, nil
	}

	macd, macdErr := s.macd.CalculateMACD(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -30, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 8, 26, 9)

	if macdErr != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate macd :%w", macdErr).Error())

		return domain.Item{}, macdErr
	}

	if !macDSp.IsSatisfiedBy(macd[len(macd)-2:]) {
		return domain.Item{}, nil
	}

	interval = int32(3)
	price, errPrice := s.marketDataServiceGrpcClient.GetCandles(ctx, &share.ID, int32(in.Interval), utils.TimeStampPbGenerator(dateNow, -10, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), &interval, true)

	if errPrice != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate price:%w", errPrice).Error())

		return domain.Item{}, errPrice
	}

	volume, errV := s.atrInstrument.AverageVolume(ctx, share.ID, in.Interval, dateNow)

	if errV != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate valume :%w", errV).Error())

		return domain.Item{}, errV
	}

	if !volumeSp.IsSatisfiedBy(price[len(price)-1], volume) {
		return domain.Item{}, nil
	}

	atrTechItem, atrErr := s.atrInstrument.TechAnalyse(ctx, &share.ID, enum.Hour1, time.Now())

	if atrErr != nil {
		logger.ErrorContext(ctx, "Failed to get ATR", share.Name)
	}

	return domain.Item{
		MACD:           macd[len(macd)-2:],
		RSI:            rsi[:2],
		ATR:            atrTechItem,
		InstrumentName: share.Name,
	}, nil
}
