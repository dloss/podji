package data

import (
	"bufio"
	"context"
	"encoding/json"
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
	networkinglisters "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

const maxLogLines = 2000

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
	ingresses    networkinglisters.IngressLister
	configMaps   corelisters.ConfigMapLister
	secrets      corelisters.SecretLister
	pvcs         corelisters.PersistentVolumeClaimLister
	nodes        corelisters.NodeLister
	events       corelisters.EventLister
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
	out, _, err := k.ListResourcesMeta(contextName, namespace, resourceName)
	return out, err
}

func (k *clientGoAPI) ListResourcesMeta(contextName, namespace, resourceName string) ([]resources.ResourceItem, bool, error) {
	key := strings.ToLower(strings.TrimSpace(resourceName))
	cacheKey := contextName + "|" + namespace + "|" + key
	if cached, ok := k.listCacheGet(cacheKey); ok {
		return cached, true, nil
	}

	var (
		out         []resources.ResourceItem
		err         error
		client      kubernetes.Interface
		cacheBacked bool
	)

	switch key {
	case "contexts":
		out, err = k.listContexts()
	case "namespaces":
		out, err = k.listNamespaces(contextName)
	default:
		client, err = k.clientForContext(contextName)
		if err != nil {
			return nil, false, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		switch key {
		case "pods":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listPodsFromInformer(inf, namespace)
				break
			}
			out, err = k.listPods(ctx, client, namespace)
		case "services":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listServicesFromInformer(inf, namespace)
				break
			}
			out, err = k.listServices(ctx, client, namespace)
		case "deployments":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listDeploymentsFromInformer(inf, namespace)
				break
			}
			out, err = k.listDeployments(ctx, client, namespace)
		case "workloads":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listWorkloadsFromInformer(inf, namespace)
				break
			}
			out, err = k.listWorkloads(ctx, client, namespace)
		case "ingresses":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listIngressesFromInformer(inf, namespace)
				break
			}
			out, err = k.listIngresses(ctx, client, namespace)
		case "configmaps":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listConfigMapsFromInformer(inf, namespace)
				break
			}
			out, err = k.listConfigMaps(ctx, client, namespace)
		case "secrets":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listSecretsFromInformer(inf, namespace)
				break
			}
			out, err = k.listSecrets(ctx, client, namespace)
		case "persistentvolumeclaims":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listPVCsFromInformer(inf, namespace)
				break
			}
			out, err = k.listPVCs(ctx, client, namespace)
		case "nodes":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listNodesFromInformer(inf)
				break
			}
			out, err = k.listNodes(ctx, client)
		case "events":
			if inf := k.ensureInformers(contextName, client); inf != nil && inf.synced {
				cacheBacked = true
				out, err = k.listEventsFromInformer(inf, namespace)
				break
			}
			out, err = k.listEvents(ctx, client, namespace)
		default:
			return nil, false, fmt.Errorf("%w: %s", ErrListNotSupported, resourceName)
		}
	}

	if err != nil {
		return nil, false, err
	}
	k.listCacheSet(cacheKey, out)
	return out, cacheBacked, nil
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

	lines, err := boundedNonEmptyLines(stream, maxLogLines)
	if err != nil {
		return nil, fmt.Errorf("failed reading logs for %s/%s: %w", namespace, pod, err)
	}
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

func (k *clientGoAPI) ResourceYAML(contextName, namespace, resourceName string, item resources.ResourceItem) (string, error) {
	obj, err := k.resourceObject(contextName, namespace, resourceName, item)
	if err != nil {
		return "", err
	}
	return marshalKubeObjectYAML(obj, resourceName, item)
}

func (k *clientGoAPI) ResourceDetail(contextName, namespace, resourceName string, item resources.ResourceItem) (resources.DetailData, error) {
	obj, err := k.resourceObject(contextName, namespace, resourceName, item)
	if err != nil {
		return resources.DetailData{}, err
	}
	return detailFromObject(obj, resourceName, item), nil
}

func (k *clientGoAPI) ResourceDescribe(contextName, namespace, resourceName string, item resources.ResourceItem) (string, error) {
	obj, err := k.resourceObject(contextName, namespace, resourceName, item)
	if err != nil {
		return "", err
	}
	return describeKubeObject(obj, resourceName, item, namespace), nil
}

