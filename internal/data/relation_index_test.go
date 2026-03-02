package data

import (
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestRelationIndexWorkloadHasPodsAndServices(t *testing.T) {
	store := NewMockStore()
	rel := store.RelationIndex()

	item := resources.ResourceItem{
		Name:     "api",
		Kind:     "DEP",
		Selector: map[string]string{"app": "api"},
	}
	got := rel.Related(Scope{Context: "default", Namespace: "default"}, "workloads", item)
	if len(got["pods"]) == 0 {
		t.Fatalf("expected related pods, got %#v", got)
	}
	if _, ok := got["services"]; !ok {
		t.Fatalf("expected services relation key, got %#v", got)
	}
}

func TestRelationIndexPodHasOwner(t *testing.T) {
	store := NewMockStore()
	rel := store.RelationIndex()

	item := resources.ResourceItem{
		Name:   "api-7c6c8d5f7d-x8p2k",
		Labels: map[string]string{"app": "api"},
	}
	got := rel.Related(Scope{Context: "default", Namespace: "default"}, "pods", item)
	if len(got["owner"]) == 0 {
		t.Fatalf("expected owner relation for pod, got %#v", got)
	}
}
