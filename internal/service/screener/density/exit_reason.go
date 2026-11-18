package density

// ExitReasonText translates journal reason codes for trading notifications.
func ExitReasonText(reason string) string {
	switch reason {
	case "density_disappeared":
		return "Плотность исчезла"
	case "density_lost":
		return "Плотность исчезла или её объём упал ниже порога"
	case "stop", "stop_loss", "stop_price", "stop_trade", "broker_stop":
		return "Стоп-лосс"
	case ExitRSIProfit:
		return "Ранний выход по RSI(6) на 1м"
	case "take", "take_profit":
		return "Тейк-профит"
	case "max_hold", "position_timeout":
		return "Истекло время удержания"
	case "data_unavailable", "data_gap", "invalid_book", "off_tick_book", "stale_book", "stream_disconnected":
		return "Потеря или некорректность рыночных данных"
	case "trading_closed":
		return "Торги закрыты"
	case "protection_missing":
		return "Защитное поручение отменено или недоступно"
	case "protection_rejected":
		return "Брокер отклонил защитное поручение"
	case "restart_reconcile":
		return "Сверка позиции после перезапуска"
	case "shutdown":
		return "Остановка приложения"
	case "partial_entry":
		return "Частичное исполнение входа"
	case "broker_exit":
		return "Исполнено поручение на выход у брокера"
	case "":
		return "Причина не определена"
	default:
		return reason
	}
}
