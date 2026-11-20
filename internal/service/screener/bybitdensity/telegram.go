package bybitdensity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"

	tgbot "github.com/go-telegram/bot"
)

type sendMessage func(context.Context, string) error

type telegramDestination struct {
	chatID  int64
	topicID int
}

func destinations(send func(context.Context, telegramDestination, string) error, screener, simulation telegramDestination) (sendMessage, sendMessage) {
	bind := func(destination telegramDestination) sendMessage {
		return func(ctx context.Context, msg string) error { return send(ctx, destination, msg) }
	}
	return bind(screener), bind(simulation)
}

func telegramSender(token string) (func(context.Context, telegramDestination, string) error, error) {
	bot, err := tgbot.New(token, tgbot.WithSkipGetMe())
	if err != nil {
		return nil, fmt.Errorf("telegram setup: %s", strings.ReplaceAll(err.Error(), token, "[redacted]"))
	}
	return func(ctx context.Context, destination telegramDestination, message string) error {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		_, err := bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: destination.chatID, MessageThreadID: destination.topicID, Text: message,
		})
		if err != nil {
			return errors.New(strings.ReplaceAll(err.Error(), token, "[redacted]"))
		}
		slog.Info("Bybit Telegram alert delivered", "chat", destination.chatID, "topic", destination.topicID)
		return nil
	}, nil
}

func marketLabel(market model.Market) string {
	if market == model.Spot {
		return "Спот"
	}
	return "USDT-фьючерс"
}

func priceText(price, tick float64) string {
	if tick <= 0 {
		return strconv.FormatFloat(price, 'f', -1, 64)
	}
	return strconv.FormatFloat(price, 'f', pricePrecision(tick), 64)
}

func pricePrecision(tick float64) int {
	step := strconv.FormatFloat(tick, 'f', -1, 64)
	_, decimals, _ := strings.Cut(step, ".")
	return len(decimals)
}

func formatSignal(signal model.Signal, tick float64) string {
	side := "покупка (bid)"
	if signal.Side == model.Ask {
		side = "продажа (ask)"
	}
	msg := fmt.Sprintf("%s · %s · плотность на %s\nЦена %s USDT\nОбъём %.8g монет (%.2f USDT)\n%.2f× к медиане · %.3f%% от лучшей цены\nДержится %s\nСделки на цене %.8g монет за %s",
		marketLabel(signal.Key.Market), signal.Key.Symbol, side, priceText(signal.Price, tick),
		signal.Qty, signal.Notional, signal.Multiple, signal.Distance*100,
		signal.Observed.Truncate(time.Second), signal.TradeQty, signal.TradeCoverage.Truncate(time.Second))
	if signal.HasAverageVolume {
		msg += fmt.Sprintf("\nСредний объём M5: %.8g монет", signal.AverageVolume)
	}
	return msg
}

var moscowTime = time.FixedZone("МСК", 3*60*60)

func formatClosedTrade(fill model.Fill, tick float64) string {
	trade := fill.Closed
	if trade == nil {
		return ""
	}
	direction, entryAction, exitAction := "лонг", "Купили", "Продали"
	if fill.Side == model.Ask {
		direction, entryAction, exitAction = "шорт", "Продали", "Выкупили"
	}
	const timeLayout = "2006-01-02 15:04:05 MST"
	reasons := make([]string, len(trade.ExitReasons))
	for i, reason := range trade.ExitReasons {
		reasons[i] = exitReasonText(reason)
	}
	if trade.Imbalance != nil {
		return formatBreakoutTrade(fill, tick, strings.Join(reasons, " → "), timeLayout)
	}
	msg := fmt.Sprintf("СИМУЛЯЦИЯ · %s · %s · %s\nСделка закрыта\n%s по %s USDT · %s\n%s по %s USDT · %s\nПлотность при входе — %s USDT · %.8g монет (%.2f USDT)\nСтоп-лосс — %s USDT\nТейк-профит — %s USDT\nПричина — %s\nРезультат после комиссий %+.2f%% (%+.4f USDT)",
		marketLabel(fill.Key.Market), fill.Key.Symbol, direction,
		entryAction, averagePriceText(trade.EntryPrice, tick), trade.EntryAt.In(moscowTime).Format(timeLayout),
		exitAction, averagePriceText(trade.ExitPrice, tick), fill.At.In(moscowTime).Format(timeLayout),
		priceText(trade.DensityPrice, tick), trade.DensityQty, trade.DensityPrice*trade.DensityQty,
		priceText(trade.StopPrice, tick), priceText(trade.TakePrice, tick), strings.Join(reasons, " → "),
		trade.NetReturn*100, trade.NetPnL)
	if trade.AverageVolume != nil {
		msg += fmt.Sprintf("\nСредний объём M5 при входе: %.8g монет", *trade.AverageVolume)
	}
	return msg
}

func formatBreakoutTrade(fill model.Fill, tick float64, reason, timeLayout string) string {
	trade := fill.Closed
	variant := "без фильтра"
	if fill.Variant == model.BreakoutImbalance {
		variant = "с фильтром"
	}
	return fmt.Sprintf("СИМУЛЯЦИЯ · Спот · %s · пробой ask (%s)\nКупили по %s USDT · %s\nПродали по %s USDT · %s\nУровень пробоя — %s USDT · %.8g монет\nДисбаланс стакана %+.3f\nПричина — %s\nРезультат после комиссий %+.2f%% (%+.4f USDT)",
		fill.Key.Symbol, variant, averagePriceText(trade.EntryPrice, tick), trade.EntryAt.In(moscowTime).Format(timeLayout),
		averagePriceText(trade.ExitPrice, tick), fill.At.In(moscowTime).Format(timeLayout), priceText(trade.DensityPrice, tick),
		trade.DensityQty, *trade.Imbalance, reason, trade.NetReturn*100, trade.NetPnL)
}

func averagePriceText(price, tick float64) string {
	if tick > 0 && math.Abs(price/tick-math.Round(price/tick)) > 1e-7 {
		weighted := strconv.FormatFloat(price, 'f', pricePrecision(tick)+4, 64)
		return strings.TrimRight(strings.TrimRight(weighted, "0"), ".")
	}
	return priceText(price, tick)
}

func exitReasonText(reason string) string {
	switch reason {
	case "stop":
		return "стоп-лосс"
	case "density_lost":
		return "плотность исчезла"
	case "breakout_failed":
		return "bid вернулся под уровень пробоя"
	case "timeout":
		return "истекли 10 секунд"
	case "take":
		return "тейк"
	default:
		return reason
	}
}
