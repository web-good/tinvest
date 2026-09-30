package live

import (
	"context"
	"fmt"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/reconstruct"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// pullbackStrategy — rsi_pullback как адаптер раннера счёта. Первый слот раннера: его
// порядок и есть приоритет входа, когда две стратегии дают BUY по тикеру на одном баре.
type pullbackStrategy struct {
	cfg *config.RSIPullbackConfig
	tg  telegram.Client
}

var _ adapter.Strategy = (*pullbackStrategy)(nil)

func (p *pullbackStrategy) Name() string       { return adapter.LegacyOwner }
func (p *pullbackStrategy) Label() string      { return alertLabel }
func (p *pullbackStrategy) Tickers() []string  { return p.cfg.Tickers }
func (p *pullbackStrategy) BuyPct() float64    { return p.cfg.BuyPct }
func (p *pullbackStrategy) TradeEnabled() bool { return p.cfg.TradeEnabled }

func (p *pullbackStrategy) Decider(ticker string) (adapter.Decider, bool) {
	st, ok := StrategyFor(ticker)
	if !ok {
		return nil, false // не *core.Strategy(nil): типизированный nil в интерфейсе != nil
	}
	return st, true
}

func (p *pullbackStrategy) DesiredStop(ticker string, e statestore.Entry) (float64, string) {
	return core.DesiredStop(mustParams(ticker), e.EntryPrice, e.EntryATR, e.MaxFav)
}

func (p *pullbackStrategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	return reconstruct.Entry(ctx, in.Trades, in.Candles, in.AccountID, in.InstrumentID, in.Ticker,
		in.PurchasePrice, mustParams(in.Ticker), in.Now)
}

// Notify sends a Telegram message only when NotifyEnabled.
//
// Сбой доставки логируется уровнем ERROR, а не отбрасывается: отброшенная ошибка — это
// молчание о молчании. Бот, выкинутый из группы, отозванный токен или упёртый лимит
// оставляют раннер внешне работающим, а тему — пустой, и отличить это от «событий не
// было» становится нечем. Уровень ERROR выбран потому, что его подхватывает
// errorlog-sink и дублирует в тему General — то есть сообщение о недоставке уходит по
// каналу, который в этот момент ещё может быть жив.
func (p *pullbackStrategy) Notify(msg string) {
	if !p.cfg.NotifyEnabled {
		return
	}
	if err := p.tg.SendMessage(msg); err != nil {
		// Контекст пасса сюда намеренно не протянут: Notify зовут три десятка мест, а
		// хендлер логгера ctx всё равно не использует — сигнатура подорожала бы зря.
		logger.ErrorContext(context.Background(),
			fmt.Sprintf("rsi_pullback: уведомление не доставлено: %v", err))
	}
}

// mustParams: ParamsFor гарантированно ok — тикер прошёл StrategyFor выше.
func mustParams(ticker string) core.Params {
	p, _ := ParamsFor(ticker)
	return p
}
