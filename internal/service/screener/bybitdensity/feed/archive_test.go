package feed

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestReplaySpotDayOrdersTradesBeforeBookAtSameTime(t *testing.T) {
	books, trades := writeArchiveFixture(t,
		archiveBookFixture(1000, 1)+archiveBookFixture(1001, 2),
		"1,1001,140.01,1,sell,0\n")
	instrument := archiveFixtureInstrument()
	var sequence strings.Builder
	var replay SpotArchiveReplay
	err := replay.Day(context.Background(), instrument, time.Unix(0, 0).UTC(), books, trades, func(event model.Event) {
		if event.Key != instrument.Key {
			t.Fatalf("event key = %+v", event.Key)
		}
		if event.Book != nil {
			sequence.WriteByte('b')
		}
		if len(event.Trades) != 0 {
			if !event.Trades[0].Sell {
				t.Error("sell trade lost its side")
			}
			sequence.WriteByte('t')
		}
	})
	if err != nil || sequence.String() != "btb" {
		t.Fatalf("events = %q, error = %v", sequence.String(), err)
	}
}

func TestReplaySpotDayMergesOverlappingMidnightSnapshots(t *testing.T) {
	firstBooks, firstTrades := writeArchiveFixture(t,
		archiveBookFixture(86399900, 1)+archiveBookFixture(86400050, 2)+archiveBookFixture(86400200, 3),
		"1,86399950,140.01,1,sell,0\n")
	secondBooks, secondTrades := writeArchiveFixture(t,
		archiveBookFixture(86400200, 3)+archiveBookFixture(86400250, 4)+archiveBookFixture(172800250, 5),
		"1,86400100,140.01,1,sell,0\n")
	var timestamps []int64
	emit := func(event model.Event) {
		if event.Book != nil {
			timestamps = append(timestamps, event.Book.At.UnixMilli())
		}
		if len(event.Trades) != 0 {
			timestamps = append(timestamps, event.Trades[0].At.UnixMilli())
		}
	}
	var replay SpotArchiveReplay
	instrument := archiveFixtureInstrument()
	days := [][2]string{{firstBooks, firstTrades}, {secondBooks, secondTrades}}
	for i, paths := range days {
		if err := replay.Day(context.Background(), instrument, time.Unix(0, 0).UTC().AddDate(0, 0, i), paths[0], paths[1], emit); err != nil {
			t.Fatal(err)
		}
		if i == 0 && len(timestamps) != 2 {
			t.Errorf("first day exposed later events: %v", timestamps)
		}
	}
	want := []int64{86399900, 86399950, 86400050, 86400100, 86400200, 86400250}
	if !reflect.DeepEqual(timestamps, want) {
		t.Fatalf("timestamps = %v, want %v", timestamps, want)
	}
}

func archiveFixtureInstrument() model.Instrument {
	return model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "SOLUSDT"}, TickSize: .01, TickSizeText: "0.01"}
}

func archiveBookFixture(at, id int64) string {
	return fmt.Sprintf(`{"topic":"orderbook.200.SOLUSDT","type":"snapshot","ts":%d,"cts":%d,"data":{"s":"SOLUSDT","b":[["140.01","3"]],"a":[["140.02","3"]],"u":%d,"seq":%d}}`+"\n",
		at+1, at, id, id)
}

func writeArchiveFixture(t *testing.T, bookRows, tradeRows string) (books, trades string) {
	t.Helper()
	dir := t.TempDir()
	books, trades = filepath.Join(dir, "book.zip"), filepath.Join(dir, "trades.gz")
	archiveFile, err := os.Create(books)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(archiveFile)
	member, err := archive.Create("book.data")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := member.Write([]byte(bookRows)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}
	tradeFile, err := os.Create(trades)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(tradeFile)
	if _, err := gz.Write([]byte("id,timestamp,price,volume,side,rpi\n" + tradeRows)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tradeFile.Close(); err != nil {
		t.Fatal(err)
	}
	return books, trades
}

func TestArchiveTradeFinerPricePreservesTradeThrough(t *testing.T) {
	tick, err := parseDecimal("0.1")
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader("1,1001,59966.69,0.001653,sell,0\n2,1002,59966.69,0.002,buy,0\n"))
	for _, want := range []int64{599666, 599667} {
		trade, err := readArchiveTrade(reader, tick)
		if err != nil || trade.Ticks != want {
			t.Fatalf("trade = %+v, error = %v, want %d ticks", trade, err, want)
		}
	}
}
