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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type clientGoAPI struct {
	loader  clientcmd.ClientConfigLoadingRules
	mu      sync.Mutex
	nsTTL   time.Duration
	ns      map[string]namespaceCacheEntry
	listTTL time.Duration
	list    map[string]listCacheEntry

	infMu sync.Mutex
	inf   map[string]*contextInformers
}

type namespaceCacheEntry struct {
	items     []string
	expiresAt time.Time
}

type listCacheEntry struct {
	items     []resources.ResourceItem
	expiresAt time.Time
}

type contextInformers struct {
	factory      informers.SharedInformerFactory
	stopCh       chan struct{}
	started      bool
	synced       bool
	lastSyncTry  time.Time
	pods         corelisters.PodLister
	services     corelisters.ServiceLister
	deployments  appslisters.DeploymentLister
	statefulSets appslisters.StatefulSetLister
	daemonSets   appslisters.DaemonSetLister
	jobs         batchlisters.JobLister
	cronJobs     batchlisters.CronJobLister
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
		inf:     map[string]*contextInformers{},
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

func (k *clientGoAPI) ListResources(contextName, namespace, resourceName string) ([]resources.ResourceItem, error) {
	key := strings.ToLower(strings.TrimSpace(resourceName))
	cacheKey := contextName + "|" + namespace + "|" + key
	if cached, ok := k.listCacheGet(cacheKey); ok {
		return cached, nil
	}

	var (
		out    []resources.ResourceItem
		err    error
		client kubernetes.Interface
	)

	switch key {
	case "contexts":
		out, err = k.listContexts()
	case "namespaces":
		out, err = k.listNamespaces(contextName)
	default:
		client, err = k.clientForContext(contextName)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		switch key {
		case "pods":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				out, err = k.listPodsFromInformer(inf, namespace)
				break
			}
			out, err = k.listPods(ctx, client, namespace)
		case "services":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				out, err = k.listServicesFromInformer(inf, namespace)
				break
			}
			out, err = k.listServices(ctx, client, namespace)
		case "deployments":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				out, err = k.listDeploymentsFromInformer(inf, namespace)
				break
			}
			out, err = k.listDeployments(ctx, client, namespace)
		case "workloads":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				out, err = k.listWorkloadsFromInformer(inf, namespace)
				break
			}
			out, err = k.listWorkloads(ctx, client, namespace)
		case "ingresses":
			out, err = k.listIngresses(ctx, client, namespace)
		case "configmaps":
			out, err = k.listConfigMaps(ctx, client, namespace)
		case "secrets":
			out, err = k.listSecrets(ctx, client, namespace)
		case "persistentvolumeclaims":
			out, err = k.listPVCs(ctx, client, namespace)
		case "nodes":
			out, err = k.listNodes(ctx, client)
		case "events":
			out, err = k.listEvents(ctx, client, namespace)
		default:
			return nil, fmt.Errorf("%w: %s", ErrListNotSupported, resourceName)
		}
	}

	if err != nil {
		return nil, err
	}
	k.listCacheSet(cacheKey, out)
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

	req := client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{TailLines: &tail64})
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
		return eventTime(list.Items[i]).After(eventTime(list.Items[j]))
	})
	out := make([]string, 0, len(list.Items))
	for _, ev := range list.Items {
		prefix := "—"
		ts := eventTime(ev)
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
				"selector":    labelSelectorString(selector),
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
			Selector:  copyMap(d.Spec.Selector.MatchLabels),
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

func (k *clientGoAPI) listWorkloads(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	var out []resources.ResourceItem

	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments for workloads in %q: %w", namespace, err)
	}
	for _, d := range deployments.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		out = append(out, resources.ResourceItem{
			UID:       string(d.UID),
			Name:      d.Name,
			Namespace: d.Namespace,
			Kind:      "DEP",
			Status:    deploymentStatus(d),
			Ready:     strconv.Itoa(int(d.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired)),
			Restarts:  "0",
			Age:       ageString(d.CreationTimestamp.Time),
			Selector:  copyMap(d.Spec.Selector.MatchLabels),
			Extra: map[string]string{
				"selector":   labelSelectorString(d.Spec.Selector.MatchLabels),
				"strategy":   string(d.Spec.Strategy.Type),
				"containers": containerNames(d.Spec.Template.Spec.Containers),
				"images":     containerImages(d.Spec.Template.Spec.Containers),
			},
		})
	}

	stsList, err := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets for workloads in %q: %w", namespace, err)
	}
	for _, s := range stsList.Items {
		desired := int32(1)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		status := "Healthy"
		if s.Status.ReadyReplicas < desired {
			if s.Status.ReadyReplicas == 0 {
				status = "Degraded"
			} else {
				status = "Progressing"
			}
		}
		out = append(out, resources.ResourceItem{
			UID:       string(s.UID),
			Name:      s.Name,
			Namespace: s.Namespace,
			Kind:      "STS",
			Status:    status,
			Ready:     strconv.Itoa(int(s.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired)),
			Restarts:  "0",
			Age:       ageString(s.CreationTimestamp.Time),
			Selector:  copyMap(s.Spec.Selector.MatchLabels),
			Extra: map[string]string{
				"selector":   labelSelectorString(s.Spec.Selector.MatchLabels),
				"strategy":   string(s.Spec.UpdateStrategy.Type),
				"containers": containerNames(s.Spec.Template.Spec.Containers),
				"images":     containerImages(s.Spec.Template.Spec.Containers),
			},
		})
	}

	dsList, err := client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets for workloads in %q: %w", namespace, err)
	}
	for _, d := range dsList.Items {
		desired := d.Status.DesiredNumberScheduled
		ready := d.Status.NumberReady
		status := "Healthy"
		if ready < desired {
			if ready == 0 {
				status = "Degraded"
			} else {
				status = "Progressing"
			}
		}
		out = append(out, resources.ResourceItem{
			UID:       string(d.UID),
			Name:      d.Name,
			Namespace: d.Namespace,
			Kind:      "DS",
			Status:    status,
			Ready:     strconv.Itoa(int(ready)) + "/" + strconv.Itoa(int(desired)),
			Restarts:  "0",
			Age:       ageString(d.CreationTimestamp.Time),
			Selector:  copyMap(d.Spec.Selector.MatchLabels),
			Extra: map[string]string{
				"selector":   labelSelectorString(d.Spec.Selector.MatchLabels),
				"strategy":   string(d.Spec.UpdateStrategy.Type),
				"containers": containerNames(d.Spec.Template.Spec.Containers),
				"images":     containerImages(d.Spec.Template.Spec.Containers),
			},
		})
	}

	jobs, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs for workloads in %q: %w", namespace, err)
	}
	for _, j := range jobs.Items {
		completions := int32(1)
		if j.Spec.Completions != nil {
			completions = *j.Spec.Completions
		}
		status := "Healthy"
		if j.Status.Failed > 0 {
			status = "Failed"
		} else if j.Status.Succeeded < completions {
			status = "Progressing"
		}
		out = append(out, resources.ResourceItem{
			UID:       string(j.UID),
			Name:      j.Name,
			Namespace: j.Namespace,
			Kind:      "JOB",
			Status:    status,
			Ready:     strconv.Itoa(int(j.Status.Succeeded)) + "/" + strconv.Itoa(int(completions)),
			Restarts:  strconv.Itoa(int(j.Status.Failed)),
			Age:       ageString(j.CreationTimestamp.Time),
		})
	}

	cronJobs, err := client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cronjobs for workloads in %q: %w", namespace, err)
	}
	for _, cj := range cronJobs.Items {
		ready := "Last: —"
		if cj.Status.LastScheduleTime != nil {
			ready = "Last: " + ageString(cj.Status.LastScheduleTime.Time)
		}
		status := "Healthy"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			status = "Suspended"
		}
		out = append(out, resources.ResourceItem{
			UID:       string(cj.UID),
			Name:      cj.Name,
			Namespace: cj.Namespace,
			Kind:      "CJ",
			Status:    status,
			Ready:     ready,
			Restarts:  "—",
			Age:       ageString(cj.CreationTimestamp.Time),
		})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listIngresses(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, ing := range list.Items {
		class := "nginx"
		if ing.Spec.IngressClassName != nil && *ing.Spec.IngressClassName != "" {
			class = *ing.Spec.IngressClassName
		}
		status := "Healthy"
		if len(ing.Status.LoadBalancer.Ingress) == 0 {
			status = "Pending"
		}
		tls := "False"
		if len(ing.Spec.TLS) > 0 {
			tls = "True"
		}
		backendServices := ingressBackendServices(ing)
		out = append(out, resources.ResourceItem{
			UID:       string(ing.UID),
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Kind:      class,
			Status:    status,
			Ready:     ingressHosts(ing.Spec.Rules),
			Age:       ageString(ing.CreationTimestamp.Time),
			Extra: map[string]string{
				"tls":      tls,
				"rules":    strconv.Itoa(len(ing.Spec.Rules)),
				"services": strings.Join(backendServices, ","),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ingressBackendServices(ing networkingv1.Ingress) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		add(ing.Spec.DefaultBackend.Service.Name)
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			if p.Backend.Service != nil {
				add(p.Backend.Service.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (k *clientGoAPI) listConfigMaps(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, cm := range list.Items {
		managedBy := cm.Labels["app.kubernetes.io/managed-by"]
		if managedBy == "" {
			managedBy = "unknown"
		}
		binaryData := strconv.Itoa(len(cm.BinaryData))
		out = append(out, resources.ResourceItem{
			UID:       string(cm.UID),
			Name:      cm.Name,
			Namespace: cm.Namespace,
			Status:    "Healthy",
			Age:       ageString(cm.CreationTimestamp.Time),
			Extra: map[string]string{
				"managed-by":  managedBy,
				"binary-data": binaryData,
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listSecrets(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, sec := range list.Items {
		out = append(out, resources.ResourceItem{
			UID:       string(sec.UID),
			Name:      sec.Name,
			Namespace: sec.Namespace,
			Kind:      string(sec.Type),
			Status:    "Healthy",
			Age:       ageString(sec.CreationTimestamp.Time),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listPVCs(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pvc for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, pvc := range list.Items {
		capacity := "-"
		if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = q.String()
		} else if q, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			capacity = q.String()
		}
		access := "RWO"
		if len(pvc.Spec.AccessModes) > 0 {
			access = string(pvc.Spec.AccessModes[0])
		}
		out = append(out, resources.ResourceItem{
			UID:       string(pvc.UID),
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
			Kind:      access,
			Status:    string(pvc.Status.Phase),
			Ready:     capacity,
			Age:       ageString(pvc.CreationTimestamp.Time),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listNodes(ctx context.Context, client kubernetes.Interface) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, n := range list.Items {
		out = append(out, resources.ResourceItem{
			UID:    string(n.UID),
			Name:   n.Name,
			Status: nodeReadyStatus(n),
			Ready:  nodePodsCapacity(n),
			Age:    ageString(n.CreationTimestamp.Time),
			Extra: map[string]string{
				"internal-ip":    nodeAddress(n.Status.Addresses, corev1.NodeInternalIP),
				"os":             n.Status.NodeInfo.OperatingSystem,
				"arch":           n.Status.NodeInfo.Architecture,
				"kernel-version": n.Status.NodeInfo.KernelVersion,
				"runtime":        n.Status.NodeInfo.ContainerRuntimeVersion,
				"instance-type":  nodeLabel(n.Labels, "node.kubernetes.io/instance-type"),
				"zone":           nodeLabel(n.Labels, "topology.kubernetes.io/zone"),
				"taints":         strconv.Itoa(len(n.Spec.Taints)),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listEvents(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list events for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, ev := range list.Items {
		status := "Healthy"
		if strings.EqualFold(ev.Type, "Warning") {
			status = "Warning"
		}
		out = append(out, resources.ResourceItem{
			UID:       string(ev.UID),
			Name:      ev.InvolvedObject.Name + "." + ev.Reason,
			Namespace: ev.Namespace,
			Kind:      ev.Type,
			Status:    status,
			Age:       ageString(eventTime(ev)),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listNamespaces(contextName string) ([]resources.ResourceItem, error) {
	names, err := k.Namespaces(contextName)
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(names))
	for _, n := range names {
		out = append(out, resources.ResourceItem{Name: n, Status: "Active", Age: "?"})
	}
	return out, nil
}

func (k *clientGoAPI) listContexts() ([]resources.ResourceItem, error) {
	names, err := k.Contexts()
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(names))
	for _, n := range names {
		out = append(out, resources.ResourceItem{Name: n, Status: "Available", Age: "?"})
	}
	return out, nil
}

func (k *clientGoAPI) ensureInformers(contextName string, client kubernetes.Interface) *contextInformers {
	k.infMu.Lock()
	inf := k.inf[contextName]
	if inf == nil {
		factory := informers.NewSharedInformerFactory(client, 2*time.Minute)
		inf = &contextInformers{
			factory:      factory,
			stopCh:       make(chan struct{}),
			pods:         factory.Core().V1().Pods().Lister(),
			services:     factory.Core().V1().Services().Lister(),
			deployments:  factory.Apps().V1().Deployments().Lister(),
			statefulSets: factory.Apps().V1().StatefulSets().Lister(),
			daemonSets:   factory.Apps().V1().DaemonSets().Lister(),
			jobs:         factory.Batch().V1().Jobs().Lister(),
			cronJobs:     factory.Batch().V1().CronJobs().Lister(),
		}
		k.inf[contextName] = inf
	}
	if !inf.started {
		inf.factory.Start(inf.stopCh)
		inf.started = true
	}
	shouldTrySync := !inf.synced && time.Since(inf.lastSyncTry) > 500*time.Millisecond
	inf.lastSyncTry = time.Now()
	k.infMu.Unlock()

	if shouldTrySync {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if inf.factory.Core().V1().Pods().Informer().HasSynced() &&
				inf.factory.Core().V1().Services().Informer().HasSynced() &&
				inf.factory.Apps().V1().Deployments().Informer().HasSynced() &&
				inf.factory.Apps().V1().StatefulSets().Informer().HasSynced() &&
				inf.factory.Apps().V1().DaemonSets().Informer().HasSynced() &&
				inf.factory.Batch().V1().Jobs().Informer().HasSynced() &&
				inf.factory.Batch().V1().CronJobs().Informer().HasSynced() {
				k.infMu.Lock()
				inf.synced = true
				k.infMu.Unlock()
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	return inf
}

func (k *clientGoAPI) listPodsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		pods []*corev1.Pod
		err  error
	)
	if namespace == resources.AllNamespaces {
		pods, err = inf.pods.List(labels.Everything())
	} else {
		pods, err = inf.pods.Pods(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(pods))
	for _, p := range pods {
		out = append(out, resources.ResourceItem{
			UID:       string(p.UID),
			Name:      p.Name,
			Namespace: p.Namespace,
			Status:    podStatus(*p),
			Ready:     podReady(*p),
			Restarts:  strconv.Itoa(totalRestarts(*p)),
			Age:       ageString(p.CreationTimestamp.Time),
			Labels:    copyMap(p.Labels),
			Extra: map[string]string{
				"node":           p.Spec.NodeName,
				"ip":             p.Status.PodIP,
				"qos":            string(p.Status.QOSClass),
				"controlled-by":  podController(*p),
				"nominated-node": p.Status.NominatedNodeName,
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listServicesFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		services []*corev1.Service
		err      error
	)
	if namespace == resources.AllNamespaces {
		services, err = inf.services.List(labels.Everything())
	} else {
		services, err = inf.services.Services(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(services))
	for _, s := range services {
		selector := copyMap(s.Spec.Selector)
		endpoints := "0 endpoints"
		if len(selector) > 0 {
			endpoints = "1 endpoint"
		}
		externalIP := "<none>"
		if len(s.Spec.ExternalIPs) > 0 {
			externalIP = strings.Join(s.Spec.ExternalIPs, ",")
		}
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
				"selector":    labelSelectorString(selector),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listDeploymentsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		deployments []*appsv1.Deployment
		err         error
	)
	if namespace == resources.AllNamespaces {
		deployments, err = inf.deployments.List(labels.Everything())
	} else {
		deployments, err = inf.deployments.Deployments(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(deployments))
	for _, d := range deployments {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		out = append(out, resources.ResourceItem{
			UID:       string(d.UID),
			Name:      d.Name,
			Namespace: d.Namespace,
			Status:    deploymentStatus(*d),
			Ready:     strconv.Itoa(int(d.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired)),
			Age:       ageString(d.CreationTimestamp.Time),
			Selector:  copyMap(d.Spec.Selector.MatchLabels),
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

func (k *clientGoAPI) listWorkloadsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	out := make([]resources.ResourceItem, 0)
	deployments, err := k.listDeploymentsFromInformer(inf, namespace)
	if err != nil {
		return nil, err
	}
	for _, d := range deployments {
		d.Kind = "DEP"
		d.Restarts = "0"
		out = append(out, d)
	}

	var statefulSets []*appsv1.StatefulSet
	if namespace == resources.AllNamespaces {
		statefulSets, err = inf.statefulSets.List(labels.Everything())
	} else {
		statefulSets, err = inf.statefulSets.StatefulSets(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	for _, s := range statefulSets {
		desired := int32(1)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		status := "Healthy"
		if s.Status.ReadyReplicas < desired {
			if s.Status.ReadyReplicas == 0 {
				status = "Degraded"
			} else {
				status = "Progressing"
			}
		}
		out = append(out, resources.ResourceItem{
			UID:       string(s.UID),
			Name:      s.Name,
			Namespace: s.Namespace,
			Kind:      "STS",
			Status:    status,
			Ready:     strconv.Itoa(int(s.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired)),
			Restarts:  "0",
			Age:       ageString(s.CreationTimestamp.Time),
			Selector:  copyMap(s.Spec.Selector.MatchLabels),
			Extra: map[string]string{
				"selector":   labelSelectorString(s.Spec.Selector.MatchLabels),
				"strategy":   string(s.Spec.UpdateStrategy.Type),
				"containers": containerNames(s.Spec.Template.Spec.Containers),
				"images":     containerImages(s.Spec.Template.Spec.Containers),
			},
		})
	}

	var daemonSets []*appsv1.DaemonSet
	if namespace == resources.AllNamespaces {
		daemonSets, err = inf.daemonSets.List(labels.Everything())
	} else {
		daemonSets, err = inf.daemonSets.DaemonSets(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	for _, d := range daemonSets {
		desired := d.Status.DesiredNumberScheduled
		ready := d.Status.NumberReady
		status := "Healthy"
		if ready < desired {
			if ready == 0 {
				status = "Degraded"
			} else {
				status = "Progressing"
			}
		}
		out = append(out, resources.ResourceItem{
			UID:       string(d.UID),
			Name:      d.Name,
			Namespace: d.Namespace,
			Kind:      "DS",
			Status:    status,
			Ready:     strconv.Itoa(int(ready)) + "/" + strconv.Itoa(int(desired)),
			Restarts:  "0",
			Age:       ageString(d.CreationTimestamp.Time),
			Selector:  copyMap(d.Spec.Selector.MatchLabels),
			Extra: map[string]string{
				"selector":   labelSelectorString(d.Spec.Selector.MatchLabels),
				"strategy":   string(d.Spec.UpdateStrategy.Type),
				"containers": containerNames(d.Spec.Template.Spec.Containers),
				"images":     containerImages(d.Spec.Template.Spec.Containers),
			},
		})
	}

	var jobs []*batchv1.Job
	if namespace == resources.AllNamespaces {
		jobs, err = inf.jobs.List(labels.Everything())
	} else {
		jobs, err = inf.jobs.Jobs(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		completions := int32(1)
		if j.Spec.Completions != nil {
			completions = *j.Spec.Completions
		}
		status := "Healthy"
		if j.Status.Failed > 0 {
			status = "Failed"
		} else if j.Status.Succeeded < completions {
			status = "Progressing"
		}
		out = append(out, resources.ResourceItem{
			UID:       string(j.UID),
			Name:      j.Name,
			Namespace: j.Namespace,
			Kind:      "JOB",
			Status:    status,
			Ready:     strconv.Itoa(int(j.Status.Succeeded)) + "/" + strconv.Itoa(int(completions)),
			Restarts:  strconv.Itoa(int(j.Status.Failed)),
			Age:       ageString(j.CreationTimestamp.Time),
		})
	}

	var cronJobs []*batchv1.CronJob
	if namespace == resources.AllNamespaces {
		cronJobs, err = inf.cronJobs.List(labels.Everything())
	} else {
		cronJobs, err = inf.cronJobs.CronJobs(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	for _, cj := range cronJobs {
		ready := "Last: —"
		if cj.Status.LastScheduleTime != nil {
			ready = "Last: " + ageString(cj.Status.LastScheduleTime.Time)
		}
		status := "Healthy"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			status = "Suspended"
		}
		out = append(out, resources.ResourceItem{
			UID:       string(cj.UID),
			Name:      cj.Name,
			Namespace: cj.Namespace,
			Kind:      "CJ",
			Status:    status,
			Ready:     ready,
			Restarts:  "—",
			Age:       ageString(cj.CreationTimestamp.Time),
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

func ingressHosts(rules []networkingv1.IngressRule) string {
	hosts := make([]string, 0, len(rules))
	for _, r := range rules {
		if strings.TrimSpace(r.Host) != "" {
			hosts = append(hosts, r.Host)
		}
	}
	if len(hosts) == 0 {
		return "*"
	}
	sort.Strings(hosts)
	return strings.Join(hosts, ",")
}

func nodeReadyStatus(n corev1.Node) string {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			if c.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func nodePodsCapacity(n corev1.Node) string {
	pods := n.Status.Capacity.Pods().Value()
	return "0/" + strconv.FormatInt(pods, 10)
}

func nodeAddress(addrs []corev1.NodeAddress, kind corev1.NodeAddressType) string {
	for _, a := range addrs {
		if a.Type == kind {
			return a.Address
		}
	}
	return ""
}

func nodeLabel(labels map[string]string, key string) string {
	if labels == nil {
		return ""
	}
	return labels[key]
}

func eventTime(ev corev1.Event) time.Time {
	if !ev.LastTimestamp.IsZero() {
		return ev.LastTimestamp.Time
	}
	if !ev.EventTime.IsZero() {
		return ev.EventTime.Time
	}
	return ev.CreationTimestamp.Time
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
	k.ns[contextName] = namespaceCacheEntry{items: out, expiresAt: time.Now().Add(k.nsTTL)}
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
	k.list[cacheKey] = listCacheEntry{items: out, expiresAt: time.Now().Add(k.listTTL)}
}
