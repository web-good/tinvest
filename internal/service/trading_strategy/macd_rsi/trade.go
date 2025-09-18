package macd_rsi

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sync"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/notification"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

//Trade Условия покупки по MACD
/*	1) пересечение MACD ниже отметки 0
	2) цена находится над EMA 200
	3) объёмы выше среднего
	4) RSI ниже 70
*/
//Trade Условия покупки по RSI
/*	1) цена выше ema50
	2) объёмы выше среднего
	3) RSI пересек границу 50
*/
func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	info := domain.NewInfo()
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)
	semaphore := semaphore.New(3)
	var wg sync.WaitGroup

	for _, share := range t {
		wg.Add(1)
		go func(share *model.Share) {
			defer func() {
				wg.Done()
				semaphore.Release()
			}()
			semaphore.Acquire()
			resultRSI, _ := s.processRsi(ctx, in, share)

			if resultRSI.RSI != nil {
				info.WriteToMap(share.ID, resultRSI)

				return
			}

			time.Sleep(1 * time.Second)
			resultMACD, err := s.do(ctx, in, share)

			if err != nil || resultMACD.MACD == nil {
				return
			}

			info.WriteToMap(share.ID, resultMACD)
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
	volatilitySp := specification.Volatility{}
	v, errV := s.volatility.CalculateVolatility(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -50, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 14)

	if v == nil {
		return domain.Item{}, nil
	}

	if errV != nil {
		return domain.Item{}, errV
	}

	if !volatilitySp.IsSatisfiedBy(v) {
		return domain.Item{}, nil
	}

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

	if !emaSp.IsSatisfiedBy(ema[len(ema)-1], *priceNow[0]) {
		return domain.Item{}, nil
	}

	rsi, errRsi := s.rsi.CalculateRSI(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -35, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 12)

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

	atrTechItem, atrErr := s.atrInstrument.TechAnalyse(ctx, &share.ID, in.AtrInterval, time.Now())

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

func (s *service) processRsi(ctx context.Context, in dto.Trade, share *model.Share) (domain.Item, error) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	volatilitySp := specification.Volatility{}
	emaSp := specification.EmaSpecification{}
	rsiTremdSp := specification.RsiTend{}
	interval := int32(3)
	v, errV := s.volatility.CalculateVolatility(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -50, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 14)

	if v == nil {
		return domain.Item{}, nil
	}

	if errV != nil {
		return domain.Item{}, errV
	}

	if !volatilitySp.IsSatisfiedBy(v) {
		return domain.Item{}, nil
	}

	ema, errEma := s.ema.TechAnalyse(ctx, &share.ID, int32(in.Interval), utils.TimeGenerator(dateNow, -200, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 50)

	if errEma != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema50 :%w", errEma).Error())

		return domain.Item{}, errEma
	}

	ema150, errEma150 := s.ema.TechAnalyse(ctx, &share.ID, int32(in.Interval), utils.TimeGenerator(dateNow, -300, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 150)

	if errEma150 != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema150 :%w", errEma).Error())

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

	priceEma50 := utils.CombinePrice(ema[len(ema)-1].SignalLine.Units, ema[len(ema)-1].SignalLine.Nano)
	priceEma150 := utils.CombinePrice(ema150[len(ema150)-1].SignalLine.Units, ema150[len(ema150)-1].SignalLine.Nano)

	if priceEma50 < priceEma150 {
		return domain.Item{}, nil
	}

	if !emaSp.IsSatisfiedBy(ema[len(ema)-1], *priceNow[0]) {
		return domain.Item{}, nil
	}

	price, errPrice := s.marketDataServiceGrpcClient.GetCandles(ctx, &share.ID, int32(in.Interval), utils.TimeStampPbGenerator(dateNow, -20, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), &interval, true)

	if errPrice != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate priceNow :%w", errPriceN).Error())

		return domain.Item{}, errPrice
	}

	if len(price) == 0 {
		logger.ErrorContext(ctx, fmt.Errorf("empty price").Error())

		return domain.Item{}, errPrice
	}

	rsi, errRsi := s.rsi.CalculateRSI(ctx, share.ID, in.Interval, utils.TimeStampPbGenerator(dateNow, -50, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 12)

	if errRsi != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errRsi).Error())

		return domain.Item{}, errRsi
	}

	if len(rsi) == 0 {
		logger.ErrorContext(ctx, fmt.Errorf("empty rsi").Error())

		return domain.Item{}, fmt.Errorf("empty rsi")
	}

	if !rsiTremdSp.IsSatisfiedBy(rsi) {
		return domain.Item{}, nil
	}

	volume, errV := s.atrInstrument.AverageVolume(ctx, share.ID, in.Interval, dateNow)

	if errV != nil {
		logger.ErrorContext(ctx, fmt.Errorf("error in calculate valume :%w", errV).Error())

		return domain.Item{}, errV
	}

	volumeSp := specification.Volume{}

	if !volumeSp.IsSatisfiedBy(price[len(price)-1], volume) {
		return domain.Item{}, nil
	}

	atrTechItem, atrErr := s.atrInstrument.TechAnalyse(ctx, &share.ID, in.AtrInterval, time.Now())

	if atrErr != nil {
		logger.ErrorContext(ctx, "Failed to get ATR", share.Name)
	}

	return domain.Item{
		RSI:            rsi[:2],
		ATR:            atrTechItem,
		InstrumentName: share.Name,
	}, nil
}
