package dedup_basic

import (
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

func TestDedupBasic_DropsDuplicate(t *testing.T) {
	pl := New()
	p, ok := pl.(*Plugin)
	if !ok {
		t.Fatal("expected *Plugin")
	}
	cfg := plugin.Config{}
	it := &model.Item{Title: "A", URL: "https://ex.com/x", Content: "body"}
	o1, err := p.Execute(it, cfg)
	if err != nil || o1 == nil {
		t.Fatalf("first pass: %v %v", o1, err)
	}
	o2, err := p.Execute(it, cfg)
	if err != nil || o2 != nil {
		t.Fatalf("second should drop: %#v err=%v", o2, err)
	}
}
