package notification

import (
	"strings"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/macd_rsi/enum"
)

func Trade(shares []model.Share, atrs map[string]atr.ItemTechAnalyse, interval enum.Interval) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>RSI MACD:</b></u>\n\n\n<code>")
	notifyMessageBuilder.WriteString(interval.String())

	for _, share := range shares {
		notifyMessageBuilder.WriteString("<u>" + share.Name + "</u>")

		if atr, exist := atrs[share.ID]; exist {
			notifyMessageBuilder.WriteString(" <u>" + atr.ToString() + "</u>")
		}

		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
