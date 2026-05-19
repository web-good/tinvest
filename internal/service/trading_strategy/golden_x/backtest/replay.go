package backtest

import (
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/utils"
)

// DetectFunc is the signature of golden_x.Detect; injected so the replay
// engine can be unit-tested with a fake detector.
type DetectFunc func(closed []*model.CandleItemTechAnalyse, rsiPeriod int, kind dto.StrategyKind, useTrendFilter bool) (dto.Signal, error)

type ReplayConfig struct {
	Kind           dto.StrategyKind
	RSIPeriod      int // per-share; pulled from collection.Instrument by caller
	StartIdx       int // first index at which we will evaluate a signal
	MaxWeeks       int // timeout cap (52 per spec §4.2)
	UseTrendFilter bool
}

// Replay iterates the closed-weekly-candle slice and emits Trade rows per
// §4.2 exit ordering: stop → sell-tier → timeout, with end-of-history
// mark-to-market for any remaining open position. Detect is called on the
// slice `candles[:t+1]` for each t; the caller is responsible for ensuring
// `candles` is already trimmed to closed weeks.
func Replay(shareID string, candles []*model.CandleItemTechAnalyse, detect DetectFunc, cfg ReplayConfig) []Trade {
	var (
		trades []Trade
		pos    *Position
	)
	for t := cfg.StartIdx; t < len(candles); t++ {
		closed := candles[:t+1]
		weekT := candles[t]
		sig, err := detect(closed, cfg.RSIPeriod, cfg.Kind, cfg.UseTrendFilter)
		if err != nil {
			continue
		}
		weekClose := utils.CombinePrice(weekT.Close.Units, weekT.Close.Nano)
		weekLow := utils.CombinePrice(weekT.Low.Units, weekT.Low.Nano)

		// 1. Exit-ordering for any open position.
		if pos != nil {
			if pos.StopPrice() > 0 && weekLow <= pos.StopPrice() {
				// 1a. Stop hit on this week's Low.
				trades = append(trades, pos.CloseAll(weekT.Time, pos.StopPrice(), ExitReasonStop))
				pos = nil
			} else if partials := pos.EvaluateSellExits(weekT.Time, weekClose, sig.SellThresholds, sig.RSI); len(partials) > 0 {
				// 1b. Sell-tier exits at week close.
				trades = append(trades, partials...)
				if pos.FullyClosed() {
					pos = nil
				}
			} else if pos.WeeksHeldAt(weekT.Time) >= cfg.MaxWeeks {
				// 1c. Timeout.
				trades = append(trades, pos.CloseAll(weekT.Time, weekClose, ExitReasonTimeout))
				pos = nil
			}
		}

		// 2. Entry: only on green tier, only when flat.
		if pos == nil && sig.GreenBuy {
			pos = OpenPosition(shareID, weekT.Time, weekClose, sig.Stop, cfg.Kind)
		}
	}
	// End-of-history mark-to-market.
	if pos != nil && len(candles) > 0 {
		last := candles[len(candles)-1]
		lastClose := utils.CombinePrice(last.Close.Units, last.Close.Nano)
		trades = append(trades, pos.CloseAll(last.Time, lastClose, ExitReasonOpen))
	}
	return trades
}
