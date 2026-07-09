package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"tinvest/internal/domain"
	"tinvest/internal/service/trading_strategy/bonds/notification"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/collection"
	"tinvest/pkg/logger"
)

func Sender(ctx context.Context, bondCh <-chan domain.BondReport, tgClient telegram.Client, wg *sync.WaitGroup, dateFrom, dateTo time.Time) {
	defer wg.Done()
	collectionBond := collection.New[domain.BondReport]()

	for bond := range bondCh {
		collectionBond.Add(bond)
	}

	if len(collectionBond.GetAll()) == 0 {
		return
	}

	sortByCouponYield := int(time.Until(dateTo).Hours()/8760) >= 6

	sortedResult := collectionBond.GetTopByCriteria(func(i, j domain.BondReport) bool {
		if sortByCouponYield {
			return i.CouponPercentByYear > j.CouponPercentByYear
		}
		return i.PercentByYear > j.PercentByYear
	}, 10)

	err := tgClient.SendMessage(notification.Send(sortedResult, dateFrom, dateTo))
	if err != nil {
		logger.ErrorContext(ctx, "message is not sent", slog.String("error_msg", err.Error()))
	}
}
