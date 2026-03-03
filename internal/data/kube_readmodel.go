package data

import (
	"fmt"
	"strings"

	"github.com/dloss/podji/internal/resources"
)

// KubeReadModel routes pod logs/events through KubeAPI while falling back to
// another read model for list/detail/yaml/describe and non-pod resources.
type KubeReadModel struct {
	fallback ReadModel
	api      KubeAPI
	scope    func() Scope
	onError  func(error)
}

func NewKubeReadModel(fallback ReadModel, api KubeAPI, scope func() Scope, onError func(error)) *KubeReadModel {
	return &KubeReadModel{
		fallback: fallback,
		api:      api,
		scope:    scope,
		onError:  onError,
	}
}

func (k *KubeReadModel) List(resourceName string, scope Scope) ([]resources.ResourceItem, error) {
	if k.fallback == nil {
		return nil, fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.List(resourceName, scope)
}

func (k *KubeReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	if k.fallback == nil {
		return resources.DetailData{}, fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.Detail(resourceName, item, scope)
}

func (k *KubeReadModel) YAML(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	if k.fallback == nil {
		return "", fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.YAML(resourceName, item, scope)
}

func (k *KubeReadModel) Describe(resourceName string, item resources.ResourceItem, scope Scope) (string, error) {
	if k.fallback == nil {
		return "", fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.Describe(resourceName, item, scope)
}

func (k *KubeReadModel) Logs(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	if k.isPodResourceName(resourceName) {
		if k.api == nil {
			return nil, fmt.Errorf("kube api is nil")
		}
		ns, ctx := k.resolveScope(scope, item)
		lines, err := k.api.PodLogs(ctx, ns, item.Name, 200)
		if err != nil {
			k.report(err)
			return nil, err
		}
		return lines, nil
	}
	if k.fallback == nil {
		return nil, fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.Logs(resourceName, item, scope)
}

func (k *KubeReadModel) Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	if k.isPodResourceName(resourceName) {
		if k.api == nil {
			return nil, fmt.Errorf("kube api is nil")
		}
		ns, ctx := k.resolveScope(scope, item)
		lines, err := k.api.PodEvents(ctx, ns, item.Name)
		if err != nil {
			k.report(err)
			return nil, err
		}
		return lines, nil
	}
	if k.fallback == nil {
		return nil, fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.Events(resourceName, item, scope)
}

func (k *KubeReadModel) isPodResourceName(resourceName string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(resourceName)), "pods")
}

func (k *KubeReadModel) resolveScope(scope Scope, item resources.ResourceItem) (namespace, context string) {
	if k.scope != nil {
		scope = k.scope()
	}
	namespace = scope.Namespace
	if strings.TrimSpace(item.Namespace) != "" {
		namespace = item.Namespace
	}
	context = scope.Context
	return namespace, context
}

func (k *KubeReadModel) report(err error) {
	if err != nil && k.onError != nil {
		k.onError(err)
	}
}
