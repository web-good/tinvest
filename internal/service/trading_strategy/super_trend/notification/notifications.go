package notification

import (
	"strings"
	"tinvest/internal/domain/atr"
	notification2 "tinvest/internal/domain/notification"
)

func Trade(shares []notification2.SuperTrend) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("<u><b>Super Trend:</b></u>\n\n\n")
	notifyMessageBuilder.WriteString("<i>Пересеклись ema35 и ema5, rsi50(2ч), macd зелёный свет на текущемтайминге + на более высок таймфрейме зелёный свет macd</i>\n\n")
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
