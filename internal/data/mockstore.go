package data

import "github.com/dloss/podji/internal/resources"

type MockStore struct {
	registry *resources.Registry
	scope    Scope
}

func NewMockStore() *MockStore {
	registry := resources.DefaultRegistry()
	scope := Scope{
		Context:   "default",
		Namespace: resources.DefaultNamespace,
	}
	registry.SetNamespace(scope.Namespace)
	return &MockStore{
		registry: registry,
		scope:    scope,
	}
}

func (s *MockStore) Registry() *resources.Registry {
	return s.registry
}

func (s *MockStore) Scope() Scope {
	return s.scope
}

func (s *MockStore) SetScope(scope Scope) {
	if scope.Context == "" {
		scope.Context = "default"
	}
	scope.Namespace = normalizeScopeNamespace(scope.Namespace)
	s.scope = scope
	s.registry.SetNamespace(s.scope.Namespace)
}

func (s *MockStore) NamespaceNames() []string {
	ns := resources.NewNamespaces()
	items := ns.Items()
	names := make([]string, 0, len(items)+1)
	names = append(names, resources.AllNamespaces)
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func (s *MockStore) ContextNames() []string {
	ctx := resources.NewContexts()
	items := ctx.Items()
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

func (s *MockStore) UnhealthyItems() []resources.ResourceItem {
	return resources.UnhealthyItems(s.scope.Namespace)
}

func (s *MockStore) PodsByRestarts() []resources.ResourceItem {
	return resources.PodsByRestarts(s.scope.Namespace)
}

func normalizeScopeNamespace(namespace string) string {
	if namespace == "" {
		return resources.DefaultNamespace
	}
	return namespace
}
