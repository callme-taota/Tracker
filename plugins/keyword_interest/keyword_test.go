package keyword_interest

import (
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

func TestKeyword_Match(t *testing.T) {
	p := New().(*Plugin)
	in := &model.Item{Title: "AI news", Content: "about open source and tech"}
	cfg := plugin.Config{"keywords": []interface{}{"AI", "tech", "open source"}}
	out, err := p.Execute(in, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("want non-nil")
	}
	if len(out.MatchedTopics) == 0 {
		t.Error("expected some matched topics")
	}
	if out.InterestScore <= 0 {
		t.Error("expected positive interest score")
	}
}

func TestKeyword_NoMatch(t *testing.T) {
	p := New().(*Plugin)
	in := &model.Item{Title: "weather", Content: "sunny day"}
	cfg := plugin.Config{"keywords": []interface{}{"AI"}}
	out, err := p.Execute(in, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil && len(out.MatchedTopics) != 0 {
		t.Errorf("expected no match got %v", out.MatchedTopics)
	}
}
