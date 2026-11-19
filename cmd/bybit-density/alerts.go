package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"
)

type fillRecorder interface {
	Record(model.Fill, string) error
}

type instrumentState struct {
	instrument model.Instrument
	strategy   *strategy.Strategy
}

type signalLevel struct {
	key   model.Key
	side  model.Side
	ticks int64
}

type alertHub struct {
	mu         sync.Mutex
	config     strategy.Config
	recorder   fillRecorder
	states     map[model.Key]instrumentState
	signals    chan model.Signal
	pending    map[signalLevel]struct{}
	retryAfter map[signalLevel]time.Time
}

func newAlertHub(cfg strategy.Config, recorder fillRecorder) *alertHub {
	return &alertHub{
		config: cfg, recorder: recorder,
		states:     make(map[model.Key]instrumentState),
		signals:    make(chan model.Signal, 64),
		pending:    make(map[signalLevel]struct{}),
		retryAfter: make(map[signalLevel]time.Time),
	}
}

func (h *alertHub) add(instruments []model.Instrument) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, value := range instruments {
		h.states[value.Key] = instrumentState{instrument: value, strategy: strategy.New(value, h.config)}
	}
}

func (h *alertHub) apply(event model.Event, now time.Time) error {
	h.mu.Lock()
	state, ok := h.states[event.Key]
	if !ok {
		h.mu.Unlock()
		return nil
	}
	signals, fills := state.strategy.Apply(event, now)
	for _, signal := range signals {
		level := signalLevel{key: signal.Key, side: signal.Side, ticks: signal.Ticks}
		if _, queued := h.pending[level]; queued || now.Before(h.retryAfter[level]) {
			continue
		}
		select {
		case h.signals <- signal:
			h.pending[level] = struct{}{}
		default:
		}
	}
	h.mu.Unlock()

	for _, fill := range fills {
		if err := h.recorder.Record(fill, formatFill(fill, state.instrument.TickSize)); err != nil {
			return err
		}
	}
	return nil
}

func (h *alertHub) reset(instruments []model.Instrument) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for _, instrument := range instruments {
		if state, ok := h.states[instrument.Key]; ok {
			state.strategy.Apply(model.Event{Key: instrument.Key, Reset: true}, now)
		}
	}
}

func (h *alertHub) expire(ctx context.Context) error {
	period := max(10*time.Millisecond, min(time.Second, h.config.Freshness/2))
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			h.mu.Lock()
			for _, state := range h.states {
				state.strategy.Expire(now)
			}
			for level, retryAt := range h.retryAfter {
				if !now.Before(retryAt) {
					delete(h.retryAfter, level)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *alertHub) sendSignals(ctx context.Context, send sendMessage) error {
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return nil
		case signal := <-h.signals:
			h.sendSignal(ctx, signal, send)
		}
	}
	return nil
}

func (h *alertHub) sendSignal(ctx context.Context, signal model.Signal, send sendMessage) {
	level := signalLevel{key: signal.Key, side: signal.Side, ticks: signal.Ticks}
	h.mu.Lock()
	state := h.states[signal.Key]
	current, ok := state.strategy.Current(signal, time.Now())
	if !ok {
		delete(h.pending, level)
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	err := send(ctx, formatSignal(current, state.instrument.TickSize))
	now := time.Now()
	h.mu.Lock()
	delete(h.pending, level)
	if err == nil {
		state.strategy.MarkSent(signal, now)
	} else {
		h.retryAfter[level] = now.Add(5 * time.Second)
	}
	h.mu.Unlock()
	if err != nil && ctx.Err() == nil {
		slog.Warn("Bybit density alert delivery failed", "market", signal.Key.Market, "symbol", signal.Key.Symbol, "error", err)
	}
}
