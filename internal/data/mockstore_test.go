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

func TestMockStoreNamespaceAndContextDiscovery(t *testing.T) {
	store := NewMockStore()

	namespaces := store.NamespaceNames()
	if len(namespaces) == 0 || namespaces[0] != resources.AllNamespaces {
		t.Fatalf("expected all-namespaces sentinel first, got %v", namespaces)
	}

	contexts := store.ContextNames()
	if len(contexts) == 0 {
		t.Fatal("expected at least one context in mock store")
	}
}

func TestMockStoreQueriesFollowScopeNamespace(t *testing.T) {
	store := NewMockStore()

	store.SetScope(Scope{Context: "default", Namespace: "staging"})
	items := store.PodsByRestarts()
	if len(items) == 0 {
		t.Fatal("expected restart query results in staging namespace")
	}
	foundStagingPod := false
	for _, item := range items {
		if item.Name == "worker-55c6c6f9f-t6u8v" {
			foundStagingPod = true
			break
		}
	}
	if !foundStagingPod {
		t.Fatalf("expected staging restart pod in results, got %#v", items)
	}
}
