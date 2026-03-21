package format

import "testing"

func TestNormalizeMarkdown(t *testing.T) {
	s := NormalizeMarkdown("  a\r\n\r\n\r\nb  ")
	if s != "a\n\nb" {
		t.Fatalf("got %q", s)
	}
}
