package pluginhub

import "testing"

func TestFormatsCompatible(t *testing.T) {
	if !FormatsCompatible([]string{"tracker.item.v1"}, []string{"tracker.item.v1"}) {
		t.Fatal("same format should match")
	}
	if !FormatsCompatible([]string{"a"}, []string{"*"}) {
		t.Fatal("wildcard input should accept")
	}
	if !FormatsCompatible(nil, []string{"tracker.item.v1"}) {
		t.Fatal("empty output should be permissive")
	}
	if FormatsCompatible([]string{"a"}, []string{"b"}) {
		t.Fatal("mismatch should fail")
	}
}

func TestEdgeFormatsCompatible(t *testing.T) {
	if !EdgeFormatsCompatible([]string{"tracker.item.v1"}, []string{"tracker.item.v1"}) {
		t.Fatal("same format should match")
	}
	if !EdgeFormatsCompatible([]string{"a"}, []string{"*"}) {
		t.Fatal("wildcard input should accept")
	}
	if EdgeFormatsCompatible(nil, []string{"tracker.item.v1"}) {
		t.Fatal("empty output with constrained input should fail")
	}
	if EdgeFormatsCompatible([]string{"a"}, []string{"b"}) {
		t.Fatal("mismatch should fail")
	}
	if !EdgeFormatsCompatible([]string{"x"}, []string{}) {
		t.Fatal("empty input constraint should pass")
	}
}
