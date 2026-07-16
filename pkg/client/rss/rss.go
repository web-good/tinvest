// Package rss загружает и разбирает RSS 2.0-ленты (encoding/xml, без внешних
// зависимостей). Используется сервисом новостного дайджеста.
package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Item — одна запись ленты. GUID уникален в пределах ленты (при пустом guid
// в XML подставляется link).
type Item struct {
	Title   string
	Link    string
	GUID    string
	PubDate time.Time
}

// Fetcher загружает и разбирает RSS-ленту.
type Fetcher interface {
	Fetch(ctx context.Context) ([]Item, error)
}

const (
	fetchTimeout = 30 * time.Second
	// maxFeedSize ограничивает читаемое тело ответа: лента smart-lab ~200 КБ,
	// 5 МБ — защита от бесконечного/огромного ответа.
	maxFeedSize = 5 << 20
)

// Client — HTTP-реализация Fetcher для одного URL.
type Client struct {
	url  string
	http *http.Client
}

func NewClient(url string) *Client {
	return &Client{url: url, http: &http.Client{Timeout: fetchTimeout}}
}

func (c *Client) Fetch(ctx context.Context) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("rss: build request: %w", err)
	}
	// Без браузерного User-Agent часть лент (smart-lab) отвечает 403.
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss: fetch %s: %w", c.url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss: %s returned status %d", c.url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedSize))
	if err != nil {
		return nil, fmt.Errorf("rss: read body: %w", err)
	}

	return parse(body)
}

type rssDoc struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

func parse(data []byte) ([]Item, error) {
	var doc rssDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("rss: parse feed: %w", err)
	}

	items := make([]Item, 0, len(doc.Channel.Items))
	for _, it := range doc.Channel.Items {
		pub, err := parseDate(it.PubDate)
		if err != nil {
			// Запись без валидной даты не может участвовать в окне
			// дедупликации — пропускаем её целиком.
			continue
		}
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		items = append(items, Item{Title: it.Title, Link: it.Link, GUID: guid, PubDate: pub})
	}

	return items, nil
}

func parseDate(s string) (time.Time, error) {
	// smart-lab/Finam/MOEX отдают RFC1123Z; RFC1123 — на всякий случай.
	for _, layout := range []string{time.RFC1123Z, time.RFC1123} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("rss: unsupported pubDate %q", s)
}
