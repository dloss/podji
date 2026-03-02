package data

import (
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestMockStoreSetScopeUpdatesRegistryNamespace(t *testing.T) {
	store := NewMockStore()
	store.SetScope(Scope{Context: "prod-us-east-1", Namespace: "staging"})

	if got := store.Scope().Context; got != "prod-us-east-1" {
		t.Fatalf("expected context prod-us-east-1, got %q", got)
	}
	if got := store.Scope().Namespace; got != "staging" {
		t.Fatalf("expected namespace staging, got %q", got)
	}
	if got := store.Registry().Namespace(); got != "staging" {
		t.Fatalf("expected registry namespace staging, got %q", got)
	}
}

func TestMockStoreNamespaceFallback(t *testing.T) {
	store := NewMockStore()
	store.SetScope(Scope{Context: "default", Namespace: ""})
	if got := store.Scope().Namespace; got != resources.DefaultNamespace {
		t.Fatalf("expected default namespace fallback, got %q", got)
	}
}
