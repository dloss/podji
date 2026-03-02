package data

import (
	"strings"

	"github.com/dloss/podji/internal/resources"
)

// RelationIndex answers related-resource lookups from local store state.
// Implementations must avoid cluster network calls during lookup.
type RelationIndex interface {
	Related(scope Scope, resourceName string, item resources.ResourceItem) map[string][]resources.ResourceItem
}

type mockRelationIndex struct {
	registry *resources.Registry
}

func newMockRelationIndex(registry *resources.Registry) RelationIndex {
	return &mockRelationIndex{registry: registry}
}

func (r *mockRelationIndex) Related(scope Scope, resourceName string, item resources.ResourceItem) map[string][]resources.ResourceItem {
	if r.registry == nil {
		return map[string][]resources.ResourceItem{}
	}
	r.registry.SetNamespace(scope.Namespace)

	name := strings.ToLower(strings.TrimSpace(resourceName))
	out := map[string][]resources.ResourceItem{}

	switch {
	case name == "workloads" || name == "deployments":
		out["pods"] = resources.NewWorkloadPods(item, r.registry).Items()
		out["services"] = resources.NewRelatedServices(item, r.registry).Items()
		out["config"] = resources.NewRelatedConfig(item.Name).Items()
		out["storage"] = resources.NewRelatedStorage(item.Name).Items()
	case strings.HasPrefix(name, "pods"):
		out["owner"] = resources.NewPodOwner(item.Name).Items()
		out["services"] = resources.NewPodServices(item, r.registry).Items()
		out["config"] = resources.NewPodConfig(item.Name).Items()
		out["storage"] = resources.NewPodStorage(item.Name).Items()
	case strings.HasPrefix(name, "services"):
		out["backends"] = resources.NewBackends(item, r.registry).Items()
		out["ingresses"] = resources.NewRelatedIngresses(item.Name).Items()
	case strings.HasPrefix(name, "ingresses"):
		out["services"] = resources.NewIngressServices(item.Name).Items()
	case name == "nodes":
		out["pods"] = resources.NewNodePods(item.Name).Items()
	case name == "persistentvolumeclaims":
		out["mounted-by"] = resources.NewMountedBy(item.Name).Items()
	default:
		return map[string][]resources.ResourceItem{}
	}

	return out
}
