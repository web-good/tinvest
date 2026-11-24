package feed

import (
	"archive/zip"
	"bufio"
	"cmp"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	spotArchiveDepth = 200
	archiveFreshness = 3 * time.Second
)

// SpotArchiveReplay merges daily order book and trade files in exchange time.
// Use one instance per instrument and call Day in date order.
type SpotArchiveReplay struct {
	lastBook archiveBook
	tail     []archiveBook
	stale    bool
}

type archiveBook struct {
	at      time.Time
	book    *model.Book
	id, seq int64
}

func (r *SpotArchiveReplay) Day(ctx context.Context, instrument model.Instrument, day time.Time, books, trades string, emit func(model.Event)) error {
	end := day.AddDate(0, 0, 1)
	tick, err := parseDecimal(instrument.TickSizeText)
	if err != nil || tick.mantissa == 0 {
		return fmt.Errorf("invalid tick size %q", instrument.TickSizeText)
	}
	archive, err := zip.OpenReader(books)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()
	if len(archive.File) != 1 || archive.File[0].FileInfo().IsDir() {
		return fmt.Errorf("expected one orderbook file in %s", books)
	}
	bookFile, err := archive.File[0].Open()
	if err != nil {
		return err
	}
	defer func() { _ = bookFile.Close() }()

	tradeFile, err := os.Open(trades)
	if err != nil {
		return err
	}
	defer func() { _ = tradeFile.Close() }()
	gz, err := gzip.NewReader(tradeFile)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	reader := csv.NewReader(gz)
	reader.ReuseRecord = true
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("trade header: %w", err)
	}
	if len(header) < 5 || header[0] != "id" || header[1] != "timestamp" || header[2] != "price" || header[3] != "volume" || header[4] != "side" {
		return fmt.Errorf("unexpected trade header %q", header)
	}

	var trade model.Trade
	readNext := func() error {
		next, err := readArchiveTrade(reader, tick)
		if err == io.EOF {
			trade = model.Trade{}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archived trade: %w", err)
		}
		if next.At.Before(day) || !next.At.Before(end) {
			return fmt.Errorf("trade timestamp %s outside UTC day %s", next.At, day.Format(time.DateOnly))
		}
		if next.At.Before(trade.At) {
			return fmt.Errorf("out-of-order trade at %s", next.At)
		}
		trade = next
		return nil
	}
	if err := readNext(); err != nil {
		return err
	}

	key := instrument.Key
	emitTrade := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.expire(key, trade.At, emit)
		emit(model.Event{Key: key, Trades: []model.Trade{trade}})
		return readNext()
	}
	emitBook := func(update archiveBook) error {
		if update.at.Before(r.lastBook.at) {
			return fmt.Errorf("out-of-order archive timestamp %s", update.at)
		}
		if update.at.Equal(r.lastBook.at) && update.id == r.lastBook.id && update.seq == r.lastBook.seq {
			return nil
		}
		for !trade.At.IsZero() && !trade.At.After(update.at) {
			if err := emitTrade(); err != nil {
				return err
			}
		}
		r.expire(key, update.at, emit)
		r.lastBook = archiveBook{at: update.at, id: update.id, seq: update.seq}
		r.stale = false
		if update.book != nil {
			emit(model.Event{Key: key, Book: update.book})
		}
		return nil
	}

	// The previous ZIP can end just after midnight. Replay its overlap against
	// today's trades before handling today's initial (often identical) snapshot.
	for _, update := range r.tail {
		if err := emitBook(update); err != nil {
			return err
		}
	}
	clear(r.tail)
	r.tail = r.tail[:0]

	scanner := bufio.NewScanner(bookFile)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	topic := fmt.Sprintf("orderbook.%d.%s", spotArchiveDepth, key.Symbol)
	var state bookState
	for line := 1; scanner.Scan(); line++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		var message streamMessage
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			return fmt.Errorf("book line %d: %w", line, err)
		}
		if message.Topic != topic {
			return fmt.Errorf("book line %d: unexpected topic %q", line, message.Topic)
		}
		at := time.UnixMilli(int64(cmp.Or(message.CTS, message.TS)))
		book, err := state.apply(key.Symbol, tick, spotArchiveDepth, message, at)
		if err != nil {
			return fmt.Errorf("book line %d: %w", line, err)
		}
		if state.bids == nil || at.Before(day) {
			continue
		}
		update := archiveBook{at: at, book: book, id: state.id, seq: state.seq}
		if !at.Before(end) {
			r.tail = append(r.tail, update)
			continue
		}
		if err := emitBook(update); err != nil {
			return fmt.Errorf("book line %d: %w", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if state.bids == nil {
		return fmt.Errorf("missing orderbook snapshot in %s", books)
	}
	for !trade.At.IsZero() {
		if err := emitTrade(); err != nil {
			return err
		}
	}
	return nil
}

func (r *SpotArchiveReplay) expire(key model.Key, at time.Time, emit func(model.Event)) {
	if !r.stale && !r.lastBook.at.IsZero() && at.Sub(r.lastBook.at) > archiveFreshness {
		r.stale = true
		emit(model.Event{Key: key, Reset: true})
	}
}

func readArchiveTrade(reader *csv.Reader, tick decimal) (model.Trade, error) {
	fields, err := reader.Read()
	if err != nil {
		return model.Trade{}, err
	}
	stamp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || stamp <= 0 {
		return model.Trade{}, fmt.Errorf("invalid trade timestamp %q", fields[1])
	}
	sell := fields[4] == "sell"
	if !sell && fields[4] != "buy" {
		return model.Trade{}, fmt.Errorf("invalid trade side %q", fields[4])
	}
	price, err := tick.tradeTicks(fields[2], sell)
	if err != nil {
		return model.Trade{}, err
	}
	qty, err := nonnegative(fields[3])
	if err != nil || qty <= 0 {
		return model.Trade{}, fmt.Errorf("invalid trade volume %q", fields[3])
	}
	return model.Trade{ID: fields[0], At: time.UnixMilli(stamp), Ticks: price, Qty: qty, Sell: sell}, nil
}
