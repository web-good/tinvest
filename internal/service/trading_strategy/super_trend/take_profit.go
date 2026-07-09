package super_trend

import (
	"context"
	"time"
	notification2 "tinvest/internal/domain/notification"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/super_trend/notification"
	"tinvest/internal/service/trading_strategy/super_trend/specification"
	"tinvest/pkg/logger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const TakeProfitAccount string = "2252263587"

var (
	notificationTakeProfit []notification2.TakeProfit
)

func (s *service) TakeProfit(ctx context.Context) error {
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
		rsiSp := specification.RsiProfit{}

		for _, position := range portfolio {
			atr, err := s.atr.TechAnalyse(ctx, &position.ShareID, enum.Hour1, time.Now())
			if err != nil {
				logger.ErrorContext(ctx, "atr get error", err)

				continue
			}

			rsiModel, err := s.marketDataServiceGrpcClient.GetTechAnalyseRsi(ctx, position.ShareID, 4, timestamppb.New(time.Now().AddDate(0, 0, -1)), timestamppb.New(time.Now()), 4)
			if err != nil {
				logger.ErrorContext(ctx, "rsi get error", err)

				continue
			}

			if atrSp.IsSatisfiedBy(atr, *position) || rsiSp.IsSatisfiedBy(rsiModel) {
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
		err := s.tgClient.SendMessage(notification.TakeProfit(notificationTakeProfit))

		if err != nil {
			logger.ErrorContext(ctx, "message is not sent", err)
		}

		notificationTakeProfit = []notification2.TakeProfit{}
	}

	return nil
}
