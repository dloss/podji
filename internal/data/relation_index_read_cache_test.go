package data

import (
	"testing"
	"time"

	"github.com/dloss/podji/internal/resources"
)

type countingReadModel struct {
	lists map[string][]resources.ResourceItem
	calls map[string]int
}

func (m *countingReadModel) List(resourceName string, scope Scope) ([]resources.ResourceItem, error) {
	if m.calls == nil {
		m.calls = map[string]int{}
	}
	key := scope.Context + "/" + scope.Namespace + "/" + resourceName
	m.calls[key]++
	items := m.lists[key]
	out := make([]resources.ResourceItem, len(items))
	copy(out, items)
	return out, nil
}

func (m *countingReadModel) Detail(string, resources.ResourceItem, Scope) (resources.DetailData, error) {
	return resources.DetailData{}, nil
}

func (m *countingReadModel) Logs(string, resources.ResourceItem, Scope) ([]string, error) {
	return nil, nil
}

func (m *countingReadModel) Events(string, resources.ResourceItem, Scope) ([]string, error) {
	return nil, nil
}

func (m *countingReadModel) YAML(string, resources.ResourceItem, Scope) (string, error) {
	return "", nil
}

func (m *countingReadModel) Describe(string, resources.ResourceItem, Scope) (string, error) {
	return "", nil
}

func TestReadRelationIndexCachesListsPerScope(t *testing.T) {
	rm := &countingReadModel{
		lists: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "api-1", Labels: map[string]string{"app": "api"}},
			},
			"dev/default/services": {
				{Name: "api-svc", Selector: map[string]string{"app": "api"}},
			},
		},
	}
	rel := newReadRelationIndex(rm)
	scope := Scope{Context: "dev", Namespace: "default"}
	workload := resources.ResourceItem{Name: "api", Selector: map[string]string{"app": "api"}}

	_ = rel.Related(scope, "workloads", workload)
	_ = rel.Related(scope, "workloads", workload)

	if got := rm.calls["dev/default/pods"]; got != 1 {
		t.Fatalf("expected one pod list call, got %d", got)
	}
	if got := rm.calls["dev/default/services"]; got != 1 {
		t.Fatalf("expected one service list call, got %d", got)
	}
}

func TestReadRelationIndexCacheIsScopeBound(t *testing.T) {
	rm := &countingReadModel{
		lists: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "api-1", Labels: map[string]string{"app": "api"}},
			},
			"dev/default/services": {
				{Name: "api-svc", Selector: map[string]string{"app": "api"}},
			},
			"dev/staging/pods": {
				{Name: "api-staging-1", Labels: map[string]string{"app": "api"}},
			},
			"dev/staging/services": {
				{Name: "api-staging-svc", Selector: map[string]string{"app": "api"}},
			},
		},
	}
	rel := newReadRelationIndex(rm)
	workload := resources.ResourceItem{Name: "api", Selector: map[string]string{"app": "api"}}

	_ = rel.Related(Scope{Context: "dev", Namespace: "default"}, "workloads", workload)
	_ = rel.Related(Scope{Context: "dev", Namespace: "staging"}, "workloads", workload)

	if got := rm.calls["dev/default/pods"]; got != 1 {
		t.Fatalf("expected default scope pod list once, got %d", got)
	}
	if got := rm.calls["dev/staging/pods"]; got != 1 {
		t.Fatalf("expected staging scope pod list once, got %d", got)
	}
}

func TestReadRelationIndexOwnerUsesControllerUIDWhenAvailable(t *testing.T) {
	pod := resources.ResourceItem{
		Name: "api-123",
		Extra: map[string]string{
			"controlled-by":     "ReplicaSet/other",
			"controlled-by-uid": "uid-dep-1",
		},
	}
	workloads := []resources.ResourceItem{
		{Name: "api", Kind: "DEP", UID: "uid-dep-1"},
		{Name: "other", Kind: "DEP", UID: "uid-dep-2"},
	}

	owners := relatedOwnerForPod(pod, workloads)
	if len(owners) != 1 || owners[0].Name != "api" {
		t.Fatalf("expected UID-based owner match to api, got %#v", owners)
	}
}

func TestReadRelationIndexCacheExpiresAndReloads(t *testing.T) {
	rm := &countingReadModel{
		lists: map[string][]resources.ResourceItem{
			"dev/default/pods": {
				{Name: "api-1", Labels: map[string]string{"app": "api"}},
			},
			"dev/default/services": {
				{Name: "api-svc", Selector: map[string]string{"app": "api"}},
			},
		},
	}
	rel := newReadRelationIndex(rm)
	rr, ok := rel.(*readRelationIndex)
	if !ok {
		t.Fatalf("expected *readRelationIndex, got %T", rel)
	}
	now := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	rr.now = func() time.Time { return now }
	rr.ttl = 500 * time.Millisecond

	scope := Scope{Context: "dev", Namespace: "default"}
	workload := resources.ResourceItem{Name: "api", Selector: map[string]string{"app": "api"}}
	_ = rel.Related(scope, "workloads", workload)
	_ = rel.Related(scope, "workloads", workload)

	now = now.Add(600 * time.Millisecond)
	_ = rel.Related(scope, "workloads", workload)

	if got := rm.calls["dev/default/pods"]; got != 2 {
		t.Fatalf("expected pod list to reload after ttl expiry, got %d calls", got)
	}
	if got := rm.calls["dev/default/services"]; got != 2 {
		t.Fatalf("expected service list to reload after ttl expiry, got %d calls", got)
	}
}
