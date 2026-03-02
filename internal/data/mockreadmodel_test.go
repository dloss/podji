package data

import "testing"

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
