package config

import "testing"

func TestNewNewsConfig_DefaultFeedURL(t *testing.T) {
	cfg := NewNewsConfig()
	if cfg.FeedURL != "https://smart-lab.ru/news/rss/" {
		t.Fatalf("FeedURL = %q, want smart-lab default", cfg.FeedURL)
	}
}
