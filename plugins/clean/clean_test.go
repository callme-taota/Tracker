package clean

import (
	"testing"

	"Tracker/internal/model"
	"Tracker/internal/plugin"
)

func TestClean_StripHTML(t *testing.T) {
	p := New().(*Plugin)
	in := &model.Item{Content: "<p>hello</p> world <br/>"}
	out, err := p.Execute(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil {
		t.Fatal("want non-nil item")
	}
	if out.Content != "hello world" {
		t.Errorf("want 'hello world' got %q", out.Content)
	}
}

func TestClean_NilItem(t *testing.T) {
	p := New().(*Plugin)
	out, err := p.Execute(nil, plugin.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Error("want nil out for nil in")
	}
}
