package bybitdensity

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tinvest/internal/service/screener/bybitdensity/strategy"
)

func spotArchives(ctx context.Context, client *http.Client, dir, symbol, day string) (books, trades string, err error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	bookName := day + "_" + symbol + "_ob200.data.zip"
	tradeName := symbol + "_" + day + ".csv.gz"
	books, trades = filepath.Join(dir, bookName), filepath.Join(dir, tradeName)
	if err := downloadArchive(ctx, client, "https://quote-saver.bycsi.com/orderbook/spot/"+symbol+"/"+bookName, books); err != nil {
		return "", "", err
	}
	if err := downloadArchive(ctx, client, "https://public.bybit.com/spot/"+symbol+"/"+tradeName, trades); err != nil {
		return "", "", err
	}
	return books, trades, nil
}

func downloadArchive(ctx context.Context, client *http.Client, url, path string) error {
	if info, err := os.Stat(path); err == nil {
		if info.Mode().IsRegular() && info.Size() > 0 {
			return nil
		}
		return fmt.Errorf("invalid cached archive %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, response.Status)
	}
	return writeBacktestFile(path, func(w io.Writer) error {
		size, err := io.Copy(w, response.Body)
		if err != nil {
			return fmt.Errorf("download %s: %w", url, err)
		}
		if size == 0 {
			return fmt.Errorf("empty archive at %s", url)
		}
		return nil
	})
}

func writeBacktestFile(path string, write func(io.Writer) error) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	writeErr := write(file)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func writeBacktestReport(out io.Writer, dir, symbol string, from, end time.Time, profile spotProfile, notional, fee float64, result strategy.ZoneResult, volumes []backtestVolume) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tradesPath := filepath.Join(dir, "trades.csv")
	if err := writeBacktestFile(tradesPath, func(out io.Writer) error {
		w := csv.NewWriter(out)
		if err := w.Write([]string{"signal_at_msk", "entry_at_msk", "exit_at_msk", "zone_low", "zone_notional_usdt", "entry_price", "exit_price", "quantity", "stop", "take", "fees_usdt", "net_pnl_usdt", "exit_reason"}); err != nil {
			return err
		}
		for _, trade := range result.Trades {
			row := []string{trade.SignalAt.In(moscowTime).Format(time.RFC3339Nano), trade.EntryAt.In(moscowTime).Format(time.RFC3339Nano), trade.ExitAt.In(moscowTime).Format(time.RFC3339Nano)}
			for _, value := range []float64{trade.ZoneLow, trade.ZoneNotional, trade.EntryPrice, trade.ExitPrice, trade.Quantity, trade.StopPrice, trade.TakePrice, trade.Fees, trade.NetPnL} {
				row = append(row, strconv.FormatFloat(value, 'f', 8, 64))
			}
			if err := w.Write(append(row, trade.ExitReason)); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	}); err != nil {
		return err
	}

	reportPath := filepath.Join(dir, "summary.md")
	if err := writeBacktestFile(reportPath, func(w io.Writer) error {
		_, err := io.WriteString(w, renderZoneBacktest(symbol, from, end, profile, notional, fee, result, volumes))
		return err
	}); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\nЗакрыто сделок: %d | Чистый реализованный P&L: %+.4f USDT\nОтчёт: %s\nСделки CSV: %s\n",
		len(result.Trades), result.Realized, reportPath, tradesPath)
	return err
}

