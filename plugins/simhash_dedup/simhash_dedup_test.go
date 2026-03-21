package simhash_dedup

import (
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

func TestHamming(t *testing.T) {
	if hamming(0, 0) != 0 {
		t.Fatal()
	}
	if hamming(1, 0) != 1 {
		t.Fatal()
	}
}

func TestSimhashDuplicateDrops(t *testing.T) {
	p := New().(*Plugin)
	cfg := plugin.Config{}
	it := &model.Item{Title: "t", Content: "identical body for simhash"}
	o1, _ := p.Execute(it, cfg)
	if o1 == nil {
		t.Fatal("first item dropped")
	}
	o2, _ := p.Execute(it, cfg)
	if o2 != nil {
		t.Fatal("exact duplicate should drop")
	}
}
