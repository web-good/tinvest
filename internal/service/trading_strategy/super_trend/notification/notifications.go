package notification

import (
	"strings"
	"tinvest/internal/domain/atr"
	"tinvest/internal/model"
)

func Trade(shares []model.Share, atrs map[string]atr.ItemTechAnalyse) string {
	notifyMessageBuilder := strings.Builder{}
	notifyMessageBuilder.WriteString("🟢 \n<u><b>К ПОКУПКЕ (Super Trend):</b></u>\n\n\n<code>")

	for _, share := range shares {
		notifyMessageBuilder.WriteString("<u>" + share.Name + "</u>")

		if atr, exist := atrs[share.ID]; exist {
			notifyMessageBuilder.WriteString("<u>" + atr.ToString() + "</u>")
		}

		notifyMessageBuilder.WriteString("\n")
	}

	notifyMessageBuilder.WriteString("</code>")

	return notifyMessageBuilder.String()
}
