// Package simulation connects market-data-only density paper trading to alerts.
package simulation

import (
	"fmt"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/paper"
)

func FormatFill(r paper.Record) string {
	if r.Order == nil || r.Kind != "exit_filled" || r.Order.State != "closed" || r.Order.OpenLots != 0 {
		return ""
	}
	o := r.Order
	exit := r.FillRubles
	if o.FilledLots > 0 && o.Lot > 0 && r.EntryRubles > 0 {
		// GrossPnL includes every exit fill, before commissions.
		exit = r.EntryRubles + o.GrossPnL/(float64(o.FilledLots)*float64(o.Lot))
	}
	moscow := time.FixedZone("МСК", 3*60*60)
	const layout = "2006-01-02 15:04:05 MST"
	msg := fmt.Sprintf("СИМУЛЯЦИЯ — реальных заявок нет\nDensity %s — Сделка завершена\nКупили бы по %.4f ₽\nВремя покупки: %s\nПлотность при входе: %.4f ₽\nРазмер плотности: %d лотов\nСтоп-лосс: %.4f ₽\nТейк-профит: %.4f ₽\nПродали бы по %.4f ₽\nВремя продажи: %s\nПричина: %s",
		o.Ticker, r.EntryRubles, o.Opened.In(moscow).Format(layout),
		float64(o.Density)/1e9, o.DensityLots, r.StopRubles, r.TakeRubles,
		exit, o.Closed.In(moscow).Format(layout), density.ExitReasonText(r.Detail))
	if r.EntryRubles > 0 {
		msg += fmt.Sprintf("\nРезультат: %+.2f%%", (exit-r.EntryRubles)/r.EntryRubles*100)
	}
	return msg
}
