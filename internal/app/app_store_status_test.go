package app

import (
	"strings"
	"testing"
	"time"

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
		Message: "live unhealthy query unavailable",
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
	if !strings.Contains(m.errorMsg, "store (partial): live unhealthy query unavailable") {
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

func TestSyncStoreStatusShowsReadyStoreFreshnessMessage(t *testing.T) {
	store := newStatusStore()
	store.status = data.StoreStatus{
		State:   data.StoreStateReady,
		Message: "cache ready for pods",
	}
	m := NewWithStore(store)
	m.syncStoreStatus()
	if m.errorMsg != "" {
		t.Fatalf("expected no error message for ready store state, got %q", m.errorMsg)
	}
	if m.statusMsg != "" {
		t.Fatalf("expected no store-prefixed ready message, got %q", m.statusMsg)
	}
}

func TestViewAppliesStoreStatusSyncForRendering(t *testing.T) {
	store := newStatusStore()
	store.status = data.StoreStatus{
		State:   data.StoreStateLoading,
		Message: "warming cache for pods",
	}
	m := NewWithStore(store)
	m.width = 120
	m.height = 40
	rendered := m.View()
	if !strings.Contains(rendered, "store (loading): warming cache for pods") {
		t.Fatalf("expected rendered view to include synced store status, got %q", rendered)
	}
}

func TestScopeSwitchRenderTransitionsFromLoadingToReadyFreshness(t *testing.T) {
	store := newStatusStore()
	m := NewWithStore(store)
	m.width = 120
	m.height = 40

	updated, _ := m.Update(overlaypicker.SelectedMsg{Kind: "namespace", Value: "staging"})
	got := updated.(Model)
	loadingView := got.View()
	if !strings.Contains(loadingView, "store (loading): refreshing cluster data") {
		t.Fatalf("expected loading status after scope switch, got %q", loadingView)
	}

	store.status = data.StoreStatus{
		State:   data.StoreStateReady,
		Message: "cache ready for workloads",
	}
	readyView := got.View()
	if strings.Contains(readyView, "store: cache ready for workloads") {
		t.Fatalf("expected no ready store status banner after transition, got %q", readyView)
	}
}

func TestDecorateFooterWithStoreFreshnessCacheBadge(t *testing.T) {
	m := NewWithStore(newStatusStore())
	m.mode = data.ModeKube
	m.storeStatus = data.StoreStatus{
		State:         data.StoreStateReady,
		Source:        data.StoreDataSourceCache,
		LastSuccessAt: time.Now(),
		StaleAfter:    15 * time.Second,
	}
	m.width = 120
	footer := m.decorateFooterWithStoreFreshness("status line\nactions line")
	if !strings.Contains(footer, "data:cache") {
		t.Fatalf("expected cache freshness badge in footer, got %q", footer)
	}
}

func TestCommandQueryNavigationConsistencyAcrossStoreAdapters(t *testing.T) {
	cases := []struct {
		name  string
		store data.Store
	}{
		{name: "mock", store: data.NewMockStore()},
		{name: "query-status", store: &queryStatusStore{statusStore: newStatusStore()}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWithStore(tc.store)
			m.width = 120
			m.height = 40

			stackBefore := len(m.stack)
			if err := m.runCommand("unhealthy"); err != "" {
				t.Fatalf("expected no error for unhealthy query, got %q", err)
			}
			if len(m.stack) != stackBefore+1 {
				t.Fatalf("expected unhealthy query to push one view, got stack len %d", len(m.stack))
			}
			if got := m.crumbs[len(m.crumbs)-1]; got != "unhealthy" {
				t.Fatalf("expected unhealthy crumb, got %q", got)
			}

			stackBefore = len(m.stack)
			if err := m.runCommand("restarts"); err != "" {
				t.Fatalf("expected no error for restarts query, got %q", err)
			}
			if len(m.stack) != stackBefore+1 {
				t.Fatalf("expected restarts query to push one view, got stack len %d", len(m.stack))
			}
			if got := m.crumbs[len(m.crumbs)-1]; got != "restarts" {
				t.Fatalf("expected restarts crumb, got %q", got)
			}
		})
	}
}
