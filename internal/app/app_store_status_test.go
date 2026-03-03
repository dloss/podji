package app

import (
	"strings"
	"testing"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/data"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/overlaypicker"
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
	s.status = data.StoreStatus{
		State:   data.StoreStateLoading,
		Message: "refreshing cluster data",
	}
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

type queryStatusStore struct {
	*statusStore
}

func (s *queryStatusStore) UnhealthyItems() []resources.ResourceItem {
	s.status = data.StoreStatus{
		State:   data.StoreStatePartial,
		Message: "live unhealthy query unavailable; using mock fallback",
	}
	return []resources.ResourceItem{{Name: "api", Kind: "DEP", Status: "Degraded", Ready: "0/1"}}
}

func (s *queryStatusStore) PodsByRestarts() []resources.ResourceItem {
	s.status = data.StoreStatus{
		State:   data.StoreStateUnreachable,
		Message: "connection refused",
	}
	return []resources.ResourceItem{{Name: "pod-a", Restarts: "4"}}
}

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

func TestScopeSelectionSyncsLoadingStoreStatus(t *testing.T) {
	store := newStatusStore()
	m := NewWithStore(store)
	updated, _ := m.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := updated.(Model)
	if !strings.Contains(got.errorMsg, "store (loading): refreshing cluster data") {
		t.Fatalf("expected loading store status after scope selection, got %q", got.errorMsg)
	}
}

func TestUnhealthyCommandSyncsStoreStatus(t *testing.T) {
	store := &queryStatusStore{statusStore: newStatusStore()}
	m := NewWithStore(store)
	m.width = 120
	m.height = 40
	if err := m.runCommand("unhealthy"); err != "" {
		t.Fatalf("expected no command error, got %q", err)
	}
	if !strings.Contains(m.errorMsg, "store (partial): live unhealthy query unavailable; using mock fallback") {
		t.Fatalf("expected partial store status after unhealthy query, got %q", m.errorMsg)
	}
}

func TestRestartsCommandSyncsStoreStatus(t *testing.T) {
	store := &queryStatusStore{statusStore: newStatusStore()}
	m := NewWithStore(store)
	m.width = 120
	m.height = 40
	if err := m.runCommand("restarts"); err != "" {
		t.Fatalf("expected no command error, got %q", err)
	}
	if !strings.Contains(m.errorMsg, "store (unreachable): connection refused") {
		t.Fatalf("expected unreachable store status after restarts query, got %q", m.errorMsg)
	}
}

func TestSyncStoreStatusClearsStoreMessageWhenReady(t *testing.T) {
	store := newStatusStore()
	store.status = data.StoreStatus{
		State:   data.StoreStateLoading,
		Message: "warming cache for pods",
	}
	m := NewWithStore(store)
	m.syncStoreStatus()
	if !strings.Contains(m.errorMsg, "store (loading): warming cache for pods") {
		t.Fatalf("expected loading store message, got %q", m.errorMsg)
	}
	store.status = data.StoreStatus{State: data.StoreStateReady}
	m.syncStoreStatus()
	if strings.HasPrefix(m.errorMsg, "store (") {
		t.Fatalf("expected store-prefixed message to clear on ready state, got %q", m.errorMsg)
	}
}
