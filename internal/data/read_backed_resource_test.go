package data

import (
	"errors"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

type fakeReadModel struct {
	items []resources.ResourceItem
	err   error
}

func (f fakeReadModel) List(resourceName string, scope Scope) ([]resources.ResourceItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func (f fakeReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	if f.err != nil {
		return resources.DetailData{}, f.err
	}
	return resources.DetailData{Summary: []resources.SummaryField{{Key: "status", Value: "from-read-model"}}}, nil
}

func (f fakeReadModel) Logs(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{"read-model-log"}, nil
}

func (f fakeReadModel) Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{"read-model-event"}, nil
}

func (f fakeReadModel) YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "read-model-yaml", nil
}

func (f fakeReadModel) Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "read-model-describe", nil
}

func TestReadBackedResourceDelegatesToReadModel(t *testing.T) {
	base := resources.NewPods()
	base.SetNamespace("default")
	adapter := NewReadBackedResource(base, fakeReadModel{
		items: []resources.ResourceItem{{Name: "from-read-model"}},
	}, func() Scope { return Scope{Context: "default", Namespace: "default"} })

	items := adapter.Items()
	if len(items) != 1 || items[0].Name != "from-read-model" {
		t.Fatalf("expected read-model items, got %#v", items)
	}
	if got := adapter.Logs(resources.ResourceItem{Name: "x"}); len(got) == 0 || got[0] != "read-model-log" {
		t.Fatalf("expected read-model logs, got %#v", got)
	}
}

func TestReadBackedResourceFallsBackToBaseOnError(t *testing.T) {
	base := resources.NewPods()
	base.SetNamespace("default")
	adapter := NewReadBackedResource(base, fakeReadModel{err: errors.New("boom")}, func() Scope {
		return Scope{Context: "default", Namespace: "default"}
	})
	items := adapter.Items()
	if len(items) == 0 {
		t.Fatal("expected fallback items from base resource")
	}
}
