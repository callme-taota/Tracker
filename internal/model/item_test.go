package model

import (
	"testing"
)

func TestItemError_Error(t *testing.T) {
	e := &ItemError{Code: "x", Message: "y"}
	if e.Error() != "x: y" {
		t.Errorf("Error() want 'x: y' got %q", e.Error())
	}
	e2 := &ItemError{Message: "only"}
	if e2.Error() != "only" {
		t.Errorf("Error() want 'only' got %q", e2.Error())
	}
}
