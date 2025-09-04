package macd_rsi

import (
	"context"
	"fmt"
	"time"
	notification2 "tinvest/internal/domain/notification"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/macd_rsi/dto"
	"tinvest/internal/service/trading_strategy/macd_rsi/specification"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

// const TakeProfitAccount string = "2252263587"
const TakeProfitAccount string = "2252263587"

var (
	notificationTakeProfit []notification2.TakeProfit
)

func (s *service) TakeProfit(ctx context.Context, in dto.TakeProfit) error {
	res, err := s.usersServiceClient.GetAccounts(ctx)

	if err != nil {
		logger.ErrorContext(ctx, "error account", err)

		return err
	}

	for _, acc := range res {
		if acc.ID != TakeProfitAccount {
			continue
		}

		portfolio, _ := s.operationsServiceClient.GetPortfolio(ctx, acc.ID)
		atrSp := specification.ProfitEqualsAtr{}

		for _, position := range portfolio {
			atr, err := s.atrInstrument.TechAnalyse(ctx, &position.ShareID, enum.Hour1, time.Now())
			operations, er := s.operationsServiceClient.GetOperation(ctx, acc.ID, position.Figi)

			if utils.CombinePrice(position.PurchasePrice.Units, position.PurchasePrice.Nano) >
				utils.CombinePrice(position.Price.Units, position.Price.Nano) {
				//TODO сделать отправку уведомления о том что надо докупить акцию в случае просадки больше Н процентов
				continue
			}

			fmt.Println(operations, er)
			if err != nil {
				logger.ErrorContext(ctx, "atr get error", err)

				continue
			}

			if atrSp.IsSatisfiedBy(atr, position) {
				share, err2 := s.instrumentServiceGrpcClient.ShareByID(ctx, position.ShareID)

				if err2 != nil {
					logger.ErrorContext(ctx, "instrument get share by id error", err2)

					return err2
				}

				notificationTakeProfit = append(notificationTakeProfit, notification2.TakeProfit{Share: *share})

				return nil
			}

		}
	}

	if len(notificationTakeProfit) != 0 {
		//err := s.tgClient.SendMessage(notification.TakeProfit(notificationTakeProfit))

		//if err != nil {
		//	logger.ErrorContext(ctx, "message is not sent", err)
		//}

		notificationTakeProfit = []notification2.TakeProfit{}
	}

	return nil
}