func renderZoneBacktest(symbol string, from, end time.Time, profile spotProfile, notional, fee float64, result strategy.ZoneResult, volumes []backtestVolume) string {
	var report strings.Builder
	const dateMinute = "2006-01-02 15:04"
	fmt.Fprintf(&report, "# Бэктест %s (Bybit Spot)\n\n", symbol)
	fmt.Fprintf(&report, "Период с %s до %s (МСК).\n\n", from.In(moscowTime).Format(dateMinute), end.In(moscowTime).Format(dateMinute))
	report.WriteString("## Параметры стратегии\n\n| Параметр | Значение |\n|---|---|\n")
	fmt.Fprintf(&report, "| Ширина зоны | %g USDT |\n", profile.zoneWidthUSDT)
	if profile.zoneFilter.Enabled() {
		fmt.Fprintf(&report, "| Минимальный объём зоны | %g USDT |\n| Множитель к большей из двух соседних зон | %g |\n", profile.zoneFilter.MinNotional, profile.zoneFilter.NeighborMultiple)
	} else {
		fmt.Fprintf(&report, "| Коэффициент к среднему объёму M5 | %g |\n", profile.m5VolumeMultiplier)
	}
	if profile.inPlay.Enabled() {
		fmt.Fprintf(&report, "| In Play | Средний диапазон шести M5 от %g%%, оборот от %g× к среднему за предшествующие 24 ч |\n", profile.inPlay.MinRange*100, profile.inPlay.MinTurnoverMultiple)
		fmt.Fprintf(&report, "| Окон M5 в In Play | %d / %d |\n", result.InPlayActiveWindows, result.InPlayWindows)
	}
	report.WriteString("| Вход | После продажи в зоне и подтверждения возврата лучшего bid выше зоны покупка по лучшему ask |\n")
	fmt.Fprintf(&report, "| Подтверждение отскока | %d с |\n", profile.bounceConfirmSeconds)
	fmt.Fprintf(&report, "| Минимальный остаток объёма от уровня перед продажей | %g%% |\n", strategy.ZoneOrderRetention*100)
	fmt.Fprintf(&report, "| Стоп-лосс | %g%% ниже нижней границы зоны |\n| Тейк-профит | %g%% выше цены входа |\n", strategy.ZoneStopLoss*100, strategy.ZoneTakeProfit*100)
	fmt.Fprintf(&report, "| Максимальный размер заявки | %g USDT |\n| Комиссия на сторону | %g%% |\n", notional, fee*100)

	report.WriteString("\n## Средний объём M5\n\n")
	if profile.zoneFilter.Enabled() {
		report.WriteString("| С (МСК) | До (МСК) | День свечей (МСК) | Средний объём M5, монет |\n|---|---|---|---:|\n")
	} else {
		report.WriteString("| Действует с (МСК) | Действует до (МСК) | День свечей (МСК) | Средний объём M5, монет | Порог зоны, монет |\n|---|---|---|---:|---:|\n")
	}
	for _, ref := range volumes {
		average := "—"
		if ref.average > 0 {
			average = strconv.FormatFloat(ref.average, 'f', 8, 64)
		}
		fmt.Fprintf(&report, "| %s | %s | %s | %s |", ref.from.In(moscowTime).Format(dateMinute),
			ref.until.In(moscowTime).Format(dateMinute), ref.referenceDay.Format(time.DateOnly), average)
		if !profile.zoneFilter.Enabled() {
			fmt.Fprintf(&report, " %.8f |", ref.minimum)
		}
		report.WriteByte('\n')
	}

	var wins, losses int
	var grossProfit, grossLoss, closedPnL, fees float64
	reasons := make(map[string]int)
	for _, trade := range result.Trades {
		closedPnL += trade.NetPnL
		fees += trade.Fees
		reasons[trade.ExitReason]++
		switch {
		case trade.NetPnL > 0:
			wins++
			grossProfit += trade.NetPnL
		case trade.NetPnL < 0:
			losses++
			grossLoss -= trade.NetPnL
		}
	}

	report.WriteString("\n## Сводка метрик\n\n| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(&report, "| Закрыто сделок | %d |\n| Прибыльных / убыточных / безубыточных | %d / %d / %d |\n",
		len(result.Trades), wins, losses, len(result.Trades)-wins-losses)
	if len(result.Trades) > 0 {
		fmt.Fprintf(&report, "| Win rate | %.2f%% |\n| Средний чистый P&L сделки | %+.4f USDT |\n",
			100*float64(wins)/float64(len(result.Trades)), closedPnL/float64(len(result.Trades)))
	}
	profitFactor := "—"
	if grossLoss > 0 {
		profitFactor = fmt.Sprintf("%.3f", grossProfit/grossLoss)
	} else if grossProfit > 0 {
		profitFactor = "∞"
	}
	fmt.Fprintf(&report, "| Profit factor после комиссий | %s |\n", profitFactor)
	fmt.Fprintf(&report, "| Стоп | %d |\n| Тейк | %d |\n| Частичный тейк и стоп | %d |\n",
		reasons["stop"], reasons["take"], reasons["take+stop"])
	fmt.Fprintf(&report, "| Начато наблюдений за зоной | %d |\n| Наблюдений завершено без покупки | %d |\n", result.Placed, result.Cancelled)
	fmt.Fprintf(&report, "| P&L закрытых сделок до комиссий | %+.4f USDT |\n| Комиссии по закрытым сделкам | %.4f USDT |\n", closedPnL+fees, fees)
	fmt.Fprintf(&report, "| Чистый реализованный P&L | %+.4f USDT |\n| Максимальная просадка | %.4f USDT |\n", result.Realized, result.MaxDrawdown)

	writeEntryDiagnostics(&report, result)

	report.WriteString("\n## Журнал сделок\n\n[Сделки CSV](trades.csv)\n\n")
	if len(result.Trades) == 0 {
		report.WriteString("Закрытых сделок за период нет.\n")
	} else {
		report.WriteString("| № | Вход (МСК) | Цена входа | Нижняя граница зоны | Объём зоны, USDT | Количество | Стоп | Тейк | Выход (МСК) | Цена выхода | Причина выхода | Комиссия, USDT | Чистый P&L, USDT |\n")
		report.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---|---:|---|---:|---:|\n")
		const dateTime = "2006-01-02 15:04:05.000"
		for i, trade := range result.Trades {
			fmt.Fprintf(&report, "| %d | %s | %.8f | %.8f | %.2f | %.8f | %.8f | %.8f | %s | %.8f | %s | %.4f | %+.4f |\n",
				i+1, trade.EntryAt.In(moscowTime).Format(dateTime), trade.EntryPrice, trade.ZoneLow, trade.ZoneNotional,
				trade.Quantity, trade.StopPrice, trade.TakePrice, trade.ExitAt.In(moscowTime).Format(dateTime), trade.ExitPrice,
				trade.ExitReason, trade.Fees, trade.NetPnL)
		}
	}

	report.WriteString("\n## Состояние на конец прогона\n\n")
	pending := "нет"
	if result.Pending {
		pending = "есть"
	}
	fmt.Fprintf(&report, "- Активное наблюдение за зоной — %s.\n", pending)
	fmt.Fprintf(&report, "- Остаток позиции — %.8f.\n", result.OpenQuantity)
	fmt.Fprintf(&report, "- Нереализованный P&L по последнему bid — %+.4f USDT.\n", result.Unrealized)
	return report.String()
}

