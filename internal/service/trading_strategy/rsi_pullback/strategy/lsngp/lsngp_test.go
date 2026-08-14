package lsngp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Пакет заведён ДО калибровки и обязан ОТСЛЕЖИВАТЬ baseline ядра, а не хранить его копию.
// Разница видна только в момент, когда baseline меняют: копия молча останется старой, и
// каталог сеток lsngp/ начнёт свипать оси вокруг точки, которой у стратегии больше нет —
// калибратор ведь засевает несвипуемые поля именно из DefaultParams() этого пакета. Тест
// красный ровно в двух случаях: кто-то поставил сюда литерал, не заменив этот тест снимком
// (как это сделано в lent/, wush/, fesh/), либо baseline разъехался с пакетом.
func TestParamsTrackTheBaseline(t *testing.T) {
	if got, want := DefaultParams(), core.DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want baseline ядра %+v.\nЕсли LSNGP откалиброван — заменить этот тест снимком литерала и обновить док пакета", got, want)
	}
}

func TestTicker(t *testing.T) {
	if Ticker != "LSNGP" {
		t.Fatalf("Ticker = %q, want LSNGP", Ticker)
	}
}
