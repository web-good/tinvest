package notification

import (
	"fmt"
	"strings"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// Trade renders an aggregated HTML Telegram message for the hourly run.
func Trade(signals []model.Signal) string {
	return render("⚡️ <b>Скальпинг (1H)</b>\n\n", signals)
}

// SellWatch renders an aggregated HTML Telegram message for the out-of-schedule exit-monitor run.
func SellWatch(signals []model.Signal) string {
	return render("⚠️ <b>Мониторинг выхода (1H)</b>\n\n", signals)
}

func render(header string, signals []model.Signal) string {
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
	b.WriteString(header)

	if len(buys) > 0 {
		b.WriteString("<u><b>Сигналы на покупку:</b></u>\n")
		for _, s := range buys {
			fmt.Fprintf(&b, "🟢 <b>%s</b> (%s)\n  Цена: %.2f | TP: %.2f | SL: %.2f | RSI: %.0f\n",
				s.InstrumentName, s.Ticker, s.Price, s.TakeProfit, s.StopLoss, s.RSI)
		}
		b.WriteString("\n")
	}

	if len(sells) > 0 {
		b.WriteString("<u><b>Сигналы на продажу:</b></u>\n")
		for _, s := range sells {
			fmt.Fprintf(&b, "🔴 <b>%s</b> (%s) [%s]\n  Цена: %.2f | TP: %.2f | SL: %.2f\n",
				s.InstrumentName, s.Ticker, s.Reason, s.Price, s.TakeProfit, s.StopLoss)
		}
	}

	return b.String()
}
