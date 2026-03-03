package data

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dloss/podji/internal/resources"
)

// KubeReadModel routes pod logs/events through KubeAPI while falling back to
// another read model for list/detail/yaml/describe and non-pod resources.
type KubeReadModel struct {
	fallback  ReadModel
	api       KubeAPI
	scope     func() Scope
	onError   func(error)
	onPartial func(string)
	onReady   func()
}

func NewKubeReadModel(
	fallback ReadModel,
	api KubeAPI,
	scope func() Scope,
	onError func(error),
	onPartial func(string),
	onReady func(),
) *KubeReadModel {
	return &KubeReadModel{
		fallback:  fallback,
		api:       api,
		scope:     scope,
		onError:   onError,
		onPartial: onPartial,
		onReady:   onReady,
	}
}

func (k *KubeReadModel) List(resourceName string, scope Scope) ([]resources.ResourceItem, error) {
	if k.api != nil {
		active := scope
		if k.scope != nil {
			active = k.scope()
		}
		items, err := k.api.ListResources(active.Context, active.Namespace, resourceName)
		if err == nil {
			k.markReady()
			return items, nil
		}
		if !errors.Is(err, ErrListNotSupported) {
			k.report(err)
			return nil, err
		}
		if k.onPartial != nil {
			k.onPartial(resourceName)
		}
	}
	if k.fallback == nil {
		return nil, fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.List(resourceName, scope)
}

func (k *KubeReadModel) Detail(resourceName string, item resources.ResourceItem, scope Scope) (resources.DetailData, error) {
	if detail, ok := liveDetail(resourceName, item); ok {
		k.markReady()
		return detail, nil
	}
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
	return k.LogsWithContext(context.Background(), resourceName, item, scope, LogOptions{Tail: 200})
}

func (k *KubeReadModel) LogsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts LogOptions) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if k.isPodResourceName(resourceName) {
		if k.api == nil {
			return nil, fmt.Errorf("kube api is nil")
		}
		ns, contextName := k.resolveScope(scope, item)
		tail := opts.Tail
		if tail <= 0 {
			tail = 200
		}
		lines, err := k.api.PodLogs(contextName, ns, item.Name, tail)
		if err != nil {
			k.report(err)
			return nil, err
		}
		k.markReady()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return lines, nil
	}
	if k.fallback == nil {
		return nil, fmt.Errorf("kube read model fallback is nil")
	}
	return k.fallback.Logs(resourceName, item, scope)
}

func (k *KubeReadModel) Events(resourceName string, item resources.ResourceItem, scope Scope) ([]string, error) {
	return k.EventsWithContext(context.Background(), resourceName, item, scope, EventOptions{})
}

func (k *KubeReadModel) EventsWithContext(ctx context.Context, resourceName string, item resources.ResourceItem, scope Scope, opts EventOptions) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if k.isPodResourceName(resourceName) {
		if k.api == nil {
			return nil, fmt.Errorf("kube api is nil")
		}
		ns, contextName := k.resolveScope(scope, item)
		lines, err := k.api.PodEvents(contextName, ns, item.Name)
		if err != nil {
			k.report(err)
			return nil, err
		}
		k.markReady()
		if opts.Limit > 0 && len(lines) > opts.Limit {
			lines = lines[:opts.Limit]
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
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

func (k *KubeReadModel) markReady() {
	if k.onReady != nil {
		k.onReady()
	}
}

func liveDetail(resourceName string, item resources.ResourceItem) (resources.DetailData, bool) {
	name := strings.ToLower(strings.TrimSpace(resourceName))
	switch {
	case strings.HasPrefix(name, "pods"):
		return podLiveDetail(item), true
	case strings.HasPrefix(name, "services"):
		return serviceLiveDetail(item), true
	case strings.HasPrefix(name, "deployments"), strings.HasPrefix(name, "workloads"):
		return workloadLiveDetail(item), true
	default:
		return resources.DetailData{}, false
	}
}

func podLiveDetail(item resources.ResourceItem) resources.DetailData {
	status := valueOr(item.Status, "Unknown")
	ready := valueOr(item.Ready, "unknown")
	node := valueOr(item.Extra["node"], "<none>")
	ip := valueOr(item.Extra["ip"], "<none>")
	qos := valueOr(item.Extra["qos"], "Unknown")
	containers := strings.Split(strings.TrimSpace(item.Extra["containers"]), ",")
	images := strings.Split(strings.TrimSpace(item.Extra["images"]), ",")
	rows := make([]resources.ContainerRow, 0, len(containers))
	for i := range containers {
		name := strings.TrimSpace(containers[i])
		if name == "" {
			continue
		}
		image := "<unknown>"
		if i < len(images) && strings.TrimSpace(images[i]) != "" {
			image = strings.TrimSpace(images[i])
		}
		rows = append(rows, resources.ContainerRow{
			Name:     name,
			Image:    image,
			State:    "Unknown",
			Restarts: valueOr(item.Restarts, "0"),
		})
	}
	return resources.DetailData{
		Summary: []resources.SummaryField{
			{Key: "status", Label: "Status", Value: status},
			{Key: "ready", Label: "Ready", Value: ready},
			{Key: "node", Label: "Node", Value: node},
			{Key: "ip", Label: "IP", Value: ip},
			{Key: "qos", Label: "QoS", Value: qos},
		},
		Containers: rows,
		Labels:     labelsFromMap(item.Labels),
	}
}

func serviceLiveDetail(item resources.ResourceItem) resources.DetailData {
	return resources.DetailData{
		Summary: []resources.SummaryField{
			{Key: "status", Label: "Status", Value: valueOr(item.Status, "Healthy")},
			{Key: "type", Label: "Type", Value: valueOr(item.Kind, "ClusterIP")},
			{Key: "endpoints", Label: "Endpoints", Value: valueOr(item.Ready, "0 endpoints")},
			{Key: "selector", Label: "Selector", Value: valueOr(item.Extra["selector"], "<none>")},
			{Key: "external_ip", Label: "External IP", Value: valueOr(item.Extra["external-ip"], "<none>")},
		},
		Labels: labelsFromMap(item.Labels),
	}
}

func workloadLiveDetail(item resources.ResourceItem) resources.DetailData {
	return resources.DetailData{
		Summary: []resources.SummaryField{
			{Key: "kind", Label: "Kind", Value: valueOr(item.Kind, "Workload")},
			{Key: "status", Label: "Status", Value: valueOr(item.Status, "Unknown")},
			{Key: "ready", Label: "Ready", Value: valueOr(item.Ready, "unknown")},
			{Key: "strategy", Label: "Strategy", Value: valueOr(item.Extra["strategy"], "<none>")},
			{Key: "selector", Label: "Selector", Value: valueOr(item.Extra["selector"], "<none>")},
			{Key: "images", Label: "Images", Value: valueOr(item.Extra["images"], "<unknown>")},
		},
		Labels: labelsFromMap(item.Labels),
	}
}

func labelsFromMap(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+labels[key])
	}
	return out
}

func valueOr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
