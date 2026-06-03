package notification

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

func TestTrade_RendersBuyAndSell(t *testing.T) {
	signals := []model.Signal{
		{Kind: model.SignalBuy, InstrumentName: "Sberbank", Ticker: "SBER", Price: 100, TakeProfit: 103, StopLoss: 98, RSI: 36},
		{Kind: model.SignalSell, InstrumentName: "Gazprom", Ticker: "GAZP", Price: 104, TakeProfit: 103, StopLoss: 98, Reason: "TP"},
	}

	got := Trade(signals)

	for _, want := range []string{"покупку", "продажу", "Sberbank", "SBER", "Gazprom", "GAZP", "TP"} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q\n---\n%s", want, got)
		}
	}
}

func TestTrade_OnlyBuysOmitsSellSection(t *testing.T) {
	signals := []model.Signal{
		{Kind: model.SignalBuy, InstrumentName: "Sberbank", Ticker: "SBER", Price: 100, TakeProfit: 103, StopLoss: 98, RSI: 36},
	}

	got := Trade(signals)

	if strings.Contains(got, "продажу") {
		t.Errorf("sell section should be absent\n---\n%s", got)
	}
}

func TestTrade_NoSignals(t *testing.T) {
	got := Trade(nil)

	if strings.Contains(got, "покупку") {
		t.Errorf("buy section should be absent for nil signals\n---\n%s", got)
	}
	if strings.Contains(got, "продажу") {
		t.Errorf("sell section should be absent for nil signals\n---\n%s", got)
	}
}

func TestSellWatch_UsesMonitorHeaderAndRendersSells(t *testing.T) {
	signals := []model.Signal{
		{Kind: model.SignalSell, InstrumentName: "Gazprom", Ticker: "GAZP", Price: 104, TakeProfit: 103, StopLoss: 98, Reason: "TP"},
	}

	got := SellWatch(signals)

	for _, want := range []string{"Мониторинг выхода", "продажу", "Gazprom", "GAZP", "TP"} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(got, "Скальпинг (1H)") {
		t.Errorf("sell-watch must not use the hourly header\n---\n%s", got)
	}
}
