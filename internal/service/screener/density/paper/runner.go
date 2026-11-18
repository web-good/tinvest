package paper

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"tinvest/internal/service/screener/density"
)

type Runner struct {
	AllowEntry      func(density.Instrument, density.Signal) bool
	HistorySource   density.UniverseSource
	HistoryInterval time.Duration
	profileUpdates  <-chan []density.Instrument
	Source          density.StreamSource
	Universe        []density.Instrument
	Engine          *Engine
	Detector        density.Config
	Report          func(Summary) error
	ReportInterval  time.Duration
	RetryDelay      time.Duration
}
type batch struct {
	events []density.Event
	err    error
}

func (r *Runner) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	limit := 10
	if r.Engine != nil && r.Engine.cfg.BookExecution {
		limit = 80
	}
	if r.Source == nil || r.Engine == nil || len(r.Universe) == 0 || len(r.Universe) > limit {
		return fmt.Errorf("paper runner requires a supported universe and market data source")
	}
	if r.ReportInterval <= 0 {
		r.ReportInterval = 10 * time.Second
	}
	if r.RetryDelay <= 0 {
		r.RetryDelay = time.Second
	}
	delay := r.RetryDelay
	r.profileUpdates = density.StreamProfiles(ctx, r.HistorySource, r.Detector, r.HistoryInterval, r.Universe, nil)
	for {
		err := r.connection(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if r.Engine.Err() != nil {
			return r.Engine.Err()
		}
		// Report failures are fatal: never continue silently with an unwritable report.
		if _, ok := err.(reportError); ok {
			return err
		}
		r.Engine.Gap("stream_disconnected", time.Now())
		if r.Engine.Err() != nil {
			return r.Engine.Err()
		}
		slog.WarnContext(ctx, "paper market stream disconnected; reconnecting", "error", err, "delay", delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay = min(delay*2, 30*time.Second)
	}
}

type reportError struct{ error }

func (r *Runner) connection(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	lots := make(map[string]int64)
	ids := make([]string, 0, len(r.Universe))
	for _, i := range r.Universe {
		lots[i.ID] = i.Lot
		ids = append(ids, i.ID)
	}
	stream, err := r.Source.OpenStream(ctx, lots)
	if err != nil {
		return err
	}
	defer func() {
		if err := stream.Close(); err != nil {
			slog.Debug("paper stream close", "error", err)
		}
	}()
	if err = stream.SubscribeBooks(ids, r.Detector.BookDepth, true); err != nil {
		return err
	}
	if err = stream.SubscribeInfo(ids, true); err != nil {
		return err
	}
	if r.Engine.cfg.PinbarEntries {
		if err = density.SubscribeClosedCandles(stream, ids); err != nil {
			return err
		}
	}
	if !r.Engine.cfg.BookExecution {
		if err = stream.SubscribeTrades(ids, true); err != nil {
			return err
		}
	}
	received := make(chan batch, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			events, err := stream.Receive()
			select {
			case received <- batch{events, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	defer func() { cancel(); <-done }()
	bookAck := map[string]bool{}
	infoAck := map[string]bool{}
	tradeAck := map[string]time.Time{}
	open := map[string]bool{}
	candles := map[string]bool{}
	detector := density.NewDetector(r.Detector)
	detector.SetInstruments(r.Universe)
	ready := func(id string, now time.Time) bool {
		at := tradeAck[id]
		return bookAck[id] && infoAck[id] && open[id] && (r.Engine.cfg.BookExecution || (!at.IsZero() && now.Sub(at) >= r.Detector.TradeReadiness))
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	report := time.NewTicker(r.ReportInterval)
	defer report.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case profiles, ok := <-r.profileUpdates:
			if !ok {
				r.profileUpdates = nil
				continue
			}
			// Keep the subscription universe, including held inventory, unchanged.
			for n := range r.Universe {
				r.Universe[n].ProfileAt = time.Time{}
				for _, i := range profiles {
					if i.ID == r.Universe[n].ID {
						r.Universe[n] = i
						break
					}
				}
			}
			detector.SetInstruments(profiles)
			clear(r.Engine.pinbars)
		case now := <-tick.C:
			r.Engine.Tick(now)
			if r.Engine.Err() != nil {
				return r.Engine.Err()
			}
		case now := <-report.C:
			if r.Report != nil {
				if err := r.Report(r.Engine.Summary(now)); err != nil {
					return reportError{err}
				}
			}
		case got := <-received:
			if got.err != nil {
				return got.err
			}
			for _, event := range got.events {
				now := time.Now()
				id := event.InstrumentID
				if _, ok := lots[id]; !ok {
					continue
				}
				switch event.Kind {
				case density.EventBookAck:
					if !event.Success {
						return fmt.Errorf("book subscription rejected: %s", id)
					}
					bookAck[id] = true
				case density.EventInfoAck:
					if !event.Success {
						return fmt.Errorf("info subscription rejected: %s", id)
					}
					infoAck[id] = true
				case density.EventTradeAck:
					if !event.Success {
						return fmt.Errorf("trade subscription rejected: %s", id)
					}
					tradeAck[id] = now
				case density.EventCandleAck:
					if !event.Success {
						return fmt.Errorf("candle subscription rejected: %s", id)
					}
					candles[id] = true
				case density.EventCandle:
					if candles[id] && ready(id, now) {
						r.Engine.candle(id, event.Candle, now)
					}
				case density.EventTradingStatus:
					open[id] = event.TradingOpen
					if !event.TradingOpen {
						detector.Reset(id)
						r.Engine.invalidate(id, "trading_closed", now)
					}
				case density.EventInvalidBook:
					detector.Reset(id)
					r.Engine.invalidate(id, "invalid_book", now)
				case density.EventBook:
					if !ready(id, now) {
						detector.Reset(id)
						continue
					}
					signals := detector.Update(event.Book, now)
					r.Engine.OnBookWithCandidates(event.Book, detector.Candidates(), now)
					var admitted []density.Signal
					for _, s := range signals {
						if r.Engine.cfg.BookExecution && r.AllowEntry != nil && !r.AllowEntry(r.Engine.instruments[id], s) {
							continue
						}
						if r.Engine.cfg.PinbarEntries {
							admitted = append(admitted, s)
						} else {
							r.Engine.Observe(s, now)
						}
					}
					if r.Engine.cfg.PinbarEntries {
						r.Engine.pinbarBook(event.Book, admitted, now)
					}
				case density.EventTrade:
					if ready(id, now) {
						r.Engine.OnTrade(event, now)
					}
				}
				if r.Engine.Err() != nil {
					return r.Engine.Err()
				}
			}
		}
	}
}

// Stop cancels pending virtual entries and preserves held inventory as unresolved.
// It deliberately does not invent a fill using the last book during shutdown.
func (e *Engine) Stop(now time.Time) {
	for _, o := range e.orders {
		if o.State == "pending" {
			e.cancel(o, "shutdown", now)
		}
		if o.OpenLots > 0 {
			o.Uncertain = true
			e.complete = false
			delete(e.books, o.InstrumentID)
			e.emit("position_unresolved", o, now, "shutdown with inventory", 0, 0)
		}
	}
	e.emit("session_end", nil, now, "", 0, 0)
}
