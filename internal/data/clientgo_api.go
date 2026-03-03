package data

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dloss/podji/internal/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type clientGoAPI struct {
	loader  clientcmd.ClientConfigLoadingRules
	mu      sync.Mutex
	nsTTL   time.Duration
	ns      map[string]namespaceCacheEntry
	listTTL time.Duration
	list    map[string]listCacheEntry
}

type namespaceCacheEntry struct {
	items     []string
	expiresAt time.Time
}

type listCacheEntry struct {
	items     []resources.ResourceItem
	expiresAt time.Time
}

func newClientGoAPI() (KubeAPI, error) {
	loader := *clientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed loading kubeconfig: %w", err)
	}
	if len(cfg.Contexts) == 0 {
		return nil, fmt.Errorf("kubeconfig has no contexts")
	}
	return &clientGoAPI{
		loader:  loader,
		nsTTL:   5 * time.Second,
		ns:      map[string]namespaceCacheEntry{},
		listTTL: 3 * time.Second,
		list:    map[string]listCacheEntry{},
	}, nil
}

func (k *clientGoAPI) Contexts() ([]string, error) {
	cfg, err := k.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed loading kubeconfig: %w", err)
	}
	out := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func (k *clientGoAPI) Namespaces(contextName string) ([]string, error) {
	if cached, ok := k.namespaceCacheGet(contextName); ok {
		return cached, nil
	}
	client, err := k.clientForContext(contextName)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	list, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces for context %q: %w", contextName, err)
	}
	out := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		out = append(out, ns.Name)
	}
	sort.Strings(out)
	k.namespaceCacheSet(contextName, out)
	return out, nil
}

func (k *clientGoAPI) PodLogs(contextName, namespace, pod string, tail int) ([]string, error) {
	client, err := k.clientForContext(contextName)
	if err != nil {
		return nil, err
	}
	if tail <= 0 {
		tail = 200
	}
	tail64 := int64(tail)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		TailLines: &tail64,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs for %s/%s: %w", namespace, pod, err)
	}
	defer stream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stream); err != nil {
		return nil, fmt.Errorf("failed reading logs for %s/%s: %w", namespace, pod, err)
	}
	lines := splitNonEmptyLines(buf.String())
	if len(lines) == 0 {
		return []string{"No log lines returned."}, nil
	}
	return lines, nil
}

func (k *clientGoAPI) ListResources(contextName, namespace, resourceName string) ([]resources.ResourceItem, error) {
	key := strings.ToLower(strings.TrimSpace(resourceName))
	cacheKey := contextName + "|" + namespace + "|" + key
	if cached, ok := k.listCacheGet(cacheKey); ok {
		return cached, nil
	}

	client, err := k.clientForContext(contextName)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var out []resources.ResourceItem
	switch key {
	case "pods":
		out, err = k.listPods(ctx, client, namespace)
	case "services":
		out, err = k.listServices(ctx, client, namespace)
	case "deployments":
		out, err = k.listDeployments(ctx, client, namespace)
	case "workloads":
		out, err = k.listDeployments(ctx, client, namespace)
		for i := range out {
			out[i].Kind = "DEP"
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrListNotSupported, resourceName)
	}
	if err != nil {
		return nil, err
	}
	k.listCacheSet(cacheKey, out)
	return out, nil
}

func (k *clientGoAPI) PodEvents(contextName, namespace, pod string) ([]string, error) {
	client, err := k.clientForContext(contextName)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	list, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("involvedObject.name", pod).String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing events for %s/%s: %w", namespace, pod, err)
	}
	if len(list.Items) == 0 {
		return []string{"—   No recent events"}, nil
	}

	sort.SliceStable(list.Items, func(i, j int) bool {
		return list.Items[i].LastTimestamp.Time.After(list.Items[j].LastTimestamp.Time)
	})
	out := make([]string, 0, len(list.Items))
	for _, ev := range list.Items {
		ts := ev.LastTimestamp.Time
		if ts.IsZero() {
			ts = ev.EventTime.Time
		}
		prefix := "—"
		if !ts.IsZero() {
			prefix = ts.UTC().Format(time.RFC3339)
		}
		evType := strings.TrimSpace(ev.Type)
		if evType == "" {
			evType = "Normal"
		}
		reason := strings.TrimSpace(ev.Reason)
		if reason == "" {
			reason = "Unknown"
		}
		msg := strings.TrimSpace(ev.Message)
		out = append(out, fmt.Sprintf("%s   %s   %s   %s", prefix, evType, reason, msg))
	}
	return out, nil
}

func (k *clientGoAPI) clientForContext(contextName string) (kubernetes.Interface, error) {
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&k.loader,
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	)
	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed kube client config for context %q: %w", contextName, err)
	}
	client, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("failed creating kube client for context %q: %w", contextName, err)
	}
	return client, nil
}

