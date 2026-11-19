package live

import (
	"context"
	"errors"
	"fmt"
	"time"
	"tinvest/internal/service/screener/density"
)

type Runner struct {
	BookOnly        bool
	AllowEntry      func(density.Instrument, density.Signal) bool
	HistorySource   density.UniverseSource
	HistoryInterval time.Duration
	profileUpdates  <-chan []density.Instrument
	Source          density.StreamSource
	Universe        []density.Instrument
	Detector        density.Config
	Feed            *Feed
	Engine          *Engine
	Report          func(State) error
	Warn            func(error)
	Notify          func(context.Context, Notification) error
}
type receivedBatch struct {
	events []density.Event
	err    error
}

func (r *Runner) Run(parent context.Context) error {
	limit := 10
	if r.BookOnly {
		limit = 80
	}
	if r.Source == nil || r.Feed == nil || r.Engine == nil || len(r.Universe) < 1 || len(r.Universe) > limit {
		return fmt.Errorf("live runner requires a supported universe and market/execution dependencies")
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); r.market(ctx) }()
	defer func() { cancel(); <-done }()
	batches := make(chan []Notification, 1)
	acks := make(chan string, 1)
	notificationDone := make(chan struct{})
	go func() {
		defer close(notificationDone)
		RunNotificationDelivery(ctx, batches, acks, r.Notify, r.Warn)
	}()
	defer func() { cancel(); <-notificationDone }()
	queueNotifications := func() {
		if r.Notify == nil {
			return
		}
		select {
		case batches <- r.Engine.PendingNotifications():
		default:
		}
	}
	queueNotifications()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	report := time.NewTicker(10 * time.Second)
	defer report.Stop()
	var lastWarning string
	var warnedAt time.Time
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case id := <-acks:
			if err := r.Engine.AcknowledgeNotification(id); err != nil {
				return err
			}
		case now := <-tick.C:
			if err := r.Engine.Step(ctx, now); err != nil {
				if r.Engine.Err() != nil {
					return err
				}
				if r.Warn != nil && (err.Error() != lastWarning || now.Sub(warnedAt) >= 10*time.Second) {
					r.Warn(err)
					lastWarning = err.Error()
					warnedAt = now
				}
			}
			queueNotifications()
		case <-report.C:
			if r.Report != nil {
				if err := r.Report(r.Engine.State()); err != nil {
					return err
				}
			}
		}
	}
}
func (r *Runner) market(ctx context.Context) {
	r.profileUpdates = density.StreamProfiles(ctx, r.HistorySource, r.Detector, r.HistoryInterval, r.Universe, r.Warn)
	delay := time.Second
	for {
		err := r.connection(ctx)
		if ctx.Err() != nil {
			return
		}
		r.Feed.Gap(time.Now())
		if r.Warn != nil {
			r.Warn(fmt.Errorf("market stream reconnecting: %w", err))
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		delay = min(delay*2, 30*time.Second)
	}
}

func (r *Runner) connection(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	lots := map[string]int64{}
	instruments := map[string]density.Instrument{}
	var ids []string
	for _, i := range r.Universe {
		lots[i.ID] = i.Lot
		instruments[i.ID] = i
		ids = append(ids, i.ID)
	}
	stream, err := r.Source.OpenStream(ctx, lots)
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()
	if err = stream.SubscribeBooks(ids, r.Detector.BookDepth, true); err != nil {
		return err
	}
	if err = stream.SubscribeInfo(ids, true); err != nil {
		return err
	}
	if r.Feed.cfg.CandleEntries {
		if err = density.SubscribeClosedCandles(stream, ids); err != nil {
			return err
		}
	}
	if !r.BookOnly {
		if err = stream.SubscribeTrades(ids, true); err != nil {
			return err
		}
	}
	received := make(chan receivedBatch, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			events, err := stream.Receive()
			select {
			case received <- receivedBatch{events, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	defer func() { cancel(); _ = stream.Close(); <-done }()
	books, info, open := map[string]bool{}, map[string]bool{}, map[string]bool{}
	trades := map[string]time.Time{}
	candles := map[string]bool{}
	detector := density.NewDetector(r.Detector)
	detector.SetInstruments(r.Universe)
	ready := func(id string, now time.Time) bool {
		return books[id] && info[id] && open[id] && (r.BookOnly || (!trades[id].IsZero() && now.Sub(trades[id]) >= r.Detector.TradeReadiness))
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case profiles, ok := <-r.profileUpdates:
			if !ok {
				r.profileUpdates = nil
				continue
			}
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
			r.Feed.ClearSignals()
		case batch := <-received:
			if batch.err != nil {
				return batch.err
			}
			for _, event := range batch.events {
				now, id := time.Now(), event.InstrumentID
				lot, known := lots[id]
				if !known {
					continue
				}
				switch event.Kind {
				case density.EventBookAck, density.EventInfoAck, density.EventTradeAck, density.EventCandleAck:
					if !event.Success {
						return fmt.Errorf("market subscription rejected")
					}
					switch event.Kind {
					case density.EventBookAck:
						books[id] = true
					case density.EventInfoAck:
						info[id] = true
					case density.EventTradeAck:
						trades[id] = now
					case density.EventCandleAck:
						candles[id] = true
					}
				case density.EventTradingStatus:
					open[id] = event.TradingOpen
					if !event.TradingOpen {
						r.Feed.SetReady(id, false)
						detector.Reset(id)
					}
				case density.EventInvalidBook:
					r.Feed.SetReady(id, false)
					detector.Reset(id)
				case density.EventBook:
					if event.Book.InstrumentID != id || event.Book.Lot != lot {
						r.Feed.SetReady(id, false)
						detector.Reset(id)
						continue
					}
					isReady := ready(id, now)
					r.Feed.SetReady(id, isReady)
					var signals []density.Signal
					if isReady {
						signals = detector.Update(event.Book, now)
					} else {
						detector.Reset(id)
					}
					if r.BookOnly && r.AllowEntry != nil {
						var admitted []density.Signal
						for _, signal := range signals {
							if r.AllowEntry(instruments[id], signal) {
								admitted = append(admitted, signal)
							}
						}
						signals = admitted
					}
					r.Feed.bookWithCandidates(event.Book, signals, detector.Candidates(), instruments[id].PriceStep, now)
				case density.EventCandle:
					if candles[id] && ready(id, now) {
						r.Feed.candle(id, event.Candle, now)
					}
				case density.EventTrade:
					if ready(id, now) {
						r.Feed.Trade(event, now)
					}
				}
			}
		}
	}
}

// Normalize only the lifecycle context error, never errors joined from journal/cleanup.
func FinishError(runErr, cleanupErr error) error {
	if runErr == context.Canceled || runErr == context.DeadlineExceeded {
		runErr = nil
	}
	return errors.Join(runErr, cleanupErr)
}
