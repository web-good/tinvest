package notification

import (
	"strconv"
	"strings"
	"tinvest/internal/service/trading_strategy/golden_x/dto"
	"tinvest/internal/utils"
)

func Heartbeat(in *dto.Trade) string {
	var builder strings.Builder

	builder.WriteString("💟 <b>Процесс работает, отслеживает следующие акции:</b>\n\n")

	for _, share := range in.ShareList {
		instrumentName := utils.EscapeMarkdown(share.Name)
		builder.WriteString(
			"• <b>Акция:</b> " + instrumentName + "\n" +
				"  <b>RSI:</b> " + strconv.Itoa(share.RSILength) + "\n\n",
		)
	}

	return builder.String()
}
