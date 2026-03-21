package llm

import (
	"context"
	"testing"
)

func TestNewRouterFromEnv_NoKey(t *testing.T) {
	r := NewRouterFromEnv()
	if len(r.ProfileNames()) != 0 {
		t.Fatalf("expected no profiles without OPENAI_API_KEY")
	}
	_, err := r.Chat(context.Background(), "default", []Message{{Role: "user", Content: "hi"}}, 0)
	if err == nil {
		t.Fatal("expected error without profiles")
	}
}

func TestRegisterProfile(t *testing.T) {
	r := NewRouterFromEnv()
	r.RegisterProfile(Profile{Name: "test", BaseURL: "http://invalid.local", Model: "x", APIKey: "k"})
	if len(r.ProfileNames()) != 1 {
		t.Fatalf("want 1 profile")
	}
}
