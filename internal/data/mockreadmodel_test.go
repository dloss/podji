package data

import (
	"context"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestMockReadModelListRespectsScope(t *testing.T) {
	store := NewMockStore()
	read := store.ReadModel()

	items, err := read.List("workloads", Scope{Context: "default", Namespace: "staging"})
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected workload items")
	}
	foundStaging := false
	for _, item := range items {
		if item.Name == "seed-data" {
			foundStaging = true
			break
		}
	}
	if !foundStaging {
		t.Fatalf("expected staging workload in result set, got %#v", items)
	}
}

func TestMockReadModelUnknownResource(t *testing.T) {
	store := NewMockStore()
	_, err := store.ReadModel().List("nope", Scope{Context: "default", Namespace: "default"})
	if err == nil {
		t.Fatal("expected unknown resource error")
	}
}

func TestMockReadModelLogsWithContextHonorsTail(t *testing.T) {
	store := NewMockStore()
	read := store.ReadModel()

	item := resources.ResourceItem{Name: "api-7c6c8d5f7d-x8p2k", Namespace: "default"}
	all, err := read.Logs("pods", item, Scope{Context: "default", Namespace: "default"})
	if err != nil {
		t.Fatalf("expected logs to succeed, got %v", err)
	}
	if len(all) < 6 {
		t.Fatalf("expected enough mock logs for tail check, got %d", len(all))
	}

	streaming, ok := read.(StreamingReadModel)
	if !ok {
		t.Fatalf("expected mock read model to implement StreamingReadModel, got %T", read)
	}
	got, err := streaming.LogsWithContext(context.Background(), "pods", item, Scope{Context: "default", Namespace: "default"}, LogOptions{Tail: 5})
	if err != nil {
		t.Fatalf("expected LogsWithContext to succeed, got %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 tailed lines, got %d", len(got))
	}
	wantStart := len(all) - 5
	for i := range got {
		if got[i] != all[wantStart+i] {
			t.Fatalf("expected tailed logs to match source at index %d", i)
		}
	}
}

func TestMockReadModelLogsWithContextCancelled(t *testing.T) {
	store := NewMockStore()
	read := store.ReadModel()
	streaming, ok := read.(StreamingReadModel)
	if !ok {
		t.Fatalf("expected mock read model to implement StreamingReadModel, got %T", read)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := streaming.LogsWithContext(ctx, "pods", resources.ResourceItem{Name: "api-7c6c8d5f7d-x8p2k", Namespace: "default"}, Scope{Context: "default", Namespace: "default"}, LogOptions{Tail: 10})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}
