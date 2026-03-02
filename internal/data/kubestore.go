package data

import (
	"fmt"
	"strings"

	"github.com/dloss/podji/internal/resources"
)

type KubeStore struct {
	registry  *resources.Registry
	read      ReadModel
	relations RelationIndex
	scope     Scope
	api       KubeAPI
	lastErr   string
}

func NewKubeStore() (*KubeStore, error) {
	return newKubeStore(newKubectlAPI(execRunner{}))
}

func newKubeStore(api KubeAPI) (*KubeStore, error) {
	if api == nil {
		return nil, fmt.Errorf("kube api is nil")
	}
	contexts, err := api.Contexts()
	if err != nil {
		return nil, err
	}

	scope := Scope{
		Context:   "default",
		Namespace: resources.DefaultNamespace,
	}
	if len(contexts) > 0 {
		scope.Context = contexts[0]
	}

	registry := resources.DefaultRegistry()
	registry.SetNamespace(scope.Namespace)

	store := &KubeStore{
		registry:  registry,
		read:      NewMockReadModel(registry),
		relations: newMockRelationIndex(registry),
		scope:     scope,
		api:       api,
	}
	store.configurePodFetchers()
	return store, nil
}

func (s *KubeStore) Registry() *resources.Registry {
	return s.registry
}

func (s *KubeStore) Scope() Scope {
	return s.scope
}

func (s *KubeStore) ReadModel() ReadModel {
	return s.read
}

func (s *KubeStore) RelationIndex() RelationIndex {
	return s.relations
}

func (s *KubeStore) AdaptResource(resource resources.ResourceType) resources.ResourceType {
	if resource == nil {
		return nil
	}
	return NewReadBackedResource(resource, s.read, s.Scope)
}

func (s *KubeStore) Status() StoreStatus {
	if strings.TrimSpace(s.lastErr) != "" {
		return StoreStatus{
			State:   StoreStateDegraded,
			Message: s.lastErr,
		}
	}
	return StoreStatus{State: StoreStateReady}
}

func (s *KubeStore) SetScope(scope Scope) {
	if scope.Context == "" {
		scope.Context = s.scope.Context
	}
	scope.Namespace = normalizeScopeNamespace(scope.Namespace)
	s.scope = scope
	s.registry.SetNamespace(s.scope.Namespace)
}

func (s *KubeStore) NamespaceNames() []string {
	namespaces, err := s.api.Namespaces(s.scope.Context)
	if err != nil || len(namespaces) == 0 {
		if err != nil {
			s.lastErr = err.Error()
		} else {
			s.lastErr = "no namespaces discovered"
		}
		return []string{resources.AllNamespaces, resources.DefaultNamespace}
	}
	s.lastErr = ""
	out := make([]string, 0, len(namespaces)+1)
	out = append(out, resources.AllNamespaces)
	out = append(out, namespaces...)
	return out
}

func (s *KubeStore) ContextNames() []string {
	contexts, err := s.api.Contexts()
	if err != nil || len(contexts) == 0 {
		if err != nil {
			s.lastErr = err.Error()
		} else {
			s.lastErr = "no contexts discovered"
		}
		return []string{s.scope.Context}
	}
	s.lastErr = ""
	return contexts
}

func (s *KubeStore) UnhealthyItems() []resources.ResourceItem {
	// Query aggregation still uses the shared resource query path for now.
	return resources.UnhealthyItems(s.scope.Namespace)
}

func (s *KubeStore) PodsByRestarts() []resources.ResourceItem {
	// Query aggregation still uses the shared resource query path for now.
	return resources.PodsByRestarts(s.scope.Namespace)
}

func (s *KubeStore) configurePodFetchers() {
	pods, ok := s.registry.ByName("pods").(*resources.Pods)
	if !ok {
		return
	}
	pods.SetLiveFetchers(s.podLogs, s.podEvents)
}

func (s *KubeStore) podLogs(namespace, pod string) ([]string, error) {
	return s.api.PodLogs(s.scope.Context, namespace, pod, 200)
}

func (s *KubeStore) podEvents(namespace, pod string) ([]string, error) {
	return s.api.PodEvents(s.scope.Context, namespace, pod)
}
