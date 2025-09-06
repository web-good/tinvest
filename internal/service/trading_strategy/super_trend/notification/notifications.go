package notification

import (
	"strings"
	"tinvest/internal/domain/atr"
	notification2 "tinvest/internal/domain/notification"
)

func Trade(shares []notification2.SuperTrend) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("<u><b>Super Trend:</b></u>\n\n\n")
	notifyMessageBuilder.WriteString("<i>Дневной MACD зеленый. на 4 часах ema5 над ema35. на одном часа ema5 над ema35 и rsi пересёк 50 или 55 снизу вверх </i>\n\n")
	notifyMessageBuilder.WriteString("<code>")

	for _, share := range shares {
		if share.Indicator == notification2.Green {
			notifyMessageBuilder.WriteString("<u> 🟢")
		}

		if share.Indicator == notification2.Yellow {
			notifyMessageBuilder.WriteString("<u> 🟡")
		}

		notifyMessageBuilder.WriteString(share.Share.Name + "</u>")

		if share.Atr != (atr.ItemTechAnalyse{}) {
			notifyMessageBuilder.WriteString(" <u>" + share.Atr.ToString() + "</u>")
		}

		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}

func TakeProfit(shares []notification2.TakeProfit) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("<u><b>Take Profit MACD RSI:</b></u>\n\n\n")
	notifyMessageBuilder.WriteString("<i>Условие по снятию прибыли</i>\n\n")
	notifyMessageBuilder.WriteString("<code>")

	for _, share := range shares {
		notifyMessageBuilder.WriteString("<u> 🟡")
		notifyMessageBuilder.WriteString(share.Share.Name + "</u>")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}

func TakeBuy(shares []notification2.TakeProfit) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("<u><b>Усреднение акций:</b></u>\n\n\n")
	notifyMessageBuilder.WriteString("<code>")

	for _, share := range shares {
		notifyMessageBuilder.WriteString("<u> 🟡")
		notifyMessageBuilder.WriteString(share.Share.Name + "</u>")
		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
