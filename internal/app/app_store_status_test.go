package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/data"
	"github.com/dloss/podji/internal/resources"
)

type statusStore struct {
	registry *resources.Registry
	scope    data.Scope
	status   data.StoreStatus
}

func newStatusStore() *statusStore {
	reg := resources.DefaultRegistry()
	scope := data.Scope{Context: "default", Namespace: resources.DefaultNamespace}
	reg.SetNamespace(scope.Namespace)
	return &statusStore{
		registry: reg,
		scope:    scope,
		status:   data.StoreStatus{State: data.StoreStateReady},
	}
}

func (s *statusStore) Registry() *resources.Registry { return s.registry }
func (s *statusStore) ReadModel() data.ReadModel     { return data.NewMockReadModel(s.registry) }
func (s *statusStore) RelationIndex() data.RelationIndex {
	return nil
}
func (s *statusStore) AdaptResource(resource resources.ResourceType) resources.ResourceType {
	return resource
}
func (s *statusStore) Status() data.StoreStatus { return s.status }
func (s *statusStore) Scope() data.Scope        { return s.scope }
func (s *statusStore) SetScope(scope data.Scope) {
	s.scope = scope
	s.registry.SetNamespace(scope.Namespace)
}
func (s *statusStore) NamespaceNames() []string {
	s.status = data.StoreStatus{
		State:   data.StoreStateDegraded,
		Message: "namespace discovery failed",
	}
	return []string{resources.AllNamespaces, resources.DefaultNamespace}
}
func (s *statusStore) ContextNames() []string {
	s.status = data.StoreStatus{
		State:   data.StoreStateDegraded,
		Message: "context discovery failed",
	}
	return []string{"default"}
}
func (s *statusStore) UnhealthyItems() []resources.ResourceItem { return nil }
func (s *statusStore) PodsByRestarts() []resources.ResourceItem { return nil }

func TestNamespacePickerSyncsDegradedStoreStatus(t *testing.T) {
	store := newStatusStore()
	m := NewWithStore(store)
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'N'}})
	got := updated.(Model)
	if !strings.Contains(got.errorMsg, "store (degraded):") {
		t.Fatalf("expected state-prefixed store status, got %q", got.errorMsg)
	}
	if !strings.Contains(got.errorMsg, "namespace discovery failed") {
		t.Fatalf("expected degraded store error message, got %q", got.errorMsg)
	}
}

func TestContextPickerSyncsDegradedStoreStatus(t *testing.T) {
	store := newStatusStore()
	m := NewWithStore(store)
	updated, _ := m.Update(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune{'X'}})
	got := updated.(Model)
	if !strings.Contains(got.errorMsg, "store (degraded):") {
		t.Fatalf("expected state-prefixed store status, got %q", got.errorMsg)
	}
	if !strings.Contains(got.errorMsg, "context discovery failed") {
		t.Fatalf("expected degraded store error message, got %q", got.errorMsg)
	}
}

func TestModelInitRendersLoadingStoreStatus(t *testing.T) {
	store := newStatusStore()
	store.status = data.StoreStatus{
		State:   data.StoreStateLoading,
		Message: "connecting to cluster",
	}
	m := NewWithStore(store)
	m.syncStoreStatus()
	if !strings.Contains(m.errorMsg, "store (loading): connecting to cluster") {
		t.Fatalf("expected loading store status message, got %q", m.errorMsg)
	}
}
