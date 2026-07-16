// Package news раз в запуск публикует дайджест свежих записей RSS-ленты
// новостей фондового рынка в Telegram-тему «Новости».
// Дизайн: docs/superpowers/specs/2026-07-15-news-digest-design.md.
package news

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"tinvest/pkg/client/rss"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// startupLookback — окно первого запуска после старта процесса: постим только
// новости за последний час, а не всю ленту.
const startupLookback = time.Hour

// futureSkewSlack — терпим небольшой перекос часов источника, но будущая
// дата не должна глушить окно навсегда: advance() не даёт lastSeen уйти
// дальше этого зазора относительно текущего времени.
const futureSkewSlack = 15 * time.Minute

// Service хранит состояние дедупликации в памяти процесса: после рестарта
// возможен редкий повтор записей на границе окна — принято осознанно,
// хранилище ради этого не заводим.
type Service struct {
	fetcher rss.Fetcher
	tg      telegram.Client
	now     func() time.Time // подменяется в тестах

	// mu сериализует Run: cron не гарантирует отсутствие перекрытия тиков.
	mu sync.Mutex

	// lastSeen — максимальный PubDate уже отправленных записей;
	// boundaryGUIDs — GUID отправленных записей с PubDate == lastSeen
	// (отсекают дубли ровно на границе окна).
	lastSeen      time.Time
	boundaryGUIDs map[string]struct{}
}

// NewService строит сервис. tg должен быть уже привязан к теме «Новости»
// (telegram.NewTopicSender).
func NewService(fetcher rss.Fetcher, tg telegram.Client) *Service {
	return &Service{
		fetcher:       fetcher,
		tg:            tg,
		now:           time.Now,
		boundaryGUIDs: map[string]struct{}{},
	}
}

// Run — одна итерация: fetch → отбор новых записей → отправка дайджеста →
// сдвиг окна. При любой ошибке окно не сдвигается: следующий запуск повторит
// невышедшие записи. Это касается и ошибки на k-м сообщении
// многосообщенческого дайджеста: уже доставленные сообщения будут повторены
// в следующем запуске (следствие правила «окно не сдвигается при ошибке»;
// дубли предпочтительнее потерь).
func (s *Service) Run(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.fetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("news: fetch feed: %w", err)
	}

	if s.lastSeen.IsZero() {
		s.lastSeen = s.now().Add(-startupLookback)
	}

	fresh := s.selectNew(items)
	if len(fresh) == 0 {
		logger.InfoContext(ctx, "news: новых записей нет")

		return nil
	}

	// RSS отдаёт новые сверху — в дайджесте порядок хронологический.
	// SliceStable: записи с равным PubDate сохраняют порядок ленты.
	sort.SliceStable(fresh, func(i, j int) bool { return fresh[i].PubDate.Before(fresh[j].PubDate) })

	for _, msg := range formatDigest(fresh) {
		if err := s.tg.SendMessage(msg); err != nil {
			return fmt.Errorf("news: send digest: %w", err)
		}
	}

	s.advance(fresh)
	logger.InfoContext(ctx, "news: дайджест отправлен", slog.Int("items", len(fresh)))

	return nil
}

// selectNew отбирает записи новее lastSeen (плюс граница по boundaryGUIDs).
// seen отсекает дубли GUID внутри одного ответа Fetch (лента отдавала один
// и тот же элемент дважды на практике) — без него дайджест мог бы получить
// одну запись несколько раз в одном сообщении.
func (s *Service) selectNew(items []rss.Item) []rss.Item {
	var fresh []rss.Item
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if _, dup := seen[it.GUID]; dup {
			continue
		}
		seen[it.GUID] = struct{}{}

		switch {
		case it.PubDate.After(s.lastSeen):
			fresh = append(fresh, it)
		case it.PubDate.Equal(s.lastSeen):
			if _, sent := s.boundaryGUIDs[it.GUID]; !sent {
				fresh = append(fresh, it)
			}
		}
	}

	return fresh
}

// advance сдвигает окно на максимальный PubDate отправленных записей (с
// клампом от будущих дат, см. futureSkewSlack) и перезаполняет множество
// GUID на новой границе.
func (s *Service) advance(sent []rss.Item) {
	maxPub := s.lastSeen
	for _, it := range sent {
		if it.PubDate.After(maxPub) {
			maxPub = it.PubDate
		}
	}
	if clamp := s.now().Add(futureSkewSlack); maxPub.After(clamp) {
		maxPub = clamp
	}
	if maxPub.After(s.lastSeen) {
		s.lastSeen = maxPub
		s.boundaryGUIDs = map[string]struct{}{}
	}
	for _, it := range sent {
		if it.PubDate.Equal(s.lastSeen) {
			s.boundaryGUIDs[it.GUID] = struct{}{}
		}
	}
}
