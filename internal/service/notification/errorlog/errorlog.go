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
	// dedupWindow — сколько молчим о повторе того же класса ошибки.
	dedupWindow = 5 * time.Minute
	// limitWindow и maxPerWindow — потолок отправок за окно. Telegram режет
	// группу на ~20 сообщениях в минуту, а туда же пишут стратегии, новости и
	// отчёты портфеля: половина лимита остаётся им.
	limitWindow  = time.Minute
	maxPerWindow = 10
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

	// Состояние тротлинга. Живёт целиком внутри Run — единственной горутины,
	// которая его трогает.
	lastSent     map[string]time.Time
	suppressed   map[string]int
	windowStart  time.Time
	sentInWindow int
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
		lastSent:    make(map[string]time.Time),
		suppressed:  make(map[string]int),
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

// handle решает судьбу одного события: дедуп по классу, затем общий лимит.
// Ключ дедупа — только текст сообщения: в ретрай-цикле атрибуты меняются
// (тикер, id заявки, текст gRPC-ошибки), и ключ с ними не поймал бы дубли.
func (s *Sink) handle(event logger.ErrorEvent) {
	now := s.now()
	s.rollWindow(now)

	key := event.Message
	if last, ok := s.lastSent[key]; ok && now.Sub(last) < dedupWindow {
		s.suppressed[key]++

		return
	}
	if s.sentInWindow >= maxPerWindow {
		s.suppressed[key]++

		return
	}

	s.lastSent[key] = now
	s.sentInWindow++
	s.send(formatEvent(event, s.appEnv, s.loc))
}

// rollWindow сбрасывает счётчик отправок при переходе в новое окно.
func (s *Sink) rollWindow(now time.Time) {
	if s.windowStart.IsZero() || now.Sub(s.windowStart) >= limitWindow {
		s.windowStart = now
		s.sentInWindow = 0
	}
}

// flushSummary отправляет сводку подавленного и обнуляет счётчики. Сводка не
// расходует лимит окна: она одна за период и обязана дойти — иначе тишина в
// чате означала бы «ошибок нет» там, где они просто подавлены.
func (s *Sink) flushSummary() {
	dropped := int(s.dropped.Swap(0))
	classes := sortedClasses(s.suppressed)
	s.suppressed = make(map[string]int)

	if msg := formatSummary(classes, dropped); msg != "" {
		s.send(msg)
	}
}

// send пишет в Telegram. Сбой уходит прямо в stderr — см. комментарий к пакету.
func (s *Sink) send(msg string) {
	if err := s.tg.SendMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "errorlog: отправка в telegram не удалась: %v\n", err)
	}
}
