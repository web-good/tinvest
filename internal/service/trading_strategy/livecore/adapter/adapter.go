// Package adapter описывает, что живой раннер счёта ждёт от подключённой к нему стратегии.
// Раннер один на брокерский счёт (rsi_pullback/live); стратегии, торгующие на этом счёте,
// приходят к нему адаптерами — так rsi_zone/live не зависит от rsi_pullback/live.
package adapter

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// LegacyOwner — владелец записей стейта без поля strategy: прод-файл rsi_pullback записан
// до того, как на счёт пришла вторая стратегия.
const LegacyOwner = "rsi_pullback"

// Owner возвращает стратегию-владельца записи стейта.
func Owner(e statestore.Entry) string {
	if e.Strategy == "" {
		return LegacyOwner
	}
	return e.Strategy
}

// Decider — то, что раннеру нужно от ядра стратегии для одного тикера.
type Decider interface {
	Decide(md strategy.MarketData) model.Signal
	Lookback() int
}

// TradesClient — срез operations-клиента, нужный реконструкции входа.
type TradesClient interface {
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// ReconstructInput — всё, что раннер знает о позиции без локального стейта.
type ReconstructInput struct {
	Trades        TradesClient
	Candles       candles.CandleClient
	AccountID     string
	InstrumentID  string
	Ticker        string
	PurchasePrice float64 // средняя цена покупки из портфеля брокера
	Now           time.Time
}

// Strategy — стратегия, подключённая к раннеру счёта.
type Strategy interface {
	// Name пишется в стейт как владелец позиции: "rsi_pullback" | "rsi_zone".
	Name() string
	// Label — заголовок алертов: "RSI Pullback" | "RSI Zone".
	Label() string
	// Tickers — вселенная входов. Позиции вне её владелец ведёт до выхода.
	Tickers() []string
	// Decider — ядро с параметрами тикера; false — тикер не зарегистрирован.
	Decider(ticker string) (Decider, bool)
	// DesiredStop — защитный уровень открытой позиции; reason "" — стопа нет.
	DesiredStop(ticker string, e statestore.Entry) (level float64, reason string)
	// Reconstruct — вход позиции без локального стейта, по API брокера.
	Reconstruct(ctx context.Context, in ReconstructInput) (statestore.Entry, error)
	BuyPct() float64
	TradeEnabled() bool
	// Notify — своя тема Telegram и свой рубильник NotifyEnabled.
	Notify(msg string)
}

// EntryWindow — необязательная подсказка стратегии: может ли она войти на пассе в момент now.
// Ложь — пасс по свободному тикеру не собирает данные и не спрашивает стратегию: сборка
// MarketData — запросы свечей, а стратегия с узким окном входа (gap_fade — только утро) иначе
// тянула бы их на каждом пассе. Сопровождение открытых позиций окно не ограничивает.
type EntryWindow interface {
	EntryPossible(now time.Time) bool
}

// EntryFilter — необязательное вето стратегии на вход после BUY-сигнала ядра (например,
// день дивидендной отсечки). Непустая причина — вход пропускается с уведомлением; ошибка —
// вход пропускается с алертом (fail-closed). В обоих случаях бар не расходуется: пасс
// спрашивает следующую по приоритету стратегию.
type EntryFilter interface {
	EntryBlocked(ctx context.Context, ticker, instrumentID string, md strategy.MarketData) (reason string, err error)
}
