package macd_rsi

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io/ioutil"
	"sync"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/service/account"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

type Logs struct {
	BuyMacD  []*domain.MACDItemTechAnalyse
	BuyRsi   []*domain.RSIItemTechAnalyse
	SellMacD []*domain.MACDItemTechAnalyse
	SellRsi  []*domain.RSIItemTechAnalyse
	Volume   float64
}

func (s *service) BackTest(ctx context.Context, in dto.BackTest) error {
	fmt.Println(in.DateFrom)
	t, _ := s.instrumentServiceGrpcClient.Shares(ctx)

	for _, share := range t {
		fmt.Println("Share: ", share)
	}
	//loc, _ := time.LoadLocation("Europe/Moscow")
	times := generateTimes(in.DateFrom, in.DateTo, in.Interval)
	dateNow := in.DateFrom
	rsiSp := specification.RsiTrade{}
	macDSp := specification.Macd{}
	//macRed := specification.MacdCrossRed{}
	emaSp := specification.EmaSpecification{}
	rsiProfit := specification.RsiProfit{}
	account := account.NewAccount(200000)
	volumeSp := specification.Volume{}
	logs := make(map[string]map[time.Time]Logs)
	//y, _ := s.rsi.CalculateRSI(ctx, "87db07bc-0e02-4e29-90bb-05e8ef791d7b", in.Interval, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 12)
	//y, _ := s.volatility.CalculateVolatility(ctx, "87db07bc-0e02-4e29-90bb-05e8ef791d7b", in.Interval, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 14)
	//fmt.Println(y)
	//m, _ := s.macd.CalculateMACD(ctx, "b993e814-9986-4434-ae88-b086066714a0", in.Interval, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 8, 26, 9)
	//fmt.Println(m, time.Now().AddDate(0, 0, -1), "||||||||||", time.Now())
	var wg sync.WaitGroup
	for _, id := range in.InstrumentID {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			logs[id] = make(map[time.Time]Logs)
			for _, t := range times {
				dateNow = t
				fmt.Println(dateNow, "|||", t)
				time.Sleep(1000 * time.Millisecond)
				//	rsi5, _ := s.rsi.CalculateRSI(ctx, id, in.Interval, utils.TimeStampPbGenerator(time.Date(2025, 7, 1, 11, 54, 0, 0, time.Local), -25, in.Interval), utils.TimeStampPbGenerator(time.Date(2025, 7, 1, 11, 54, 0, 0, time.Local), -1, in.Interval), 12)
				//fmt.Println(rsi5)
				if account.ExistInstrument(id) {
					time.Sleep(500 * time.Millisecond)
					interval := int32(1)
					priceNow, err := s.marketDataServiceGrpcClient.GetCandles(ctx, &id, int32(in.Interval), timestamppb.New(dateNow), timestamppb.New(dateNow), &interval, true)
					if err != nil || len(priceNow) == 0 {
						logger.ErrorContext(ctx, fmt.Errorf("error in GET PRICE :%w", err).Error())

						continue
					}
					order := account.GetOrderLine(id)
					price := utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano)

					if order.StopLoss >= price {
						account.Sell(id, price, dateNow)

						continue
					}

					rsi, errR := s.rsi.CalculateRSI(ctx, id, in.Interval, utils.TimeStampPbGenerator(dateNow, -25, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 12)

					if order.TakeProfit <= price && (rsi[0].SignalLine.Units < 70 || rsi[0].SignalLine.Units > 85) {
						account.Sell(id, price, dateNow)

						continue
					}

					macd, errMacd := s.macd.CalculateMACD(ctx, id, in.Interval, utils.TimeStampPbGenerator(dateNow, -30, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 8, 26, 9)

					if errMacd != nil || len(macd) == 0 {
						logger.ErrorContext(ctx, fmt.Errorf("error in calculate macd :%w", errMacd).Error())

						continue
					}

					if errR != nil || len(rsi) == 0 {
						logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errR).Error())

						continue
					}
					logs[id][t] = Logs{
						SellMacD: macd,
						SellRsi:  rsi,
					}
					if
					//macRed.IsSatisfiedBy(macd[len(macd)-1]) ||
					rsiProfit.IsSatisfiedBy(rsi[:3], order.PurchaseTime, t, in.Interval) {
						account.Sell(id, price, t)

						continue
					}
				}

				ema100, err100 := s.ema.TechAnalyse(ctx, &id, int32(in.Interval), utils.TimeGenerator(dateNow, -350, in.Interval), utils.TimeGenerator(dateNow, -1, in.Interval), 200)

				if err100 != nil {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate ema100 :%w", err100).Error())

					continue
				}

				rsi, errR := s.rsi.CalculateRSI(ctx, id, in.Interval, utils.TimeStampPbGenerator(dateNow, -25, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 12)

				if errR != nil {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errR).Error())

					continue
				}

				macd, macdErr := s.macd.CalculateMACD(ctx, id, in.Interval, utils.TimeStampPbGenerator(dateNow, -30, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), 8, 26, 9)

				if macdErr != nil {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate macd :%w", errR).Error())

					continue
				}

				interval := int32(3)

				price, err := s.marketDataServiceGrpcClient.GetCandles(ctx, &id, int32(in.Interval), utils.TimeStampPbGenerator(dateNow, -10, in.Interval), utils.TimeStampPbGenerator(dateNow, -1, in.Interval), &interval, true)

				if err != nil {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate :%w", err).Error())

					continue
				}

				priceNow, errPrice := s.marketDataServiceGrpcClient.GetCandles(ctx, &id, int32(in.Interval), timestamppb.New(dateNow), timestamppb.New(dateNow), &interval, true)
				if errPrice != nil {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate priceNow :%w", err).Error())

					continue
				}

				volume, err := s.atrInstrument.AverageVolume(ctx, id, in.Interval, dateNow)

				if err != nil {
					logger.ErrorContext(ctx, fmt.Errorf("error in calculate valume :%w", err).Error())
				}

				if len(rsi) == 0 || len(macd) == 0 || len(price) == 0 || len(priceNow) == 0 || len(ema100) == 0 {
					fmt.Println("Не стали раасчитывать попадание, тк не удалось собрать всех данных")

					continue
				}

				atr, _ := s.atrInstrument.TechAnalyse(ctx, &id, in.AtrInterval, dateNow)
				logs[id][t] = Logs{
					BuyMacD: macd,
					BuyRsi:  rsi,
					Volume:  volume,
				}

				if rsiSp.IsSatisfiedBy(rsi[0]) && macDSp.IsSatisfiedBy(macd[len(macd)-2:]) && emaSp.IsSatisfiedBy(ema100[len(ema100)-2], *priceNow[0]) && volumeSp.IsSatisfiedBy(price[len(price)-1], volume) {
					fmt.Println("Хотим купить", id)
					if !account.ExistInstrument(id) {
						fmt.Println("Купили", id)
						err := account.Buy(id, &backtest.OrderLine{
							InstrumentId:  id,
							PurchasePrice: utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano),
							TakeProfit:    utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano) + atr.Value*0.8,
							StopLoss:      utils.CombinePrice(priceNow[0].Open.Units, priceNow[0].Open.Nano) - (atr.Value * 0.4),
							PurchaseTime:  t,
						})
						if err != nil {
							logger.ErrorContext(ctx, fmt.Errorf("error in buy :%w", err).Error())
						}
					}
				}
				time.Sleep(1500 * time.Millisecond)
			}
		}(id)
	}

	wg.Wait()
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
