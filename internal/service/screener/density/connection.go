package density

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type subscription struct {
	kind      EventKind
	ids       []string
	subscribe bool
}
type sendJob struct{ signal Signal }

func enqueueSignal(jobs chan<- sendJob, pending map[key]time.Time, signal Signal) bool {
	select {
	case jobs <- sendJob{signal}:
		pending[key{signal.InstrumentID, signal.Price}] = signal.Since
		return true
	default:
		return false
	}
}

type sendResult struct {
	signal  Signal
	err     error
	skipped bool
}

func applySendResult(result sendResult, now time.Time, pending, delivered, sent, retryAfter map[key]time.Time) bool {
	k := key{result.signal.InstrumentID, result.signal.Price}
	since, ok := pending[k]
	if !ok || !since.Equal(result.signal.Since) {
		return false
	}
	delete(pending, k)
	if result.err != nil && !result.skipped {
		retryAfter[k] = now.Add(5 * time.Second)
	}
	if result.err == nil && !result.skipped {
		sent[k] = now
		delivered[k] = result.signal.Since
	}
	return true
}

type sendCheck struct {
	signal   Signal
	response chan *Signal
}
type streamResult struct {
	events []Event
	err    error
}

func runConnection(parent context.Context, s *Service, universe []Instrument) error {
	ctx, cancel := context.WithCancel(parent)
	lots := make(map[string]int64, len(universe))
	tickers := make(map[string]string, len(universe))
	ids := make([]string, 0, len(universe))
	for _, i := range universe {
		lots[i.ID] = i.Lot
		tickers[i.ID] = i.Ticker
		ids = append(ids, i.ID)
	}
	stream, err := s.stream.OpenStream(ctx, lots)
	if err != nil {
		cancel()
		return err
	}
	var wg sync.WaitGroup

	commands := make(chan subscription, 32)
	received := make(chan streamResult, 32)
	jobs := make(chan sendJob, 32)
	checks := make(chan sendCheck)
	results := make(chan sendResult, 32)
	defer func() {
		cancel()
		wg.Wait()
		_ = stream.Close()
		for {
			select {
			case result := <-results:
				if result.err == nil && !result.skipped {
					s.sent[key{result.signal.InstrumentID, result.signal.Price}] = time.Now()
				}
			default:
				return
			}
		}
	}()

	writeErr := make(chan error, 1)
	wg.Add(3)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case cmd := <-commands:
				var e error
				switch cmd.kind {
				case EventBookAck:
					e = stream.SubscribeBooks(cmd.ids, s.cfg.BookDepth, cmd.subscribe)
				case EventInfoAck:
					e = stream.SubscribeInfo(cmd.ids, cmd.subscribe)
				case EventTradeAck:
					e = stream.SubscribeTrades(cmd.ids, cmd.subscribe)
				}
				if e != nil {
					select {
					case writeErr <- e:
					case <-ctx.Done():
					}
					return
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			events, e := stream.Receive()
			select {
			case received <- streamResult{events, e}:
			case <-ctx.Done():
				return
			}
			if e != nil {
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-jobs:
				reply := make(chan *Signal, 1)
				select {
				case checks <- sendCheck{job.signal, reply}:
				case <-ctx.Done():
					return
				}
				var latest *Signal
				select {
				case latest = <-reply:
				case <-ctx.Done():
					return
				}
				if latest == nil {
					select {
					case results <- sendResult{signal: job.signal, skipped: true}:
					case <-ctx.Done():
					}
					continue
				}
				sendCtx, stop := context.WithTimeout(ctx, 30*time.Second)
				e := s.sender.Send(sendCtx, *latest)
				stop()
				results <- sendResult{signal: job.signal, err: e}
			}
		}
	}()
	commands <- subscription{EventBookAck, ids, true}
	commands <- subscription{EventInfoAck, ids, true}
	detector := newScreenerDetector(s.cfg)
	detector.SetInstruments(universe)
	candidates := NewCandidates(s.cfg.MaxTrades, s.cfg.TradeReadiness)
	open := make(map[string]bool)
	unavailable := make(map[string]bool)
	bookReady := make(map[string]bool)
	infoReady := make(map[string]bool)
	selected := make(map[string]bool)
	tradeAcks := newAckTracker()
	pending := make(map[key]time.Time)
	delivered := make(map[key]time.Time)
	retryAfter := make(map[key]time.Time)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	queue := func(cmd subscription) error {
		select {
		case commands <- cmd:
			return nil
		default:
			return fmt.Errorf("subscription queue full")
		}
	}
	current := func(sig Signal, now time.Time) *Signal {
		detector.Expire(now)
		for _, v := range detector.Candidates() {
			if v.InstrumentID == sig.InstrumentID && v.Price == sig.Price && v.Since.Equal(sig.Since) && now.Sub(v.Since) >= s.cfg.Persistence && open[v.InstrumentID] && bookReady[v.InstrumentID] && infoReady[v.InstrumentID] && candidates.Ready(v.InstrumentID, now) {
				v.Ticker = tickers[v.InstrumentID]
				v.Age = now.Sub(v.Since)
				if n, ok := candidates.Trades(v.InstrumentID, v.Price, now); ok {
					v.TradeLots = n
					v.TradeCoverage = now.Sub(candidates.readySince[v.InstrumentID])
				}
				return &v
			}
		}
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-writeErr:
			return err
		case req := <-checks:
			req.response <- current(req.signal, time.Now())
		case result := <-results:
			if applySendResult(result, time.Now(), pending, delivered, s.sent, retryAfter) && result.err != nil && !result.skipped {
				slog.WarnContext(ctx, "density notification failed", "instrument", result.signal.InstrumentID, "error", result.err)
			}

		case batch := <-received:
			if batch.err != nil {
				return batch.err
			}
			now := time.Now()
			for _, e := range batch.events {
				if _, ok := lots[e.InstrumentID]; !ok || unavailable[e.InstrumentID] {
					continue
				}
				switch e.Kind {
				case EventBookAck:
					if !e.Success {
						slog.WarnContext(ctx, "density book subscription rejected", "instrument", e.InstrumentID)
						unavailable[e.InstrumentID] = true
						detector.Reset(e.InstrumentID)
						continue
					}
					bookReady[e.InstrumentID] = true
				case EventInfoAck:
					if !e.Success {
						slog.WarnContext(ctx, "density info subscription rejected", "instrument", e.InstrumentID)
						unavailable[e.InstrumentID] = true
						detector.Reset(e.InstrumentID)
						continue
					}
					infoReady[e.InstrumentID] = true
				case EventTradeAck:
					subscribed, current := tradeAcks.ack(e.InstrumentID)
					if !current {
						continue
					}
					if !e.Success {
						if !subscribed {
							return fmt.Errorf("trade unsubscribe failed for %s", e.InstrumentID)
						}
						slog.WarnContext(ctx, "density trade subscription rejected", "instrument", e.InstrumentID)
						unavailable[e.InstrumentID] = true
						detector.Reset(e.InstrumentID)
						continue
					}
					if subscribed {
						candidates.Acknowledge(e.InstrumentID, 1, now)
					}
				case EventTradingStatus:
					open[e.InstrumentID] = e.TradingOpen
					if !e.TradingOpen {
						detector.Reset(e.InstrumentID)
					}
				case EventInvalidBook:
					detector.Reset(e.InstrumentID)
				case EventTrade:
					candidates.RecordTrade(e.InstrumentID, e.Price, e.Quantity, 1)
				case EventBook:
					if !open[e.InstrumentID] || !infoReady[e.InstrumentID] || !bookReady[e.InstrumentID] {
						continue
					}
					signals := detector.Update(e.Book, now)
					for _, sig := range signals {
						k := key{sig.InstrumentID, sig.Price}
						if since, ok := delivered[k]; ok && since.Equal(sig.Since) {
							continue
						}
						if _, ok := pending[k]; ok {
							continue
						}
						if retry, ok := retryAfter[k]; ok && now.Before(retry) {
							continue
						}
						if last, ok := s.sent[k]; ok && now.Sub(last) < 30*time.Minute {
							continue
						}
						if !candidates.Ready(sig.InstrumentID, now) {
							continue
						}
						sig.Ticker = tickers[sig.InstrumentID]
						sig.Age = now.Sub(sig.Since)
						if n, ok := candidates.Trades(sig.InstrumentID, sig.Price, now); ok {
							sig.TradeLots = n
							sig.TradeCoverage = now.Sub(candidates.readySince[sig.InstrumentID])
						}
						enqueueSignal(jobs, pending, sig)
					}
				}
			}
		case now := <-ticker.C:
			detector.Expire(now)
			list := candidates.Select(detector.Candidates(), now)
			next := make(map[string]bool, len(list))
			for _, id := range list {
				next[id] = true
			}
			var remove, add []string
			for id := range selected {
				if !next[id] {
					tradeAcks.changed(id, false)
					remove = append(remove, id)
				}
			}
			for _, id := range list {
				if !selected[id] {
					tradeAcks.changed(id, true)
					add = append(add, id)
				}
			}
			if len(remove) > 0 {
				if err := queue(subscription{EventTradeAck, remove, false}); err != nil {
					return err
				}
			}
			if len(add) > 0 {
				if err := queue(subscription{EventTradeAck, add, true}); err != nil {
					return err
				}
			}

			selected = next
		}
	}
}
