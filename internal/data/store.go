package data

import "github.com/dloss/podji/internal/resources"

type Scope struct {
	Context   string
	Namespace string
}

type Store interface {
	Registry() *resources.Registry
	ReadModel() ReadModel
	Scope() Scope
	SetScope(scope Scope)
	NamespaceNames() []string
	ContextNames() []string
	UnhealthyItems() []resources.ResourceItem
	PodsByRestarts() []resources.ResourceItem
}
