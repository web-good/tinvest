// Package errorlog дублирует ERROR-логи приложения в тему Telegram «General».
//
// Пакет намеренно не использует pkg/logger: собственные сбои пишутся в stderr.
// Иначе ошибка отправки в Telegram логировалась бы как ERROR, возвращалась в
// этот же sink и порождала бесконечный цикл отправок.
package errorlog

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

const (
	// eventBufferSize — глубина очереди событий. Переживает всплеск ретраев и
	// при этом не даёт очереди расти неограниченно.
	eventBufferSize = 256
	// summaryPeriod — период отправки сводки подавленного.
	summaryPeriod = time.Minute
)

// Sink принимает ERROR-записи логгера и доставляет их в Telegram.
// Реализует logger.ErrorSink.
type Sink struct {
	tg     telegram.Client
	appEnv string
	loc    *time.Location
	// now — источник времени для дедупа и лимита; подменяется в тестах.
	now func() time.Time

	events  chan logger.ErrorEvent
	dropped atomic.Int64

	ticker      *time.Ticker
	summaryTick <-chan time.Time
}

// New собирает sink поверх готового Client, уже привязанного к нужному чату и теме.
func New(tg telegram.Client, appEnv string) *Sink {
	// Ошибка загрузки зоны не должна ронять приложение: без МСК время в
	// сообщении будет в UTC, что хуже, но не смертельно.
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	ticker := time.NewTicker(summaryPeriod)

	return &Sink{
		tg:          tg,
		appEnv:      appEnv,
		loc:         loc,
		now:         time.Now,
		events:      make(chan logger.ErrorEvent, eventBufferSize),
		ticker:      ticker,
		summaryTick: ticker.C,
	}
}

// Publish кладёт событие в очередь и немедленно возвращается. Переполненная
// очередь означает потерю события для Telegram (в stdout оно уже есть);
// счётчик попадёт в ближайшую сводку.
func (s *Sink) Publish(event logger.ErrorEvent) {
	select {
	case s.events <- event:
	default:
		s.dropped.Add(1)
	}
}

// Run крутит единственную горутину доставки. Последовательная отправка
// сохраняет порядок сообщений в теме.
func (s *Sink) Run(ctx context.Context) {
	defer s.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.events:
			s.handle(event)
		case <-s.summaryTick:
			s.flushSummary()
		}
	}
}

// handle решает судьбу одного события.
func (s *Sink) handle(event logger.ErrorEvent) {
	s.send(formatEvent(event, s.appEnv, s.loc))
}

// flushSummary отправляет сводку подавленного за период.
func (s *Sink) flushSummary() {
	dropped := int(s.dropped.Swap(0))
	if msg := formatSummary(nil, dropped); msg != "" {
		s.send(msg)
	}
}

// send пишет в Telegram. Сбой уходит прямо в stderr — см. комментарий к пакету.
func (s *Sink) send(msg string) {
	if err := s.tg.SendMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "errorlog: отправка в telegram не удалась: %v\n", err)
	}
}
