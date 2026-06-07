package scalping

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/notification"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
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
	byTicker := make(map[string]*imodel.Share, len(shares))
	for _, sh := range shares {
		byTicker[sh.Ticker] = sh
	}

	positions, err := s.operationsClient.GetPortfolio(ctx, s.accountID)
	if err != nil {
		return fmt.Errorf("scalping: load portfolio: %w", err)
	}
	posByID := make(map[string]strategy.Position, len(positions))
	for _, p := range positions {
		if p.InstrumentType == "share" && p.Quantity > 0 {
			// NOTE: only PurchasePrice/Quantity come from the broker. The levels
			// strategy's entry-locked fields (StopLoss/EntryATR/MaxFavorablePrice)
			// stay zero here, so its protective hard stop and trail arming are
			// DISABLED live. Per-position entry state must be persisted before
			// levels trades live — see docs/superpowers/specs/2026-06-08-levels-entry-locked-stops-design.md.
			posByID[p.ShareID] = strategy.Position{
				PurchasePrice: utils.CombinePrice(p.PurchasePrice.Units, p.PurchasePrice.Nano),
				Quantity:      p.Quantity,
			}
		}
	}

	signals := make([]model.Signal, 0, len(s.strategies))
	for _, st := range s.strategies {
		sh, ok := byTicker[st.Ticker()]
		if !ok || !sh.Trading {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: share %s not tradable, skipped", st.Ticker()))
			continue
		}
		id := sh.ID
		if in.SellOnly {
			if _, held := posByID[id]; !held {
				continue
			}
		}
		lookback := st.Lookback()
		limit := int32(lookback)

		time.Sleep(300 * time.Millisecond)
		candles, candErr := s.marketDataClient.GetCandles(ctx, &id, interval.ToNumberInvestApi(),
			utils.TimeStampPbGenerator(dateNow, -int64(lookback), interval), timestamppb.New(dateNow), &limit, true)
		if candErr != nil || len(candles) == 0 {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: candles %s skipped", st.Ticker()))
			continue
		}

		md := buildMarketData(candles)

		const dailyLookback = 250 // ~trading days of lead-in to warm the daily EMA(200)
		dailyLimit := int32(dailyLookback)
		time.Sleep(300 * time.Millisecond)
		dailyCandles, dailyErr := s.marketDataClient.GetCandles(ctx, &id, enum.Day1.ToNumberInvestApi(),
			utils.TimeStampPbGenerator(dateNow, -int64(dailyLookback), enum.Day1), timestamppb.New(dateNow), &dailyLimit, true)
		if dailyErr != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("scalping: daily candles %s skipped", st.Ticker()))
		} else {
			md.DailyCloses = completedDailyCloses(dailyCandles)
		}

		if pos, held := posByID[id]; held {
			md.Position = &pos
		}

		sig := st.Decide(md)
		if sig.Kind == model.SignalNone {
			continue
		}
		if in.SellOnly && sig.Kind != model.SignalSell {
			continue
		}
		sig.InstrumentID = sh.ID
		sig.InstrumentName = sh.Name
		signals = append(signals, sig)
	}

	if len(signals) == 0 {
		logger.InfoContext(ctx, "scalping: no signals this run")
		return nil
	}

	msg := notification.Trade(signals)
	if in.SellOnly {
		msg = notification.SellWatch(signals)
	}
	if err := s.tgClient.SendMessage(msg); err != nil {
		return fmt.Errorf("scalping: send message: %w", err)
	}
	return nil
}

// completedDailyCloses returns oldest-first closes of completed daily candles only,
// dropping the still-forming current day so the strategy never sees an unclosed bar.
func completedDailyCloses(candles []*imodel.CandleItemTechAnalyse) []float64 {
	out := make([]float64, 0, len(candles))
	for _, c := range candles {
		if !c.IsComplete {
			continue
		}
		out = append(out, utils.CombinePrice(c.Close.Units, c.Close.Nano))
	}
	return out
}

// buildMarketData converts an oldest-first candle series into a strategy snapshot.
func buildMarketData(candles []*imodel.CandleItemTechAnalyse) strategy.MarketData {
	md := strategy.MarketData{
		Highs:   make([]float64, len(candles)),
		Lows:    make([]float64, len(candles)),
		Closes:  make([]float64, len(candles)),
		Volumes: make([]int64, len(candles)),
	}
	for i, c := range candles {
		md.Highs[i] = utils.CombinePrice(c.High.Units, c.High.Nano)
		md.Lows[i] = utils.CombinePrice(c.Low.Units, c.Low.Nano)
		md.Closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
		md.Volumes[i] = c.Volume
	}
	if n := len(md.Closes); n > 0 {
		md.Price = md.Closes[n-1]
	}
	return md
}
