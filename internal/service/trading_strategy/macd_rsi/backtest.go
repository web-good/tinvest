package macd_rsi

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io/ioutil"
	"time"
	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/model"
	"tinvest/internal/service/account"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

type Logs struct {
	BuyMacD  []*model.MacDItemTechAnalyse
	BuyRsi   []*model.RsiItemTechAnalyse
	SellMacD []*model.MacDItemTechAnalyse
	SellRsi  []*model.RsiItemTechAnalyse
	Volume   float64
}

func (s *service) BackTest(ctx context.Context, in dto.BackTest) error {
	fmt.Println(in.DateFrom)
	//t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	//for _, share := range t {
	//	fmt.Println("Share: ", share)
	//}
	loc, _ := time.LoadLocation("Europe/Moscow")
	times := generateTimes(in.DateFrom, in.DateTo, in.Interval)
	dateNow := in.DateFrom
	rsiSp := specification.RsiTrade{}
	macDSp := specification.Macd{}
	macRed := specification.MacdCrossRed{}
	emaSp := specification.EmaSpecification{}
	rsiProfit := specification.RsiProfit{}
	account := account.NewAccount(200000)
	volumeSp := specification.Volume{}
	logs := make(map[string]map[time.Time]Logs)
	_, _ = s.macd.CalculateMACD(ctx, "87db07bc-0e02-4e29-90bb-05e8ef791d7b", in.Interval, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 8, 26, 9)

	//var wg sync.WaitGroup
	for _, id := range in.InstrumentID {
		//wg.Add(1)
		//go func(id string) {
		//	defer wg.Done()
		logs[id] = make(map[time.Time]Logs)
		for _, t := range times {
			dateNow = t
			fmt.Println(dateNow, "|||", t)
			time.Sleep(800 * time.Millisecond)

			if account.ExistInstrument(id) {
				interval := int32(1)
				priceNow, err := s.marketDataServiceGrpcClient.GetCandles(ctx, &id, int32(in.Interval), timestamppb.New(dateNow), timestamppb.New(dateNow), &interval, true)
				if err != nil || len(priceNow) == 0 {
					logger.ErrorContext(ctx, fmt.Errorf("error in GET PRICE :%w", err).Error())

					continue
				}
				order := account.GetOrderLine(id)
				price := utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano)

				if order.StopLoss >= price || order.TakeProfit <= price {
					account.Sell(id, price, dateNow)
				}

				macd, errMacd := s.marketDataServiceGrpcClient.GetTechAnalyseMacDCustom(ctx, id, int(in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 8, 26, 9)
				if errMacd != nil || len(macd) == 0 {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate macd :%w", errMacd).Error())

					continue
				}
				rsi, errR := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, id, int(in.Interval), utils.TimeStampPbGenerator(dateNow, -3, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 12)

				if errR != nil || len(rsi) == 0 {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errR).Error())

					continue
				}
				logs[id][t] = Logs{
					SellMacD: macd,
					BuyRsi:   rsi,
				}
				if macRed.IsSatisfiedBy(macd[0]) || rsiProfit.IsSatisfiedBy(rsi, order.PurchaseTime, t, in.Interval) {
					account.Sell(id, price, t)

					continue
				}

			}

			ema100, err100 := s.ema.TechAnalyse(ctx, &id, int32(in.Interval), utils.TimeGenerator(dateNow, -180, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 100)

			if err100 != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema100 :%w", err100).Error())

				continue
			}

			rsi, errR := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, id, int(in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), timestamppb.New(dateNow), 12)
			fmt.Println(dateNow, rsi, utils.TimeStampPbGenerator(dateNow, -1, in.Interval).AsTime().In(loc))
			if errR != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errR).Error())

				continue
			}

			macd, macdErr := s.marketDataServiceGrpcClient.GetTechAnalyseMacDCustom(ctx, id, int(in.Interval), timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 8, 26, 9)

			macd, macdErr = s.marketDataServiceGrpcClient.GetTechAnalyseMacDCustom(ctx, id, int(in.Interval), utils.TimeStampPbGenerator(dateNow, -3, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 8, 26, 9)

			m, err := s.macd.CalculateMACD(ctx, id, in.Interval, utils.TimeStampPbGenerator(dateNow, -3, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 8, 26, 9)
			fmt.Println(m, err)

			if macdErr != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate macd :%w", errR).Error())

				continue
			}

			interval := int32(1)

			price, err := s.marketDataServiceGrpcClient.GetCandles(ctx, &id, int32(in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), &interval, true)

			if err != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate :%w", err).Error())

				continue
			}

			volume, errV := s.atrInstrument.AverageVolume(ctx, id, in.AtrInterval)

			if errV != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate volume :%w", err).Error())
			}

			priceNow, errPrice := s.marketDataServiceGrpcClient.GetCandles(ctx, &id, int32(in.Interval), timestamppb.New(dateNow), timestamppb.New(dateNow), &interval, true)
			if errPrice != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate priceNow :%w", err).Error())

				continue
			}

			if len(rsi) == 0 || len(macd) == 0 || len(price) == 0 || len(priceNow) == 0 || len(ema100) == 0 {
				fmt.Println("Не стали раасчитывать попадание, тк не удалось собрать всех данных")

				continue
			}

			atr, _ := s.atrInstrument.TechAnalyse(ctx, &id, in.AtrInterval)
			fmt.Println(rsiSp.IsSatisfiedBy(rsi[0]), macDSp.IsSatisfiedBy(macd), volumeSp.IsSatisfiedBy(price[0], volume), rsi[0])
			logs[id][t] = Logs{
				BuyMacD: macd,
				BuyRsi:  rsi,
				Volume:  volume,
			}
			if rsiSp.IsSatisfiedBy(rsi[0]) && macDSp.IsSatisfiedBy(macd) && volumeSp.IsSatisfiedBy(price[0], volume) && emaSp.IsSatisfiedBy(ema100[len(ema100)-1], *price[0]) {
				fmt.Println("Хотим купить", id)
				if !account.ExistInstrument(id) {
					fmt.Println("Купили", id)
					err := account.Buy(id, &backtest.OrderLine{
						InstrumentId:  id,
						PurchasePrice: utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano),
						TakeProfit:    utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano) + atr.Value*1.5,
						StopLoss:      utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano) - (atr.Value / 2),
						PurchaseTime:  t,
					})
					if err != nil {
						logger.ErrorContext(ctx, fmt.Errorf("error in buy :%w", err).Error())
					}
				}
			}
		}
		//}(id)
	}

	//wg.Wait()
	js, err := json.MarshalIndent(logs, "", " ")
	if err != nil {
	}

	err = ioutil.WriteFile("dump.json", js, 0644)
	if err != nil {
		return err
	}
	fmt.Println(account)
	js, err = json.MarshalIndent(account, "", " ")
	if err != nil {
	}

	err = ioutil.WriteFile("account.json", js, 0644)
	if err != nil {
		return err
	}
	return nil
}

func generateTimes(dateFrom, dateTo time.Time, interval enum.Interval) []time.Time {
	var times []time.Time
	current := dateFrom

	for !current.After(dateTo) {
		hour := current.Hour()
		// Исключаем промежутки с 24:00 (ровно 0:00 следующего дня) и до 6:00
		if !(hour >= 0 && hour < 6) {
			times = append(times, current)
		}

		current = utils.TimeGenerator(current, 1, interval)
	}

	return times
}
