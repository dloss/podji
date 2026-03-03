package data

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type clientGoAPI struct {
	loader clientcmd.ClientConfigLoadingRules
	mu     sync.Mutex
	nsTTL  time.Duration
	ns     map[string]namespaceCacheEntry
}

type namespaceCacheEntry struct {
	items     []string
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
		loader: loader,
		nsTTL:  5 * time.Second,
		ns:     map[string]namespaceCacheEntry{},
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
