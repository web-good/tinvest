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

	sortedResult := topByYTM(collectionBond.GetAll(), 10)

	err := tgClient.SendMessage(notification.Send(sortedResult, dateFrom, dateTo))
	if err != nil {
		logger.ErrorContext(ctx, "message is not sent", slog.String("error_msg", err.Error()))
	}
}

// topByYTM возвращает топ-n отчётов по доходности к погашению (YTM) по убыванию.
func topByYTM(reports []domain.BondReport, n int) []domain.BondReport {
	c := collection.New[domain.BondReport]()
	for _, r := range reports {
		c.Add(r)
	}
	return c.GetTopByCriteria(func(i, j domain.BondReport) bool {
		return i.PercentByYear > j.PercentByYear
	}, n)
}
