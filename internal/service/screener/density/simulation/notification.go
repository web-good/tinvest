// Package simulation connects market-data-only density paper trading to alerts.
package simulation

import (
	"fmt"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
)

func FormatFill(r paper.Record) string {
	if r.Order == nil || (r.Kind != "entry_filled" && r.Kind != "exit_filled") {
		return ""
	}
	action := "Купили бы"
	if r.Kind == "exit_filled" {
		action = "Продали бы"
	}
	msg := fmt.Sprintf("СИМУЛЯЦИЯ — реальных заявок нет\nDensity %s\n%s по %.4f ₽\nВремя: %s", r.Order.Ticker, action, r.FillRubles, r.At.Format("2006-01-02 15:04:05 MST"))
	if r.Kind == "exit_filled" {
		msg += fmt.Sprintf("\nПричина: %s", density.ExitReasonText(r.Detail))
	}
	return msg
}
