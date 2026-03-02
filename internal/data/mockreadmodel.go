package data

import (
	"fmt"

	"github.com/dloss/podji/internal/resources"
)

type MockReadModel struct {
	registry *resources.Registry
}

func NewMockReadModel(registry *resources.Registry) *MockReadModel {
	return &MockReadModel{registry: registry}
}

func (m *MockReadModel) List(resourceName string, scope Scope) ([]resources.ResourceItem, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return nil, err
	}
	return res.Items(), nil
}

func (m *MockReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return resources.DetailData{}, err
	}
	return res.Detail(item), nil
}

func (m *MockReadModel) Logs(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return nil, err
	}
	return res.Logs(item), nil
}

func (m *MockReadModel) Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return nil, err
	}
	return res.Events(item), nil
}

func (m *MockReadModel) YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return "", err
	}
	return res.YAML(item), nil
}

func (m *MockReadModel) Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return "", err
	}
	return res.Describe(item), nil
}

func (m *MockReadModel) resourceFor(resourceName string, scope Scope) (resources.ResourceType, error) {
	if m.registry == nil {
		return nil, fmt.Errorf("read model has no registry")
	}
	res := m.registry.ByName(resourceName)
	if res == nil {
		return nil, fmt.Errorf("resource %q not found", resourceName)
	}
	if scoped, ok := res.(resources.NamespaceScoped); ok {
		scoped.SetNamespace(scope.Namespace)
	}
	return res, nil
}
