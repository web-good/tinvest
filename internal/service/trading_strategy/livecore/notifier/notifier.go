// Package notifier renders Telegram messages for the reversion runner. Functions are
// pure; the caller sends the string only when NotifyEnabled.
package notifier

import "fmt"

func paperTag(paper bool) string {
	if paper {
		return " <i>(БУМАЖНАЯ сделка, ордер не выставлен)</i>"
	}
	return ""
}

// Entry renders a buy notification.
func Entry(ticker string, price float64, lots, qty int64, paper bool) string {
	return fmt.Sprintf("🟢 <b>Вход %s</b>%s\n  Цена: %.4f | Лотов: %d | Штук: %d",
		ticker, paperTag(paper), price, lots, qty)
}

// Exit renders a sell notification with the exit reason code (OB/RSI50/BE/TRAIL/...).
func Exit(ticker, reason string, price float64, qty int64, paper bool) string {
	return fmt.Sprintf("🔴 <b>Выход %s</b> [%s]%s\n  Цена: %.4f | Штук: %d",
		ticker, reason, paperTag(paper), price, qty)
}

// Skip renders a skipped-entry notification (e.g. sub-lot budget, insufficient cash).
func Skip(ticker, reason string) string {
	return fmt.Sprintf("⏭️ <b>Пропуск %s</b>\n  %s", ticker, reason)
}

// Alert renders an operational alert (e.g. state reconstructed, order rejected).
// strategy — метка раннера в заголовке: пакет общий для нескольких стратегий, и по
// сообщению должно быть видно, чей раннер его прислал.
func Alert(strategy, ticker, message string) string {
	return fmt.Sprintf("⚠️ <b>%s %s</b>\n  %s", strategy, ticker, message)
}

// StopSet reports a protective stop order (re)placed at price for reason.
func StopSet(ticker string, price float64, reason string, paper bool) string {
	return fmt.Sprintf("🛡 %s: стоп-заявка %s на %.4f%s", ticker, reason, price, paperTag(paper))
}
