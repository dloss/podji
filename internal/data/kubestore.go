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
	status    StoreStatus
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
		read:      nil,
		relations: newMockRelationIndex(registry),
		scope:     scope,
		api:       api,
		status:    StoreStatus{State: StoreStateReady},
	}
	store.read = NewKubeReadModel(NewMockReadModel(registry), api, store.Scope, store.setStatusForError)
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
	return s.status
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
			s.setStatusForError(err)
		} else {
			s.status = StoreStatus{
				State:   StoreStatePartial,
				Message: "no namespaces discovered",
			}
		}
		return []string{resources.AllNamespaces, resources.DefaultNamespace}
	}
	s.status = StoreStatus{State: StoreStateReady}
	out := make([]string, 0, len(namespaces)+1)
	out = append(out, resources.AllNamespaces)
	out = append(out, namespaces...)
	return out
}

func (s *KubeStore) ContextNames() []string {
	contexts, err := s.api.Contexts()
	if err != nil || len(contexts) == 0 {
		if err != nil {
			s.setStatusForError(err)
		} else {
			s.status = StoreStatus{
				State:   StoreStatePartial,
				Message: "no contexts discovered",
			}
		}
		return []string{s.scope.Context}
	}
	s.status = StoreStatus{State: StoreStateReady}
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
	lines, err := s.api.PodLogs(s.scope.Context, namespace, pod, 200)
	if err != nil {
		s.setStatusForError(err)
		return nil, err
	}
	return lines, nil
}

func (s *KubeStore) podEvents(namespace, pod string) ([]string, error) {
	lines, err := s.api.PodEvents(s.scope.Context, namespace, pod)
	if err != nil {
		s.setStatusForError(err)
		return nil, err
	}
	return lines, nil
}

func (s *KubeStore) setStatusForError(err error) {
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "forbidden"), strings.Contains(lower, "permission denied"), strings.Contains(lower, "(403)"):
		s.status = StoreStatus{State: StoreStateForbidden, Message: msg}
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "timed out"),
		strings.Contains(lower, "timeout"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "unreachable"),
		strings.Contains(lower, "context deadline exceeded"),
		strings.Contains(lower, "unable to connect"):
		s.status = StoreStatus{State: StoreStateUnreachable, Message: msg}
	default:
		s.status = StoreStatus{State: StoreStateDegraded, Message: msg}
	}
}