func (k *clientGoAPI) resourceObject(contextName, namespace, resourceName string, item resources.ResourceItem) (any, error) {
	client, err := k.clientForContext(contextName)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	name := strings.TrimSpace(item.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: missing resource name", ErrObjectReadNotSupported)
	}
	ns := strings.TrimSpace(item.Namespace)
	if ns == "" {
		ns = strings.TrimSpace(namespace)
	}
	key := strings.ToLower(strings.TrimSpace(resourceName))
	switch key {
	case "pods":
		return client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	case "services":
		return client.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	case "deployments":
		return client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	case "workloads":
		return workloadObject(ctx, client, ns, name, item.Kind)
	case "ingresses":
		return client.NetworkingV1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
	case "configmaps":
		return client.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	case "secrets":
		return client.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	case "persistentvolumeclaims":
		return client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
	case "nodes":
		return client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	case "namespaces":
		return client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	case "events":
		return client.CoreV1().Events(ns).Get(ctx, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("%w: %s", ErrObjectReadNotSupported, resourceName)
	}
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
	list, err := client.CoreV1().Pods(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for %q: %w", namespace, err)
	}
	out := make([]resources.ResourceItem, 0, len(list.Items))
	for _, p := range list.Items {
		controllerKind, controllerName, controllerUID := podControllerRef(p)
		configRefs, secretRefs, pvcRefs := podRefs(p.Spec)
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
				"node":              p.Spec.NodeName,
				"ip":                p.Status.PodIP,
				"qos":               string(p.Status.QOSClass),
				"controlled-by":     controllerRefString(controllerKind, controllerName),
				"controlled-by-uid": controllerUID,
				"nominated-node":    p.Status.NominatedNodeName,
				"containers":        containerNames(p.Spec.Containers),
				"images":            containerImages(p.Spec.Containers),
				"config-refs":       strings.Join(configRefs, ","),
				"secret-refs":       strings.Join(secretRefs, ","),
				"pvc-refs":          strings.Join(pvcRefs, ","),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (k *clientGoAPI) listServices(ctx context.Context, client kubernetes.Interface, namespace string) ([]resources.ResourceItem, error) {
	list, err := client.CoreV1().Services(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
	list, err := client.AppsV1().Deployments(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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

	deployments, err := client.AppsV1().Deployments(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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

	stsList, err := client.AppsV1().StatefulSets(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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

	dsList, err := client.AppsV1().DaemonSets(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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

	jobs, err := client.BatchV1().Jobs(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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

	cronJobs, err := client.BatchV1().CronJobs(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
	list, err := client.NetworkingV1().Ingresses(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
	list, err := client.CoreV1().ConfigMaps(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
	list, err := client.CoreV1().Secrets(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
	list, err := client.CoreV1().PersistentVolumeClaims(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
	list, err := client.CoreV1().Events(apiNamespace(namespace)).List(ctx, metav1.ListOptions{})
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
			ingresses:    factory.Networking().V1().Ingresses().Lister(),
			configMaps:   factory.Core().V1().ConfigMaps().Lister(),
			secrets:      factory.Core().V1().Secrets().Lister(),
			pvcs:         factory.Core().V1().PersistentVolumeClaims().Lister(),
			nodes:        factory.Core().V1().Nodes().Lister(),
			events:       factory.Core().V1().Events().Lister(),
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
				inf.factory.Batch().V1().CronJobs().Informer().HasSynced() &&
				inf.factory.Networking().V1().Ingresses().Informer().HasSynced() &&
				inf.factory.Core().V1().ConfigMaps().Informer().HasSynced() &&
				inf.factory.Core().V1().Secrets().Informer().HasSynced() &&
				inf.factory.Core().V1().PersistentVolumeClaims().Informer().HasSynced() &&
				inf.factory.Core().V1().Nodes().Informer().HasSynced() &&
				inf.factory.Core().V1().Events().Informer().HasSynced() {
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
		controllerKind, controllerName, controllerUID := podControllerRef(*p)
		configRefs, secretRefs, pvcRefs := podRefs(p.Spec)
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
				"node":              p.Spec.NodeName,
				"ip":                p.Status.PodIP,
				"qos":               string(p.Status.QOSClass),
				"controlled-by":     controllerRefString(controllerKind, controllerName),
				"controlled-by-uid": controllerUID,
				"nominated-node":    p.Status.NominatedNodeName,
				"containers":        containerNames(p.Spec.Containers),
				"images":            containerImages(p.Spec.Containers),
				"config-refs":       strings.Join(configRefs, ","),
				"secret-refs":       strings.Join(secretRefs, ","),
				"pvc-refs":          strings.Join(pvcRefs, ","),
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

func (k *clientGoAPI) listIngressesFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		ingresses []*networkingv1.Ingress
		err       error
	)
	if namespace == resources.AllNamespaces {
		ingresses, err = inf.ingresses.List(labels.Everything())
	} else {
		ingresses, err = inf.ingresses.Ingresses(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(ingresses))
	for _, ing := range ingresses {
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
		backendServices := ingressBackendServices(*ing)
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

func (k *clientGoAPI) listConfigMapsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		configMaps []*corev1.ConfigMap
		err        error
	)
	if namespace == resources.AllNamespaces {
		configMaps, err = inf.configMaps.List(labels.Everything())
	} else {
		configMaps, err = inf.configMaps.ConfigMaps(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(configMaps))
	for _, cm := range configMaps {
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

func (k *clientGoAPI) listSecretsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		secrets []*corev1.Secret
		err     error
	)
	if namespace == resources.AllNamespaces {
		secrets, err = inf.secrets.List(labels.Everything())
	} else {
		secrets, err = inf.secrets.Secrets(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(secrets))
	for _, sec := range secrets {
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

func (k *clientGoAPI) listPVCsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		pvcs []*corev1.PersistentVolumeClaim
		err  error
	)
	if namespace == resources.AllNamespaces {
		pvcs, err = inf.pvcs.List(labels.Everything())
	} else {
		pvcs, err = inf.pvcs.PersistentVolumeClaims(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(pvcs))
	for _, pvc := range pvcs {
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

func (k *clientGoAPI) listNodesFromInformer(inf *contextInformers) ([]resources.ResourceItem, error) {
	nodes, err := inf.nodes.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, resources.ResourceItem{
			UID:    string(n.UID),
			Name:   n.Name,
			Status: nodeReadyStatus(*n),
			Ready:  nodePodsCapacity(*n),
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

func (k *clientGoAPI) listEventsFromInformer(inf *contextInformers, namespace string) ([]resources.ResourceItem, error) {
	var (
		events []*corev1.Event
		err    error
	)
	if namespace == resources.AllNamespaces {
		events, err = inf.events.List(labels.Everything())
	} else {
		events, err = inf.events.Events(namespace).List(labels.Everything())
	}
	if err != nil {
		return nil, err
	}
	out := make([]resources.ResourceItem, 0, len(events))
	for _, ev := range events {
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
			Age:       ageString(eventTime(*ev)),
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

func podControllerRef(p corev1.Pod) (kind string, name string, uid string) {
	for _, ref := range p.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return string(ref.Kind), ref.Name, string(ref.UID)
		}
	}
	return "", "", ""
}

func controllerRefString(kind, name string) string {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	if kind == "" || name == "" {
		return ""
	}
	return kind + "/" + name
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

func apiNamespace(namespace string) string {
	if namespace == resources.AllNamespaces {
		return metav1.NamespaceAll
	}
	return namespace
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

func formatNodeConditions(conditions []corev1.NodeCondition) []string {
	out := make([]string, 0, len(conditions))
	for _, c := range conditions {
		out = append(out, fmt.Sprintf("%s: %s", c.Type, c.Status))
	}
	return out
}

func nodeAddressLines(addrs []corev1.NodeAddress) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, fmt.Sprintf("%s: %s", a.Type, a.Address))
	}
	return out
}

func nodeLabel(labels map[string]string, key string) string {
	if labels == nil {
		return ""
	}
	return labels[key]
}

func podRefs(spec corev1.PodSpec) (configRefs []string, secretRefs []string, pvcRefs []string) {
	seenCfg := map[string]bool{}
	seenSec := map[string]bool{}
	seenPVC := map[string]bool{}
	add := func(name string, seen map[string]bool, dest *[]string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		*dest = append(*dest, name)
	}

	for _, v := range spec.Volumes {
		if v.ConfigMap != nil {
			add(v.ConfigMap.Name, seenCfg, &configRefs)
		}
		if v.Secret != nil {
			add(v.Secret.SecretName, seenSec, &secretRefs)
		}
		if v.PersistentVolumeClaim != nil {
			add(v.PersistentVolumeClaim.ClaimName, seenPVC, &pvcRefs)
		}
		if v.Projected != nil {
			for _, s := range v.Projected.Sources {
				if s.ConfigMap != nil {
					add(s.ConfigMap.Name, seenCfg, &configRefs)
				}
				if s.Secret != nil {
					add(s.Secret.Name, seenSec, &secretRefs)
				}
			}
		}
	}

	for _, c := range spec.Containers {
		for _, env := range c.Env {
			if env.ValueFrom == nil {
				continue
			}
			if env.ValueFrom.ConfigMapKeyRef != nil {
				add(env.ValueFrom.ConfigMapKeyRef.Name, seenCfg, &configRefs)
			}
			if env.ValueFrom.SecretKeyRef != nil {
				add(env.ValueFrom.SecretKeyRef.Name, seenSec, &secretRefs)
			}
		}
		for _, envFrom := range c.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				add(envFrom.ConfigMapRef.Name, seenCfg, &configRefs)
			}
			if envFrom.SecretRef != nil {
				add(envFrom.SecretRef.Name, seenSec, &secretRefs)
			}
		}
	}

	sort.Strings(configRefs)
	sort.Strings(secretRefs)
	sort.Strings(pvcRefs)
	return configRefs, secretRefs, pvcRefs
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

func workloadObject(ctx context.Context, client kubernetes.Interface, namespace, name, kind string) (any, error) {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "DEP", "DEPLOYMENT", "":
		return client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "STS", "STATEFULSET":
		return client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "DS", "DAEMONSET":
		return client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "JOB":
		return client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	case "CJ", "CRONJOB":
		return client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("%w: workload kind %q", ErrObjectReadNotSupported, kind)
	}
}

func marshalKubeObjectYAML(obj any, resourceName string, item resources.ResourceItem) (string, error) {
	raw, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed marshaling %s object: %w", resourceName, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("failed decoding %s object json: %w", resourceName, err)
	}
	if _, ok := doc["apiVersion"]; !ok {
		doc["apiVersion"] = valueOr(item.APIVersion, "v1")
	}
	if _, ok := doc["kind"]; !ok {
		doc["kind"] = valueOr(item.Kind, singularKindName(resourceName))
	}
	meta, _ := doc["metadata"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		doc["metadata"] = meta
	}
	if _, ok := meta["name"]; !ok && strings.TrimSpace(item.Name) != "" {
		meta["name"] = item.Name
	}
	if _, ok := meta["namespace"]; !ok && strings.TrimSpace(item.Namespace) != "" {
		meta["namespace"] = item.Namespace
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed marshaling %s yaml: %w", resourceName, err)
	}
	return string(out), nil
}

func describeKubeObject(obj any, resourceName string, item resources.ResourceItem, fallbackNamespace string) string {
	kind := valueOr(item.Kind, singularKindName(resourceName))
	name := valueOr(item.Name, "<unknown>")
	ns := valueOr(item.Namespace, fallbackNamespace)
	lines := []string{
		"Name:        " + name,
		"Namespace:   " + valueOr(ns, resources.DefaultNamespace),
		"Kind:        " + kind,
	}
	switch o := obj.(type) {
	case *corev1.Pod:
		lines = append(lines,
			"Status:      "+valueOr(string(o.Status.Phase), "Unknown"),
			"Node:        "+valueOr(o.Spec.NodeName, "<none>"),
			"IP:          "+valueOr(o.Status.PodIP, "<none>"),
			"Containers:  "+containerNames(o.Spec.Containers),
		)
	case *corev1.Service:
		ports := servicePortsString(o.Spec.Ports)
		externalIPs := "<none>"
		if len(o.Spec.ExternalIPs) > 0 {
			externalIPs = strings.Join(o.Spec.ExternalIPs, ",")
		}
		lines = append(lines,
			"Type:        "+valueOr(string(o.Spec.Type), "ClusterIP"),
			"ClusterIP:   "+valueOr(o.Spec.ClusterIP, "<none>"),
			"Selector:    "+valueOr(labelSelectorString(o.Spec.Selector), "<none>"),
			"Ports:       "+valueOr(ports, "<none>"),
			"External IP: "+externalIPs,
		)
	case *appsv1.Deployment:
		desired := int32(1)
		if o.Spec.Replicas != nil {
			desired = *o.Spec.Replicas
		}
		lines = append(lines,
			"Status:      "+deploymentStatus(*o),
			"Ready:       "+strconv.Itoa(int(o.Status.ReadyReplicas))+"/"+strconv.Itoa(int(desired)),
			"Strategy:    "+valueOr(string(o.Spec.Strategy.Type), "<none>"),
			"Images:      "+containerImages(o.Spec.Template.Spec.Containers),
		)
	case *appsv1.StatefulSet:
		desired := int32(1)
		if o.Spec.Replicas != nil {
			desired = *o.Spec.Replicas
		}
		lines = append(lines,
			"Status:      "+valueOr(item.Status, "Progressing"),
			"Ready:       "+strconv.Itoa(int(o.Status.ReadyReplicas))+"/"+strconv.Itoa(int(desired)),
			"Service:     "+valueOr(o.Spec.ServiceName, "<none>"),
			"Selector:    "+valueOr(labelSelectorString(o.Spec.Selector.MatchLabels), "<none>"),
			"Images:      "+containerImages(o.Spec.Template.Spec.Containers),
		)
	case *appsv1.DaemonSet:
		lines = append(lines,
			"Status:      "+valueOr(item.Status, "Healthy"),
			"Ready:       "+strconv.Itoa(int(o.Status.NumberReady))+"/"+strconv.Itoa(int(o.Status.DesiredNumberScheduled)),
			"Selector:    "+valueOr(labelSelectorString(o.Spec.Selector.MatchLabels), "<none>"),
			"Images:      "+containerImages(o.Spec.Template.Spec.Containers),
		)
	case *batchv1.Job:
		completions := int32(1)
		if o.Spec.Completions != nil {
			completions = *o.Spec.Completions
		}
		lines = append(lines,
			"Status:      "+valueOr(item.Status, "Running"),
			"Completions: "+strconv.Itoa(int(o.Status.Succeeded))+"/"+strconv.Itoa(int(completions)),
			"Parallelism: "+strconv.Itoa(int(ptrInt32(o.Spec.Parallelism, 1))),
			"Images:      "+containerImages(o.Spec.Template.Spec.Containers),
		)
	case *batchv1.CronJob:
		suspend := "false"
		if o.Spec.Suspend != nil && *o.Spec.Suspend {
			suspend = "true"
		}
		lines = append(lines,
			"Status:      Scheduled",
			"Schedule:    "+valueOr(o.Spec.Schedule, "<none>"),
			"Suspend:     "+suspend,
			"Images:      "+containerImages(o.Spec.JobTemplate.Spec.Template.Spec.Containers),
		)
	case *corev1.ConfigMap:
		lines = append(lines, "Data Keys:    "+strconv.Itoa(len(o.Data)))
	case *corev1.Secret:
		lines = append(lines,
			"Type:        "+valueOr(string(o.Type), "Opaque"),
			"Data Keys:    "+strconv.Itoa(len(o.Data)),
		)
	case *corev1.PersistentVolumeClaim:
		lines = append(lines,
			"Status:      "+valueOr(string(o.Status.Phase), "Unknown"),
			"Volume:      "+valueOr(o.Spec.VolumeName, "<none>"),
		)
	case *networkingv1.Ingress:
		services := ingressBackendServices(*o)
		servicesValue := "<none>"
		if len(services) > 0 {
			servicesValue = strings.Join(services, ",")
		}
		lines = append(lines,
			"Class:       "+valueOr(ptrString(o.Spec.IngressClassName), "<none>"),
			"Hosts:       "+valueOr(ingressHosts(o.Spec.Rules), "*"),
			"Services:    "+servicesValue,
			"Backends:",
		)
		for _, backend := range ingressBackendRoutes(*o) {
			lines = append(lines, "  "+backend)
		}
	case *corev1.Node:
		lines = append(lines,
			"Status:      "+valueOr(nodeReadyStatus(*o), "Unknown"),
			"Pod CIDR:    "+valueOr(o.Spec.PodCIDR, "<none>"),
		)
	case *corev1.Namespace:
		lines = append(lines, "Status:      "+valueOr(string(o.Status.Phase), "Unknown"))
	case *corev1.Event:
		ts := eventTime(*o)
		tsValue := "<none>"
		if !ts.IsZero() {
			tsValue = ts.UTC().Format(time.RFC3339)
		}
		source := "<none>"
		if strings.TrimSpace(o.Source.Component) != "" {
			source = o.Source.Component
		}
		if strings.TrimSpace(o.Source.Host) != "" {
			source += "/" + o.Source.Host
		}
		lines = append(lines,
			"Type:        "+valueOr(o.Type, "Normal"),
			"Reason:      "+valueOr(o.Reason, "<none>"),
			"Involved:    "+valueOr(o.InvolvedObject.Kind, "<none>")+"/"+valueOr(o.InvolvedObject.Name, "<none>"),
			"Source:      "+source,
			"Count:       "+strconv.Itoa(int(o.Count)),
			"Last Seen:   "+tsValue,
			"Message:     "+valueOr(strings.TrimSpace(o.Message), "<none>"),
		)
	}
	if status := strings.TrimSpace(item.Status); status != "" {
		lines = append(lines, "List Status: "+status)
	}
	if ready := strings.TrimSpace(item.Ready); ready != "" {
		lines = append(lines, "List Ready:  "+ready)
	}
	return strings.Join(lines, "\n")
}

func detailFromObject(obj any, resourceName string, item resources.ResourceItem) resources.DetailData {
	switch o := obj.(type) {
	case *corev1.Pod:
		containers := make([]resources.ContainerRow, 0, len(o.Spec.Containers))
		for _, c := range o.Spec.Containers {
			state := "Unknown"
			reason := ""
			for _, cs := range o.Status.ContainerStatuses {
				if cs.Name != c.Name {
					continue
				}
				if cs.State.Running != nil {
					state = "Running"
				} else if cs.State.Waiting != nil {
					state = "Waiting"
					reason = cs.State.Waiting.Reason
				} else if cs.State.Terminated != nil {
					state = "Terminated"
					reason = cs.State.Terminated.Reason
				}
				break
			}
			containers = append(containers, resources.ContainerRow{
				Name:     c.Name,
				Image:    c.Image,
				State:    state,
				Restarts: "0",
				Reason:   reason,
			})
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "status", Label: "Status", Value: valueOr(string(o.Status.Phase), "Unknown")},
				{Key: "ready", Label: "Ready", Value: podReady(*o)},
				{Key: "node", Label: "Node", Value: valueOr(o.Spec.NodeName, "<none>")},
				{Key: "ip", Label: "IP", Value: valueOr(o.Status.PodIP, "<none>")},
				{Key: "qos", Label: "QoS", Value: valueOr(string(o.Status.QOSClass), "Unknown")},
			},
			Containers: containers,
			Labels:     labelsFromMap(o.Labels),
		}
	case *corev1.Service:
		ports := servicePortsString(o.Spec.Ports)
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "status", Label: "Status", Value: "Healthy"},
				{Key: "type", Label: "Type", Value: valueOr(string(o.Spec.Type), "ClusterIP")},
				{Key: "selector", Label: "Selector", Value: valueOr(labelSelectorString(o.Spec.Selector), "<none>")},
				{Key: "cluster_ip", Label: "Cluster IP", Value: valueOr(o.Spec.ClusterIP, "<none>")},
				{Key: "ports", Label: "Ports", Value: valueOr(ports, "<none>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *appsv1.Deployment:
		desired := int32(1)
		if o.Spec.Replicas != nil {
			desired = *o.Spec.Replicas
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Deployment"},
				{Key: "status", Label: "Status", Value: deploymentStatus(*o)},
				{Key: "ready", Label: "Ready", Value: strconv.Itoa(int(o.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired))},
				{Key: "strategy", Label: "Strategy", Value: valueOr(string(o.Spec.Strategy.Type), "<none>")},
				{Key: "selector", Label: "Selector", Value: valueOr(labelSelectorString(o.Spec.Selector.MatchLabels), "<none>")},
				{Key: "images", Label: "Images", Value: valueOr(containerImages(o.Spec.Template.Spec.Containers), "<unknown>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *appsv1.StatefulSet:
		desired := int32(1)
		if o.Spec.Replicas != nil {
			desired = *o.Spec.Replicas
		}
		status := "Healthy"
		if o.Status.ReadyReplicas < desired {
			status = "Progressing"
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "StatefulSet"},
				{Key: "status", Label: "Status", Value: status},
				{Key: "ready", Label: "Ready", Value: strconv.Itoa(int(o.Status.ReadyReplicas)) + "/" + strconv.Itoa(int(desired))},
				{Key: "selector", Label: "Selector", Value: valueOr(labelSelectorString(o.Spec.Selector.MatchLabels), "<none>")},
				{Key: "service", Label: "Service", Value: valueOr(o.Spec.ServiceName, "<none>")},
				{Key: "images", Label: "Images", Value: valueOr(containerImages(o.Spec.Template.Spec.Containers), "<unknown>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *appsv1.DaemonSet:
		status := "Healthy"
		if o.Status.NumberUnavailable > 0 {
			status = "Degraded"
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "DaemonSet"},
				{Key: "status", Label: "Status", Value: status},
				{Key: "ready", Label: "Ready", Value: strconv.Itoa(int(o.Status.NumberReady)) + "/" + strconv.Itoa(int(o.Status.DesiredNumberScheduled))},
				{Key: "selector", Label: "Selector", Value: valueOr(labelSelectorString(o.Spec.Selector.MatchLabels), "<none>")},
				{Key: "images", Label: "Images", Value: valueOr(containerImages(o.Spec.Template.Spec.Containers), "<unknown>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *batchv1.Job:
		completions := int32(1)
		if o.Spec.Completions != nil {
			completions = *o.Spec.Completions
		}
		status := "Running"
		if o.Status.Failed > 0 {
			status = "Failed"
		} else if o.Status.Succeeded >= completions {
			status = "Succeeded"
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Job"},
				{Key: "status", Label: "Status", Value: status},
				{Key: "ready", Label: "Completions", Value: strconv.Itoa(int(o.Status.Succeeded)) + "/" + strconv.Itoa(int(completions))},
				{Key: "parallelism", Label: "Parallelism", Value: strconv.Itoa(int(ptrInt32(o.Spec.Parallelism, 1)))},
				{Key: "images", Label: "Images", Value: valueOr(containerImages(o.Spec.Template.Spec.Containers), "<unknown>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *batchv1.CronJob:
		suspend := "false"
		if o.Spec.Suspend != nil && *o.Spec.Suspend {
			suspend = "true"
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "CronJob"},
				{Key: "status", Label: "Status", Value: "Scheduled"},
				{Key: "schedule", Label: "Schedule", Value: valueOr(o.Spec.Schedule, "<none>")},
				{Key: "suspend", Label: "Suspend", Value: suspend},
				{Key: "images", Label: "Images", Value: valueOr(containerImages(o.Spec.JobTemplate.Spec.Template.Spec.Containers), "<unknown>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *networkingv1.Ingress:
		services := ingressBackendServices(*o)
		servicesValue := "<none>"
		if len(services) > 0 {
			servicesValue = strings.Join(services, ",")
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Ingress"},
				{Key: "status", Label: "Status", Value: "Configured"},
				{Key: "class", Label: "Class", Value: valueOr(ptrString(o.Spec.IngressClassName), "<none>")},
				{Key: "hosts", Label: "Hosts", Value: valueOr(ingressHosts(o.Spec.Rules), "*")},
				{Key: "services", Label: "Services", Value: servicesValue},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *corev1.ConfigMap:
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "ConfigMap"},
				{Key: "status", Label: "Status", Value: "Available"},
				{Key: "keys", Label: "Data Keys", Value: strconv.Itoa(len(o.Data))},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *corev1.Secret:
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Secret"},
				{Key: "status", Label: "Status", Value: "Available"},
				{Key: "type", Label: "Type", Value: valueOr(string(o.Type), "Opaque")},
				{Key: "keys", Label: "Data Keys", Value: strconv.Itoa(len(o.Data))},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *corev1.PersistentVolumeClaim:
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "PersistentVolumeClaim"},
				{Key: "status", Label: "Status", Value: valueOr(string(o.Status.Phase), "Unknown")},
				{Key: "volume", Label: "Volume", Value: valueOr(o.Spec.VolumeName, "<none>")},
				{Key: "storage_class", Label: "StorageClass", Value: valueOr(ptrString(o.Spec.StorageClassName), "<none>")},
			},
			Labels: labelsFromMap(o.Labels),
		}
	case *corev1.Node:
		conditions := formatNodeConditions(o.Status.Conditions)
		addresses := nodeAddressLines(o.Status.Addresses)
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Node"},
				{Key: "status", Label: "Status", Value: valueOr(nodeReadyStatus(*o), "Unknown")},
				{Key: "pod_cidr", Label: "Pod CIDR", Value: valueOr(o.Spec.PodCIDR, "<none>")},
				{Key: "kubelet", Label: "Kubelet", Value: valueOr(o.Status.NodeInfo.KubeletVersion, "<unknown>")},
			},
			Conditions: conditions,
			Events:     addresses,
			Labels:     labelsFromMap(o.Labels),
		}
	case *corev1.Namespace:
		conditions := make([]string, 0, len(o.Spec.Finalizers))
		for _, f := range o.Spec.Finalizers {
			conditions = append(conditions, "Finalizer: "+string(f))
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Namespace"},
				{Key: "status", Label: "Status", Value: valueOr(string(o.Status.Phase), "Unknown")},
			},
			Conditions: conditions,
			Labels:     labelsFromMap(o.Labels),
		}
	case *corev1.Event:
		ts := eventTime(*o)
		tsValue := "<none>"
		if !ts.IsZero() {
			tsValue = ts.UTC().Format(time.RFC3339)
		}
		source := "<none>"
		if strings.TrimSpace(o.Source.Component) != "" {
			source = o.Source.Component
		}
		if strings.TrimSpace(o.Source.Host) != "" {
			source += "/" + o.Source.Host
		}
		return resources.DetailData{
			Summary: []resources.SummaryField{
				{Key: "kind", Label: "Kind", Value: "Event"},
				{Key: "status", Label: "Type", Value: valueOr(o.Type, "Normal")},
				{Key: "reason", Label: "Reason", Value: valueOr(o.Reason, "<none>")},
				{Key: "object", Label: "Object", Value: valueOr(o.InvolvedObject.Name, "<none>")},
				{Key: "source", Label: "Source", Value: source},
				{Key: "count", Label: "Count", Value: strconv.Itoa(int(o.Count))},
				{Key: "last_seen", Label: "Last Seen", Value: tsValue},
			},
			Events: []string{strings.TrimSpace(o.Message)},
		}
	default:
		return genericLiveDetail(item)
	}
}

func ptrInt32(v *int32, fallback int32) int32 {
	if v == nil {
		return fallback
	}
	return *v
}

func servicePortsString(ports []corev1.ServicePort) string {
	if len(ports) == 0 {
		return ""
	}
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		name := strings.TrimSpace(p.Name)
		proto := string(p.Protocol)
		if proto == "" {
			proto = "TCP"
		}
		entry := strconv.Itoa(int(p.Port)) + "/" + proto
		if name != "" {
			entry = name + ":" + entry
		}
		out = append(out, entry)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func ingressBackendRoutes(ing networkingv1.Ingress) []string {
	out := make([]string, 0)
	for _, rule := range ing.Spec.Rules {
		host := strings.TrimSpace(rule.Host)
		if host == "" {
			host = "*"
		}
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			svc := "<none>"
			if path.Backend.Service != nil {
				svc = path.Backend.Service.Name
			}
			p := strings.TrimSpace(path.Path)
			if p == "" {
				p = "/"
			}
			out = append(out, host+" "+p+" -> "+svc)
		}
	}
	sort.Strings(out)
	return out
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func boundedNonEmptyLines(r io.Reader, maxLines int) ([]string, error) {
	if maxLines <= 0 {
		maxLines = 1
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]string, 0, minInt(256, maxLines))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if len(out) < maxLines {
			out = append(out, line)
			continue
		}
		copy(out, out[1:])
		out[len(out)-1] = line
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
