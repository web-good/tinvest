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
	return fmt.Sprintf("%s · %s · плотность на %s\nЦена %s USDT\nОбъём %.8g монет (%.2f USDT)\n%.2f× к медиане · %.3f%% от лучшей цены\nДержится %s\nСделки на цене %.8g монет за %s",
		marketLabel(signal.Key.Market), signal.Key.Symbol, side, priceText(signal.Price, tick),
		signal.Qty, signal.Notional, signal.Multiple, signal.Distance*100,
		signal.Observed.Truncate(time.Second), signal.TradeQty, signal.TradeCoverage.Truncate(time.Second))
}

func formatFill(fill model.Fill, tick float64) string {
	direction := "лонг"
	if fill.Side == model.Ask {
		direction = "шорт"
	}
	title := fmt.Sprintf("СИМУЛЯЦИЯ · %s · %s · %s", marketLabel(fill.Key.Market), fill.Key.Symbol, direction)
	if fill.Entry {
		return fmt.Sprintf("%s\nВход по %s USDT", title, priceText(fill.Price, tick))
	}
	action := "выход"
	if fill.Partial {
		action = "частичный выход"
	}
	price := priceText(fill.Price, tick)
	if tick > 0 && math.Abs(fill.Price/tick-math.Round(fill.Price/tick)) > 1e-7 {
		weighted := strconv.FormatFloat(fill.Price, 'f', pricePrecision(tick)+4, 64)
		price = strings.TrimRight(strings.TrimRight(weighted, "0"), ".")
	}
	reason := fill.Reason
	switch reason {
	case "stop":
		reason = "стоп"
	case "density_lost":
		reason = "плотность исчезла"
	case "take":
		reason = "тейк"
	}
	return fmt.Sprintf("%s\n%s по %s USDT\nПричина — %s\nРезультат после комиссий %+.2f%% (%+.4f USDT)",
		title, action, price, reason, fill.NetReturn*100, fill.NetPnL)
}
