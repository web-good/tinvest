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
// невышедшие записи.
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
	sort.Slice(fresh, func(i, j int) bool { return fresh[i].PubDate.Before(fresh[j].PubDate) })

	for _, msg := range formatDigest(fresh) {
		if err := s.tg.SendMessage(msg); err != nil {
			return fmt.Errorf("news: send digest: %w", err)
		}
	}

	s.advance(fresh)
	logger.InfoContext(ctx, "news: дайджест отправлен", slog.Int("items", len(fresh)))

	return nil
}

func (s *Service) selectNew(items []rss.Item) []rss.Item {
	var fresh []rss.Item
	for _, it := range items {
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

// advance сдвигает окно на максимальный PubDate отправленных записей и
// перезаполняет множество GUID на новой границе.
func (s *Service) advance(sent []rss.Item) {
	maxPub := s.lastSeen
	for _, it := range sent {
		if it.PubDate.After(maxPub) {
			maxPub = it.PubDate
		}
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
