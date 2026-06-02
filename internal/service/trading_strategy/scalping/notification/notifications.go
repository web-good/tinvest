package notification

import (
	"fmt"
	"strings"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// Trade renders an aggregated HTML Telegram message for the given signals.
func Trade(signals []model.Signal) string {
	var buys, sells []model.Signal
	for _, s := range signals {
		switch s.Kind {
		case model.SignalBuy:
			buys = append(buys, s)
		case model.SignalSell:
			sells = append(sells, s)
		}
	}

	b := strings.Builder{}
	b.WriteString("⚡️ <b>Скальпинг (1H)</b>\n\n")

	if len(buys) > 0 {
		b.WriteString("<u><b>Сигналы на покупку:</b></u>\n")
		for _, s := range buys {
			b.WriteString(fmt.Sprintf(
				"🟢 <b>%s</b> (%s)\n  Цена: %.2f | TP: %.2f | SL: %.2f | RSI: %.0f\n",
				s.InstrumentName, s.Ticker, s.Price, s.TakeProfit, s.StopLoss, s.RSI,
			))
		}
		b.WriteString("\n")
	}

	if len(sells) > 0 {
		b.WriteString("<u><b>Сигналы на продажу:</b></u>\n")
		for _, s := range sells {
			b.WriteString(fmt.Sprintf(
				"🔴 <b>%s</b> (%s) [%s]\n  Цена: %.2f | TP: %.2f | SL: %.2f\n",
				s.InstrumentName, s.Ticker, s.Reason, s.Price, s.TakeProfit, s.StopLoss,
			))
		}
	}

	return b.String()
}
