package density

import (
	"fmt"
	"html"
	"strings"
)

func FormatSignal(s Signal) string {
	price := fmt.Sprintf("%d.%09d", s.Price.Units, s.Price.Nano)
	price = strings.TrimRight(strings.TrimRight(price, "0"), ".")
	msg := fmt.Sprintf("<b>%s</b> — покупка\nЦена: %s ₽\nОбъём: %d лотов (%.0f ₽)\nНаблюдается: %.0f с\nРасстояние от лучшего bid: %.2f%%\nКратность медиане: %.2f", html.EscapeString(s.Ticker), price, s.Quantity, s.Rubles, s.Age.Seconds(), s.Distance*100, s.Multiple)
	if s.TradeCoverage > 0 {
		msg += fmt.Sprintf("\nСделки на уровне: %d лотов за %.0f с наблюдения ленты", s.TradeLots, s.TradeCoverage.Seconds())
	}
	return msg
}
