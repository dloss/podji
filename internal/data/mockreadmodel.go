package data

import (
	"context"
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

func (m *MockReadModel) LogsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	lines, err := m.Logs(resourceName, item, scope)
	if err != nil {
		return nil, err
	}
	if opts.Tail <= 0 || opts.Tail >= len(lines) {
		return lines, nil
	}
	start := len(lines) - opts.Tail
	out := make([]string, opts.Tail)
	copy(out, lines[start:])
	return out, nil
}

func (m *MockReadModel) Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	res, err := m.resourceFor(resourceName, scope)
	if err != nil {
		return nil, err
	}
	return res.Events(item), nil
}

func (m *MockReadModel) EventsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts EventOptions) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	lines, err := m.Events(resourceName, item, scope)
	if err != nil {
		return nil, err
	}
	if opts.Limit <= 0 || opts.Limit >= len(lines) {
		return lines, nil
	}
	out := make([]string, opts.Limit)
	copy(out, lines[:opts.Limit])
	return out, nil
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
