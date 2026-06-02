package scalping

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/notification"
	"tinvest/internal/service/trading_strategy/scalping/universe"
	"tinvest/internal/utils"
	"tinvest/pkg/logger"
)

func (s *service) Trade(ctx context.Context, in dto.Trade) error {
	loc, _ := time.LoadLocation("Europe/Moscow")
	dateNow := time.Now().In(loc)
	interval := s.settings.Interval

	shares, err := s.instrumentsClient.Shares(ctx)
	if err != nil {
		return fmt.Errorf("scalping: load shares: %w", err)
	}

	// 1. Rank the universe by ATR% (ATR / last price).
	scored := make([]universe.Scored, 0, len(shares))
	for _, sh := range shares {
		if sh.Currency != "rub" || !sh.Trading || sh.LastPriceRub == 0 {
			continue
		}
		time.Sleep(300 * time.Millisecond)
		atrItem, atrErr := s.atr.TechAnalyse(ctx, &sh.ID, interval, dateNow)
		if atrErr != nil {
			logger.ErrorContext(ctx, fmt.Errorf("scalping: atr %s: %w", sh.Ticker, atrErr).Error())
			continue
		}
		scored = append(scored, universe.Scored{
			InstrumentID:   sh.ID,
			InstrumentName: sh.Name,
			Ticker:         sh.Ticker,
			ATRPercent:     atrItem.Value / sh.LastPriceRub * 100,
		})
	}
	top := universe.TopN(scored, s.settings.UniverseSize)

	// 2. Read open positions from the dedicated account.
	positions, err := s.operationsClient.GetPortfolio(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("scalping: load portfolio: %w", err)
	}
	posByID := make(map[string]float64, len(positions))
	for _, p := range positions {
		if p.InstrumentType == "share" && p.Quantity > 0 {
			posByID[p.ShareID] = utils.CombinePrice(p.PurchasePrice.Units, p.PurchasePrice.Nano)
		}
	}
	openCount := len(posByID)

	// 3. Evaluate each candidate and collect signals.
	signals := make([]model.Signal, 0, len(top))
	for _, item := range top {
		id := item.InstrumentID
		time.Sleep(300 * time.Millisecond)

		emaItems, emaErr := s.ema.TechAnalyse(ctx, &id, interval.ToNumberInvestApi(),
			utils.TimeGenerator(dateNow, -int64(s.settings.EmaPeriod)-50, interval), dateNow, s.settings.EmaPeriod)
		if emaErr != nil || len(emaItems) == 0 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: ema %s skipped", item.Ticker))
			continue
		}
		last := emaItems[len(emaItems)-1].SignalLine
		emaVal := utils.CombinePrice(last.Units, last.Nano)

		rsiItems, rsiErr := s.rsi.CalculateRSI(ctx, id, interval,
			utils.TimeStampPbGenerator(dateNow, -int64(s.settings.RsiPeriod)*3, interval), timestamppb.New(dateNow), s.settings.RsiPeriod)
		if rsiErr != nil || len(rsiItems) < 2 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: rsi %s skipped", item.Ticker))
			continue
		}
		rsiNow := utils.CombinePrice(rsiItems[len(rsiItems)-1].SignalLine.Units, rsiItems[len(rsiItems)-1].SignalLine.Nano)
		rsiPrev := utils.CombinePrice(rsiItems[len(rsiItems)-2].SignalLine.Units, rsiItems[len(rsiItems)-2].SignalLine.Nano)

		limit := int32(2)
		candles, candErr := s.marketDataClient.GetCandles(ctx, &id, interval.ToNumberInvestApi(),
			utils.TimeStampPbGenerator(dateNow, -2, interval), timestamppb.New(dateNow), &limit, true)
		if candErr != nil || len(candles) == 0 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: candles %s skipped", item.Ticker))
			continue
		}
		closeQ := candles[len(candles)-1].Close
		price := utils.CombinePrice(closeQ.Units, closeQ.Nano)

		atrItem, atrErr := s.atr.TechAnalyse(ctx, &id, interval, dateNow)
		if atrErr != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: atr %s skipped", item.Ticker))
			continue
		}

		purchase, hasPos := posByID[id]

		sig := Decide(Candidate{
			InstrumentID:   id,
			InstrumentName: item.InstrumentName,
			Ticker:         item.Ticker,
			Price:          price,
			ATR:            atrItem.Value,
			AboveEMA:       price > emaVal,
			RSIPrev:        rsiPrev,
			RSINow:         rsiNow,
			HasPosition:    hasPos,
			PurchasePrice:  purchase,
		}, s.settings, openCount)

		if sig.Kind != model.SignalNone {
			signals = append(signals, sig)
		}
	}

	if len(signals) == 0 {
		logger.InfoContext(ctx, "scalping: no signals this run")
		return nil
	}

	if err := s.tgClient.SendMessage(notification.Trade(signals)); err != nil {
		return fmt.Errorf("scalping: send message: %w", err)
	}
	return nil
}
