package news

import (
	"reflect"
	"testing"

	"Tracker/internal/plugin"
)

func TestBuildFeeds_ExplicitFeedsOverridePresets(t *testing.T) {
	feeds := buildFeeds(plugin.Config{
		"feeds": []interface{}{"https://example.com/custom.xml"},
	})
	want := []string{"https://example.com/custom.xml"}
	if !reflect.DeepEqual(feeds, want) {
		t.Fatalf("want %v got %v", want, feeds)
	}
}

func TestBuildFeeds_PresetsAndExtraFeedsDeduplicate(t *testing.T) {
	feeds := buildFeeds(plugin.Config{
		"presets":     []interface{}{"cn"},
		"extra_feeds": []interface{}{"https://www.36kr.com/feed", "https://example.com/extra.xml"},
	})
	count := 0
	for _, feed := range feeds {
		if feed == "https://www.36kr.com/feed" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected deduplicated 36kr feed, got count=%d feeds=%v", count, feeds)
	}
	if feeds[len(feeds)-1] != "https://example.com/extra.xml" {
		t.Fatalf("expected extra feed appended, got %v", feeds)
	}
}