func writeEntryDiagnostics(report *strings.Builder, result strategy.ZoneResult) {
	report.WriteString("\n## Диагностика входов\n\n| Причина завершения наблюдения | Случаев |\n|---|---:|\n")
	for _, item := range []struct {
		reason strategy.ZoneCancelReason
		label  string
	}{
		{strategy.ZoneCancelVolume, "Потеря объёма или неполная видимость зоны"},
		{strategy.ZoneCancelStop, "Касание уровня стопа до входа"},
		{strategy.ZoneCancelReturn, "Возврат bid к зоне во время подтверждения"},
		{strategy.ZoneCancelFeed, "Сброс данных стакана"},
		{strategy.ZoneCancelReference, "Смена порога отбора"},
		{strategy.ZoneCancelUnfilled, "Покупка по ask не исполнилась"},
		{strategy.ZoneCancelDistance, "Зона дальше 0,30% от bid"},
		{strategy.ZoneCancelOtherEntry, "Открыта позиция по другой зоне"},
	} {
		fmt.Fprintf(report, "| %s | %d |\n", item.label, result.Cancellations[item.reason])
	}
	if result.InPlayWindows > 0 {
		fmt.Fprintf(report, "| Выход из In Play | %d |\n", result.Cancellations[strategy.ZoneCancelInPlay])
	}
	report.WriteString("\n| Этап отмены | Случаев |\n|---|---:|\n")
	fmt.Fprintf(report, "| До рыночной продажи в зоне | %d |\n| После продажи, до выхода bid выше зоны | %d |\n| После начала подтверждения отскока | %d |\n",
		result.CancelledBeforeTouch, result.CancelledBeforeBounce, result.CancelledDuringConfirmation)
	fmt.Fprintf(report, "\nМаксимум зон под одновременным наблюдением — %d.\n", result.MaxPendingZones)
}
