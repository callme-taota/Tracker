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
