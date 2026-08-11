package errorlog

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"tinvest/pkg/logger"
)

// fakeClock — управляемые часы: воркер читает время только через now().
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{t: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestDuplicateWithinWindowIsSuppressed(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, _ := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	event := logger.ErrorEvent{Time: clock.Now(), Message: "не удалось получить свечи"}
	sink.Publish(event)
	sender.waitSent(t, 1)

	sink.Publish(event)
	sink.Publish(event)
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != 1 {
		t.Fatalf("отправок %d, ожидалась 1: повтор в окне дедупа должен подавляться", got)
	}

	clock.advance(dedupWindow + time.Second)
	sink.Publish(event)
	sender.waitSent(t, 1)

	if got := len(sender.snapshot()); got != 2 {
		t.Fatalf("отправок %d, ожидалось 2: после окна дедупа сообщение должно пройти", got)
	}
}

func TestPerWindowLimit(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, _ := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	// Сообщения различаются, поэтому дедуп не срабатывает — режет только лимит.
	for i := 0; i < maxPerWindow; i++ {
		sink.Publish(logger.ErrorEvent{Time: clock.Now(), Message: "ошибка " + string(rune('A'+i))})
	}
	sender.waitSent(t, maxPerWindow)

	sink.Publish(logger.ErrorEvent{Time: clock.Now(), Message: "ошибка сверх лимита"})
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != maxPerWindow {
		t.Fatalf("отправок %d, ожидалось %d: сообщение сверх лимита должно подавляться", got, maxPerWindow)
	}

	clock.advance(limitWindow + time.Second)
	sink.Publish(logger.ErrorEvent{Time: clock.Now(), Message: "ошибка в новом окне"})
	sender.waitSent(t, 1)

	if got := len(sender.snapshot()); got != maxPerWindow+1 {
		t.Fatalf("отправок %d, ожидалось %d: в новом окне лимит должен обнулиться", got, maxPerWindow+1)
	}
}

func TestSummaryReportsSuppressedAndDropped(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, tick := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	first := logger.ErrorEvent{Time: clock.Now(), Message: "не удалось получить свечи"}
	second := logger.ErrorEvent{Time: clock.Now(), Message: "сбой чтения стейта"}
	sink.Publish(first)
	sink.Publish(second)
	sender.waitSent(t, 2)

	// По три повтора каждого — все шесть подавляются дедупом.
	for i := 0; i < 3; i++ {
		sink.Publish(first)
		sink.Publish(second)
	}
	time.Sleep(200 * time.Millisecond)
	sink.dropped.Store(4)

	tick <- clock.Now()
	sender.waitSent(t, 1)

	msgs := sender.snapshot()
	summary := msgs[len(msgs)-1]
	for _, want := range []string{
		"подавлено 6 ошибок за минуту",
		"3 × не удалось получить свечи",
		"3 × сбой чтения стейта",
		"+ 4 событий не попали в очередь",
	} {
		if !strings.Contains(summary, want) {
			t.Errorf("в сводке нет %q:\n%s", want, summary)
		}
	}

	if got := sink.dropped.Load(); got != 0 {
		t.Errorf("dropped = %d, ожидалось 0: сводка обязана обнулять счётчик", got)
	}
}

func TestSummarySilentWhenNothingSuppressed(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, tick := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	tick <- clock.Now()
	time.Sleep(200 * time.Millisecond)

	if got := sender.snapshot(); len(got) != 0 {
		t.Fatalf("при пустой сводке отправок быть не должно, получено: %v", got)
	}
}

func TestSummaryResetsCountersBetweenPeriods(t *testing.T) {
	sender := newFakeSender()
	clock := newFakeClock(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC))
	sink, tick := newTestSink(t, sender, clock.Now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sink.Run(ctx)

	event := logger.ErrorEvent{Time: clock.Now(), Message: "повторяющаяся ошибка"}
	sink.Publish(event)
	sender.waitSent(t, 1)
	sink.Publish(event)
	time.Sleep(200 * time.Millisecond)

	tick <- clock.Now()
	sender.waitSent(t, 1)

	tick <- clock.Now()
	time.Sleep(200 * time.Millisecond)

	if got := len(sender.snapshot()); got != 2 {
		t.Fatalf("отправок %d, ожидалось 2: вторая сводка должна быть пустой", got)
	}
}
