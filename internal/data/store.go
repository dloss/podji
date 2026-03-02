package data

import "github.com/dloss/podji/internal/resources"

type Scope struct {
	Context   string
	Namespace string
}

type Store interface {
	Registry() *resources.Registry
	ReadModel() ReadModel
	RelationIndex() RelationIndex
	AdaptResource(resource resources.ResourceType) resources.ResourceType
	Status() StoreStatus
	Scope() Scope
	SetScope(scope Scope)
	NamespaceNames() []string
	ContextNames() []string
	UnhealthyItems() []resources.ResourceItem
	PodsByRestarts() []resources.ResourceItem
}
