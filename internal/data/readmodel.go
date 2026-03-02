package data

import "github.com/dloss/podji/internal/resources"

// ReadModel defines resource read operations independent from concrete data source.
// Implementations may be mock-backed, informer-backed, or API-backed.
type ReadModel interface {
	List(resourceName string, scope Scope) ([]resources.ResourceItem, error)
	Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error)
	Logs(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error)
	Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error)
	YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error)
	Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error)
}
