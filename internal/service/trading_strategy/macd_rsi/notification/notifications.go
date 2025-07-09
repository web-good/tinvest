package notification

import (
	"strings"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
)

func Trade(shares []model.Share, atrs map[string]atr.ItemTechAnalyse) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>Смена тренда, заходить после коррекции (rsi5ч+macd5ч):</b></u>\n\n\n<code>")

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
