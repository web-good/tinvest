package bybitdensity

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/strategy"
)

func TestSpotArchivesReusesCachedFiles(t *testing.T) {
	dir := t.TempDir()
	books := filepath.Join(dir, "2026-11-22_SOLUSDT_ob200.data.zip")
	trades := filepath.Join(dir, "SOLUSDT_2026-11-22.csv.gz")
	for _, path := range []string{books, trades} {
		if err := os.WriteFile(path, []byte("cached archive"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A transport that rejects all requests proves that cache hits do not access the network.
	gotBooks, gotTrades, err := spotArchives(t.Context(), &http.Client{Transport: rejectArchiveRequests{t: t}}, dir, "SOLUSDT", "2026-11-22")
	if err != nil || gotBooks != books || gotTrades != trades {
		t.Fatalf("cached archives = %s, %s, %v", gotBooks, gotTrades, err)
	}
}

type rejectArchiveRequests struct{ t *testing.T }

func (r rejectArchiveRequests) RoundTrip(req *http.Request) (*http.Response, error) {
	r.t.Fatalf("unexpected archive request to %s", req.URL)
	return nil, nil
}

func TestPreviousSpotVolumeWindowSkipsWeekend(t *testing.T) {
	for _, day := range []int{21, 22, 23} {
		now := time.Date(2026, 11, day, 0, 0, 0, 0, moscowTime)
		from, to, until := previousSpotVolumeWindow(now)
		want := time.Date(2026, 11, 20, 0, 0, 0, 0, moscowTime)
		if !from.Equal(want) || !to.Equal(want.Add(24*time.Hour)) || !until.Equal(want.Add(4*24*time.Hour)) {
			t.Fatalf("day %d: window = %s to %s, expires %s", day, from, to, until)
		}
	}
}

func TestWriteBacktestReportCreatesMarkdownWithTradeLink(t *testing.T) {
	var out bytes.Buffer
	dir := t.TempDir()
	day := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	result := strategy.ZoneResult{
		Trades:   []strategy.ZoneTrade{{SignalAt: day, EntryAt: day, ExitAt: day, Fees: .50, NetPnL: -.75, ExitReason: "stop"}},
		Realized: -.75, MaxDrawdown: .75,
		Cancellations:   map[strategy.ZoneCancelReason]int{strategy.ZoneCancelVolume: 3},
		MaxPendingZones: 4,
	}
	err := writeBacktestReport(&out, dir, "SOLUSDT", day, day.AddDate(0, 0, 1), spotProfile{zoneWidthUSDT: .01, m5VolumeMultiplier: 1}, 250, .001, result, []backtestVolume{{from: day, until: day.AddDate(0, 0, 1), referenceDay: day.AddDate(0, 0, -2), average: 123.25, minimum: 123.25}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"# Бэктест SOLUSDT", "| Чистый реализованный P&L | -0.7500 USDT |", "| Стоп | 1 |", "## Журнал сделок", "## Диагностика входов", "| Потеря объёма или неполная видимость зоны | 3 |", "Максимум зон под одновременным наблюдением — 4.", "2026-11-22 03:00:00.000", "| 2026-11-20 | 123.25000000 | 123.25000000 |", "| stop | 0.5000 | -0.7500 |", "[Сделки CSV](trades.csv)"} {
		if !strings.Contains(string(report), text) {
			t.Errorf("missing %q in report", text)
		}
	}
	if strings.Contains(string(report), "RSI") {
		t.Errorf("report contains an RSI field: %s", report)
	}
	csvData, err := os.ReadFile(filepath.Join(dir, "trades.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(csvData), "entry_rsi") {
		t.Fatalf("CSV contains an RSI field: %s", csvData)
	}
	if !strings.HasPrefix(string(csvData), "signal_at_msk,entry_at_msk,exit_at_msk,") ||
		!strings.Contains(string(csvData), "2026-11-22T03:00:00+03:00") {
		t.Fatalf("CSV did not include Moscow timestamps: %s", csvData)
	}
	if !strings.Contains(out.String(), filepath.Join(dir, "summary.md")) {
		t.Fatalf("terminal output did not include report path: %s", &out)
	}
}
