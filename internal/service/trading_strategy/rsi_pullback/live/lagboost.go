package live

import (
	"context"
	"fmt"

	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/lagboost"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/marketdata"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/logger"
)

// imoexTicker — индекс, с которым сравнивается тикер.
const imoexTicker = "IMOEX"

// lagBoostStrategies — стратегии, на которых буст измерен. gap_fade не измерялся.
// "rsi_zone" — значение rsizonelive.Name; импортировать пакет гостя сюда нельзя (он сам
// зависит от livecore), поэтому строка, а совпадение держит тест.
var lagBoostStrategies = map[string]bool{adapter.LegacyOwner: true, "rsi_zone": true}

// lagBoost решает множитель доли для входа. Любой сбой — множитель 1 и WARN: буст
// необязателен и вход не блокирует.
func (s *service) lagBoost(ctx context.Context, pc *passCtx, sl *slot, ticker string,
	md strategy.MarketData) lagboost.Decision {

	if !s.lag.Enabled || !lagBoostStrategies[sl.strat.Name()] {
		return lagboost.Decision{Mult: 1}
	}
	if !pc.indexLoaded {
		pc.indexLoaded = true
		pc.index, pc.indexErr = s.loadIndex(ctx, pc)
		if pc.indexErr != nil {
			logger.Warn(fmt.Sprintf("rsi_pullback: LAG_BOOST — индекс недоступен, вход без буста: %v", pc.indexErr))
		}
	}
	if pc.indexErr != nil {
		return lagboost.Decision{Mult: 1}
	}
	d := lagboost.Decide(lagboost.WeekdayCloses(md.DailyCloses, md.DailyTimes), pc.index, s.lag)
	if !d.Applied {
		logger.InfoContext(ctx, fmt.Sprintf("%s: %s LAG_BOOST не применён: %s", sl.strat.Name(), ticker, d.Reason))
	}
	return d
}

func (s *service) loadIndex(ctx context.Context, pc *passCtx) ([]float64, error) {
	if s.imoexUID == "" {
		uid, err := s.instruments.IndicativeUIDByTicker(ctx, imoexTicker)
		if err != nil {
			return nil, err
		}
		s.imoexUID = uid
	}
	return marketdata.IndexDailyCloses(ctx, s.market, s.imoexUID, s.lag.Days, pc.now)
}

// withNote дописывает строку буста к уведомлению о входе (пустая пометка — без изменений).
func withNote(msg, note string) string {
	if note == "" {
		return msg
	}
	return msg + "\n  " + note
}
