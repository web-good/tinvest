package macd_rsi

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
	notification2 "tinvest/internal/domain/notification"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// const TakeProfitAccount string = "2252263587"
const TakeProfitAccount string = "2252263587"

var (
	notificationTakeProfit []notification2.TakeProfit
	notificationBuy        []notification2.TakeProfit
)

// TakeProfit получаем профит если цена дошла до желаемого значения цены и rsi меньше 70, или если rsi вышел выше 70 и
// пересекает его свеху вниз а также схожие уровни 75 80 85
// а так же если акция процеса больше чем на N процентов и rsi находится в критической секции то докупить акцию
func (s *service) TakeProfit(ctx context.Context, in dto.TakeProfit) error {
	res, err := s.usersServiceClient.GetAccounts(ctx)
	dateNow := time.Now()
	if err != nil {
		logger.ErrorContext(ctx, "error account", err)

		return err
	}

	for _, acc := range res {
		if acc.ID != TakeProfitAccount {
			continue
		}

		portfolio, _ := s.operationsServiceClient.GetPortfolio(ctx, acc.ID)
		atrSp := specification.ProfitEqualsAtr{ATRProfit: 0.3}
		rsiProfit := specification.RsiProfit{}
		buySp := specification.BuyMore{Diff: 0.07}

		for _, position := range portfolio {
			atr, err := s.atrInstrument.TechAnalyse(ctx, &position.ShareID, in.ATRInterval, dateNow)
			if err != nil {
				logger.ErrorContext(ctx, "atr get error", err)

				continue
			}
			//operations, er := s.operationsServiceClient.GetOperation(ctx, acc.ID, position.Figi)
			purchasePrice := utils.CombinePrice(position.PurchasePrice.Units, position.PurchasePrice.Nano)
			price := utils.CombinePrice(position.Price.Units, position.Price.Nano)
			rsi, errRsi := s.rsi.CalculateRSI(ctx, position.ShareID, in.Interval, utils.TimeStampPbGenerator(dateNow, -50, in.Interval), timestamppb.New(dateNow), 14)

			if errRsi != nil {
				logger.ErrorContext(ctx, fmt.Errorf("error in calculate rsi :%w", errRsi).Error())

				continue
			}

			share, err2 := s.instrumentServiceGrpcClient.ShareByID(ctx, position.ShareID)

			if err2 != nil {
				logger.ErrorContext(ctx, "instrument get share by id error", err2)

				return err2
			}

			if purchasePrice > price && buySp.IsSatisfiedBy(purchasePrice, price) && rsi[1].SignalLine.Units < 28 {
				notificationBuy = append(notificationBuy, notification2.TakeProfit{Share: *share})

				continue
			}

			if atrSp.IsSatisfiedBy(atr, position, rsi[0]) || rsiProfit.IsSatisfiedByProfit(rsi, position) {
				notificationTakeProfit = append(notificationTakeProfit, notification2.TakeProfit{Share: *share})

				return nil
			}
		}
	}

	if len(notificationTakeProfit) != 0 {
		err := s.tgClient.SendMessage(notification.TakeProfit(notificationTakeProfit))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)
		}

		notificationTakeProfit = []notification2.TakeProfit{}
	}

	if len(notificationBuy) != 0 {
		err := s.tgClient.SendMessage(notification.TakeBuy(notificationBuy))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)
		}

		notificationBuy = []notification2.TakeProfit{}
	}
	return nil
}