func (k *clientGoAPI) listPods(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, p := range list.Items {
		out = append(out, resources.ResourceItem{
			UID:       string(p.UID),
			Name:      p.Name,
			Namespace: p.Namespace,
			Status:    podStatus(p),
			Ready:     podReady(p),
			Restarts:  strconv.Itoa(totalRestarts(p)),
			Age:       ageString(p.CreationTimestamp.Time),
			Labels:    copyMap(p.Labels),
			Extra: map[string]string{
				"node":           p.Spec.NodeName,
				"ip":             p.Status.PodIP,
				"qos":            string(p.Status.QOSClass),
				"controlled-by":  podController(p),
				"nominated-node": p.Status.NominatedNodeName,
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listServices(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, s := range list.Items {
		selector := copyMap(s.Spec.Selector)
		endpoints := "0 endpoints"
		if len(selector) > 0 {
			endpoints = "1 endpoint"
		}
		externalIP := "<none>"
		if len(s.Spec.ExternalIPs) > 0 {
			externalIP = strings.Join(s.Spec.ExternalIPs, ",")
		}
		parts := make([]string, 0, len(selector))
		for k, v := range selector {
			parts = append(parts, k+"="+v)
		}
		sort.Strings(parts)
		out = append(out, resources.ResourceItem{
			UID:       string(s.UID),
			Name:      s.Name,
			Namespace: s.Namespace,
			Kind:      string(s.Spec.Type),
			Status:    "Healthy",
			Ready:     endpoints,
			Age:       ageString(s.CreationTimestamp.Time),
			Selector:  selector,
			Extra: map[string]string{
				"external-ip": externalIP,
				"selector":    strings.Join(parts, ","),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listDeployments(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, d := range list.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		out = append(out, resources.ResourceItem{
			UID:       string(d.UID),
			Name:      d.Name,
			Namespace: d.Namespace,
			Status:    deploymentStatus(d),
			Ready:     strconv.Itoa(int(d.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired)),
			Age:       ageString(d.CreationTimestamp.Time),
			Selector:  copyLabelSelector(d.Spec.Selector.MatchLabels),
			Extra: map[string]string{
				"selector":   labelSelectorString(d.Spec.Selector.MatchLabels),
				"strategy":   string(d.Spec.Strategy.Type),
				"containers": containerNames(d.Spec.Template.Spec.Containers),
				"images":     containerImages(d.Spec.Template.Spec.Containers),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func deploymentStatus(d appsv1.Deployment) string {
	if d.Status.UnavailableReplicas > 0 {
		if d.Status.ReadyReplicas == 0 {
			return "Degraded"
		}
		return "Progressing"
	}
	return "Healthy"
}

func podStatus(p corev1.Pod) string {
	switch p.Status.Phase {
	case corev1.PodRunning:
		ready, total := podReadyCounts(p)
		if ready < total {
			return "NotReady"
		}
		return "Running"
	case corev1.PodPending:
		return "Pending"
	case corev1.PodFailed:
		return "Failed"
	case corev1.PodSucceeded:
		return "Succeeded"
	default:
		return "Unknown"
	}
}

func podReady(p corev1.Pod) string {
	ready, total := podReadyCounts(p)
	return strconv.Itoa(ready) + "/" + strconv.Itoa(total)
}

func podReadyCounts(p corev1.Pod) (ready, total int) {
	total = len(p.Spec.Containers)
	if total == 0 {
		total = len(p.Status.ContainerStatuses)
	}
	for _, c := range p.Status.ContainerStatuses {
		if c.Ready {
			ready++
		}
	}
	return ready, total
}

func totalRestarts(p corev1.Pod) int {
	total := 0
	for _, c := range p.Status.ContainerStatuses {
		total += int(c.RestartCount)
	}
	return total
}

func podController(p corev1.Pod) string {
	for _, ref := range p.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return string(ref.Kind) + "/" + ref.Name
		}
	}
	return ""
}

func ageString(created time.Time) string {
	if created.IsZero() {
		return "0m"
	}
	d := time.Since(created)
	if d < time.Hour {
		return strconv.Itoa(int(d.Minutes())) + "m"
	}
	if d < 24*time.Hour {
		return strconv.Itoa(int(d.Hours())) + "h"
	}
	return strconv.Itoa(int(d.Hours()/24)) + "d"
}

func copyMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func copyLabelSelector(src map[string]string) map[string]string {
	return copyMap(src)
}

func labelSelectorString(selector map[string]string) string {
	if len(selector) == 0 {
		return ""
	}
	parts := make([]string, 0, len(selector))
	for k, v := range selector {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func containerNames(containers []corev1.Container) string {
	names := make([]string, 0, len(containers))
	for _, c := range containers {
		names = append(names, c.Name)
	}
	return strings.Join(names, ",")
}

func containerImages(containers []corev1.Container) string {
	images := make([]string, 0, len(containers))
	for _, c := range containers {
		images = append(images, c.Image)
	}
	return strings.Join(images, ",")
}

func (k *clientGoAPI) namespaceCacheGet(contextName string) ([]string, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	entry, ok := k.ns[contextName]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	out := make([]string, len(entry.items))
	copy(out, entry.items)
	return out, true
}

func (k *clientGoAPI) namespaceCacheSet(contextName string, items []string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]string, len(items))
	copy(out, items)
	k.ns[contextName] = namespaceCacheEntry{
		items:     out,
		expiresAt: time.Now().Add(k.nsTTL),
	}
}

func (k *clientGoAPI) listCacheGet(cacheKey string) ([]resources.ResourceItem, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	entry, ok := k.list[cacheKey]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	out := make([]resources.ResourceItem, len(entry.items))
	copy(out, entry.items)
	return out, true
}

func (k *clientGoAPI) listCacheSet(cacheKey string, items []resources.ResourceItem) {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]resources.ResourceItem, len(items))
	copy(out, items)
	k.list[cacheKey] = listCacheEntry{
		items:     out,
		expiresAt: time.Now().Add(k.listTTL),
	}
}
