package data

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	api, err := newClientGoAPI()
	if err != nil {
		return nil, err
	}
	return newKubeStore(api)
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
		relations: nil,
		scope:     scope,
		api:       api,
		status:    StoreStatus{State: StoreStateLoading, Message: "connecting to cluster"},
	}
	store.read = NewKubeReadModel(
		NewMockReadModel(registry),
		api,
		store.Scope,
		store.setStatusForError,
		store.setStatusPartialForUnsupportedList,
		store.setStatusWarmingCacheForList,
		store.markStatusReady,
	)
	store.relations = newReadRelationIndex(store.read)
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
	prev := s.scope
	if scope.Context == "" {
		scope.Context = s.scope.Context
	}
	scope.Namespace = normalizeScopeNamespace(scope.Namespace)
	s.scope = scope
	s.registry.SetNamespace(s.scope.Namespace)
	if prev.Context != s.scope.Context || prev.Namespace != s.scope.Namespace {
		s.status = StoreStatus{State: StoreStateLoading, Message: "refreshing cluster data"}
	}
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
	pods, errPods := s.api.ListResources(s.scope.Context, s.scope.Namespace, "pods")
	deployments, errDeps := s.api.ListResources(s.scope.Context, s.scope.Namespace, "deployments")
	pvcs, errPVC := s.api.ListResources(s.scope.Context, s.scope.Namespace, "persistentvolumeclaims")
	if errPods != nil || errDeps != nil || errPVC != nil {
		s.setStatusForQueryFallback("unhealthy", errPods, errDeps, errPVC)
		return resources.UnhealthyItems(s.scope.Namespace)
	}
	s.markStatusReady()

	var out []resources.ResourceItem
	for _, item := range append(append(pods, deployments...), pvcs...) {
		if !isUnhealthy(item) {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		si := unhealthySeverity(out[i])
		sj := unhealthySeverity(out[j])
		if si != sj {
			return si < sj
		}
		ai := parseAgeForSort(out[i].Age)
		aj := parseAgeForSort(out[j].Age)
		if ai != aj {
			return ai < aj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *KubeStore) PodsByRestarts() []resources.ResourceItem {
	pods, err := s.api.ListResources(s.scope.Context, s.scope.Namespace, "pods")
	if err != nil {
		s.setStatusForQueryFallback("restarts", err)
		return resources.PodsByRestarts(s.scope.Namespace)
	}
	s.markStatusReady()
	out := make([]resources.ResourceItem, 0, len(pods))
	for _, item := range pods {
		if parseRestartCount(item.Restarts) > 0 {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri := parseRestartCount(out[i].Restarts)
		rj := parseRestartCount(out[j].Restarts)
		if ri != rj {
			return ri > rj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *KubeStore) setStatusForQueryFallback(query string, errs ...error) {
	for _, err := range errs {
		if err == nil {
			continue
		}
		if errors.Is(err, ErrListNotSupported) {
			s.status = StoreStatus{
				State:   StoreStatePartial,
				Message: fmt.Sprintf("live %s query unavailable; using mock fallback", query),
			}
			continue
		}
		s.setStatusForError(err)
		return
	}
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

func (s *KubeStore) setStatusPartialForUnsupportedList(resourceName string) {
	s.status = StoreStatus{
		State:   StoreStatePartial,
		Message: fmt.Sprintf("live %s list unavailable; using mock fallback", resourceName),
	}
}

func (s *KubeStore) setStatusWarmingCacheForList(resourceName string) {
	s.status = StoreStatus{
		State:   StoreStateLoading,
		Message: fmt.Sprintf("warming cache for %s", resourceName),
	}
}

func (s *KubeStore) markStatusReady() {
	s.status = StoreStatus{State: StoreStateReady}
}

func parseRestartCount(raw string) int {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return n
}

func unhealthySeverity(item resources.ResourceItem) int {
	if strings.Contains(strings.ToLower(item.Status), "fail") || strings.Contains(strings.ToLower(item.Status), "crash") {
		return 0
	}
	if strings.Contains(strings.ToLower(item.Status), "degrad") {
		return 1
	}
	return 2
}

func isUnhealthy(item resources.ResourceItem) bool {
	status := strings.ToLower(item.Status)
	if status == "" || status == "healthy" || status == "running" || status == "bound" {
		return false
	}
	return true
}

func parseAgeForSort(age string) int {
	age = strings.TrimSpace(age)
	if age == "" {
		return 0
	}
	suffix := age[len(age)-1]
	n, err := strconv.Atoi(age[:len(age)-1])
	if err != nil {
		return 0
	}
	switch suffix {
	case 'm':
		return n
	case 'h':
		return n * 60
	case 'd':
		return n * 24 * 60
	default:
		return 0
	}
}
