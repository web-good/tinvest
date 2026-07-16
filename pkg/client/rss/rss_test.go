package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Урезанная копия реальной ленты smart-lab (проверена 2026-07-15):
// RFC1123Z-даты, guid, одна запись с битой датой, одна без guid.
const feedFixture = `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
<channel>
<title>Лента всех новостей акций</title>
<link>https://smart-lab.ru/news/</link>
<item>
<title>Минфин повысил план по расходам &amp; дефициту</title>
<link>https://smart-lab.ru/mobile/topic/1111111</link>
<guid>https://smart-lab.ru/mobile/topic/1111111</guid>
<pubDate>Wed, 15 Jul 2026 21:00:22 +0300</pubDate>
</item>
<item>
<title>Запись с битой датой — пропускается</title>
<link>https://smart-lab.ru/mobile/topic/2222222</link>
<guid>https://smart-lab.ru/mobile/topic/2222222</guid>
<pubDate>вчера вечером</pubDate>
</item>
<item>
<title>Запись без guid — фолбэк на link</title>
<link>https://smart-lab.ru/mobile/topic/3333333</link>
<pubDate>Wed, 15 Jul 2026 20:48:16 +0300</pubDate>
</item>
</channel>
</rss>`

func TestParse_ValidFeed(t *testing.T) {
	items, err := parse([]byte(feedFixture))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (запись с битой датой пропущена)", len(items))
	}

	first := items[0]
	if first.Title != "Минфин повысил план по расходам & дефициту" {
		t.Errorf("Title = %q (XML-энтити должны быть раскрыты)", first.Title)
	}
	if first.GUID != "https://smart-lab.ru/mobile/topic/1111111" {
		t.Errorf("GUID = %q", first.GUID)
	}
	wantPub := time.Date(2026, 7, 15, 21, 0, 22, 0, time.FixedZone("", 3*3600))
	if !first.PubDate.Equal(wantPub) {
		t.Errorf("PubDate = %v, want %v", first.PubDate, wantPub)
	}

	if items[1].GUID != "https://smart-lab.ru/mobile/topic/3333333" {
		t.Errorf("пустой guid должен фолбэчиться на link, got %q", items[1].GUID)
	}
}

func TestParse_EmptyFeed(t *testing.T) {
	empty := `<?xml version="1.0"?><rss version="2.0"><channel><title>x</title></channel></rss>`
	items, err := parse([]byte(empty))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestParse_BrokenXML(t *testing.T) {
	if _, err := parse([]byte("это не xml")); err == nil {
		t.Fatal("ожидалась ошибка на невалидном XML")
	}
}

func TestFetch_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua == "" || ua == "Go-http-client/1.1" {
			t.Errorf("нужен браузерный User-Agent, got %q", ua)
		}
		_, _ = w.Write([]byte(feedFixture))
	}))
	defer srv.Close()

	items, err := NewClient(srv.URL).Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func TestFetch_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if _, err := NewClient(srv.URL).Fetch(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка на статус 403")
	}
}
