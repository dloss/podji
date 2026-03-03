package data

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dloss/podji/internal/resources"
)

type readRelationIndex struct {
	read    ReadModel
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	byScope map[string]relationSnapshot
}

type relationSnapshot struct {
	expiresAt time.Time
	lists     map[string][]resources.ResourceItem
}

func newReadRelationIndex(read ReadModel) RelationIndex {
	return &readRelationIndex{
		read:    read,
		ttl:     2 * time.Second,
		now:     time.Now,
		byScope: map[string]relationSnapshot{},
	}
}

func (r *readRelationIndex) Related(scope Scope, resourceName string, item resources.ResourceItem) map[string][]resources.ResourceItem {
	if r.read == nil {
		return map[string][]resources.ResourceItem{}
	}
	name := strings.ToLower(strings.TrimSpace(resourceName))
	list := func(resource string) []resources.ResourceItem {
		return r.list(resource, name, scope)
	}
	out := map[string][]resources.ResourceItem{}

	switch {
	case name == "workloads" || name == "deployments":
		pods := list("pods")
		services := list("services")
		out["pods"] = relatedPodsForWorkload(item, pods)
		out["services"] = relatedServicesForSelector(item.Selector, services)
		out["config"] = relatedConfigForPods(out["pods"])
		out["storage"] = relatedPVCForPods(out["pods"])
	case strings.HasPrefix(name, "pods"):
		workloads := list("workloads")
		services := list("services")
		out["owner"] = relatedOwnerForPod(item, workloads)
		out["services"] = relatedServicesForPod(item, services)
		out["config"] = relatedConfigForPods([]resources.ResourceItem{item})
		out["storage"] = relatedPVCForPods([]resources.ResourceItem{item})
	case strings.HasPrefix(name, "services"):
		pods := list("pods")
		ingresses := list("ingresses")
		out["backends"] = relatedBackendsForService(item, pods)
		out["ingresses"] = relatedIngressesForService(item, ingresses)
	case strings.HasPrefix(name, "ingresses"):
		services := list("services")
		out["services"] = relatedServicesForIngress(item, services)
	case name == "nodes":
		pods := list("pods")
		out["pods"] = relatedPodsForNode(item, pods)
	case name == "persistentvolumeclaims":
		pods := list("pods")
		out["mounted-by"] = relatedPodsForPVC(item, pods)
	default:
		return map[string][]resources.ResourceItem{}
	}
	return out
}

func (r *readRelationIndex) list(resourceName, sourceResourceName string, scope Scope) []resources.ResourceItem {
	resourceName = strings.ToLower(strings.TrimSpace(resourceName))
	snapshot := r.snapshotFor(sourceResourceName, scope)
	if items, ok := snapshot.lists[resourceName]; ok {
		return items
	}
	items, err := r.read.List(resourceName, scope)
	if err != nil {
		return nil
	}
	return items
}

func (r *readRelationIndex) snapshotFor(sourceResourceName string, scope Scope) relationSnapshot {
	now := r.now()
	scopeKey := relationScopeKey(scope)
	required := relatedListRequirements(sourceResourceName)

	r.mu.Lock()
	current, ok := r.byScope[scopeKey]
	if ok && current.expiresAt.After(now) && snapshotHasLists(current, required) {
		r.mu.Unlock()
		debugRelationf("related source=%s scope=%s snapshot=hit", sourceResourceName, scopeKey)
		return current
	}
	base := relationSnapshot{
		expiresAt: now.Add(r.ttl),
		lists:     map[string][]resources.ResourceItem{},
	}
	if ok && current.expiresAt.After(now) {
		for k, v := range current.lists {
			base.lists[k] = v
		}
	}
	r.mu.Unlock()

	missing := 0
	for _, resourceName := range required {
		if _, exists := base.lists[resourceName]; exists {
			continue
		}
		missing++
		items, err := r.read.List(resourceName, scope)
		if err != nil {
			continue
		}
		base.lists[resourceName] = items
	}

	r.mu.Lock()
	r.byScope[scopeKey] = base
	r.mu.Unlock()
	debugRelationf("related source=%s scope=%s snapshot=refresh missing=%d", sourceResourceName, scopeKey, missing)
	return base
}

func relationScopeKey(scope Scope) string {
	return fmt.Sprintf("%s|%s", strings.TrimSpace(scope.Context), strings.TrimSpace(scope.Namespace))
}

func snapshotHasLists(snapshot relationSnapshot, listNames []string) bool {
	for _, name := range listNames {
		if _, ok := snapshot.lists[name]; !ok {
			return false
		}
	}
	return true
}

func relatedListRequirements(resourceName string) []string {
	name := strings.ToLower(strings.TrimSpace(resourceName))
	switch {
	case name == "workloads" || name == "deployments":
		return []string{"pods", "services"}
	case strings.HasPrefix(name, "pods"):
		return []string{"workloads", "services"}
	case strings.HasPrefix(name, "services"):
		return []string{"pods", "ingresses"}
	case strings.HasPrefix(name, "ingresses"):
		return []string{"services"}
	case name == "nodes":
		return []string{"pods"}
	case name == "persistentvolumeclaims":
		return []string{"pods"}
	default:
		return nil
	}
}

func relatedPodsForWorkload(workload resources.ResourceItem, pods []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, p := range pods {
		if resources.MatchesSelector(workload.Selector, p.Labels) {
			out = append(out, p)
		}
	}
	return out
}

func relatedServicesForSelector(selector map[string]string, services []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, s := range services {
		if selectorsOverlap(selector, s.Selector) {
			out = append(out, s)
		}
	}
	return out
}

func relatedOwnerForPod(pod resources.ResourceItem, workloads []resources.ResourceItem) []resources.ResourceItem {
	controllerUID := strings.TrimSpace(pod.Extra["controlled-by-uid"])
	if controllerUID != "" {
		out := make([]resources.ResourceItem, 0, 1)
		for _, w := range workloads {
			if strings.TrimSpace(w.UID) == controllerUID {
				out = append(out, w)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	controller := pod.Extra["controlled-by"]
	if controller == "" {
		return nil
	}
	parts := strings.SplitN(controller, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	name := parts[1]
	kind := strings.ToLower(strings.TrimSpace(parts[0]))
	out := make([]resources.ResourceItem, 0, 1)
	for _, w := range workloads {
		if !ownerKindMatchesWorkload(kind, w.Kind) {
			continue
		}
		if w.Name == name || strings.HasPrefix(name, w.Name+"-") {
			out = append(out, w)
		}
	}
	return out
}

func ownerKindMatchesWorkload(ownerKind, workloadKind string) bool {
	if ownerKind == "" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(workloadKind)) {
	case "dep":
		return ownerKind == "deployment" || ownerKind == "replicaset"
	case "sts":
		return ownerKind == "statefulset"
	case "ds":
		return ownerKind == "daemonset"
	case "job":
		return ownerKind == "job"
	case "cj":
		return ownerKind == "cronjob"
	default:
		return true
	}
}

func debugRelationf(format string, args ...any) {
	if os.Getenv("PODJI_DEBUG_DATA") != "1" {
		return
	}
	log.Printf("podji:data "+format, args...)
}

func relatedServicesForPod(pod resources.ResourceItem, services []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, s := range services {
		if resources.MatchesSelector(s.Selector, pod.Labels) {
			out = append(out, s)
		}
	}
	return out
}

func relatedBackendsForService(service resources.ResourceItem, pods []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, p := range pods {
		if resources.MatchesSelector(service.Selector, p.Labels) {
			out = append(out, p)
		}
	}
	return out
}

func relatedIngressesForService(service resources.ResourceItem, ingresses []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, ing := range ingresses {
		backendNames := splitCSVNames(ing.Extra["services"])
		if containsName(backendNames, service.Name) || ing.Name == service.Name || strings.Contains(ing.Ready, service.Name) {
			out = append(out, ing)
		}
	}
	return out
}

func relatedServicesForIngress(ingress resources.ResourceItem, services []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	backendNames := splitCSVNames(ingress.Extra["services"])
	for _, s := range services {
		if containsName(backendNames, s.Name) || s.Name == ingress.Name {
			out = append(out, s)
		}
	}
	return out
}

func relatedPodsForNode(node resources.ResourceItem, pods []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, p := range pods {
		if p.Extra["node"] == node.Name {
			out = append(out, p)
		}
	}
	return out
}

func relatedPodsForPVC(pvc resources.ResourceItem, pods []resources.ResourceItem) []resources.ResourceItem {
	out := make([]resources.ResourceItem, 0)
	for _, p := range pods {
		if containsName(splitCSVNames(p.Extra["pvc-refs"]), pvc.Name) {
			out = append(out, p)
		}
	}
	return out
}

func relatedConfigForPods(pods []resources.ResourceItem) []resources.ResourceItem {
	seen := map[string]bool{}
	out := make([]resources.ResourceItem, 0)
	for _, p := range pods {
		for _, n := range splitCSVNames(p.Extra["config-refs"]) {
			if seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, resources.ResourceItem{Name: n, Kind: "ConfigMap"})
		}
		for _, n := range splitCSVNames(p.Extra["secret-refs"]) {
			if seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, resources.ResourceItem{Name: n, Kind: "Secret"})
		}
	}
	return out
}

func relatedPVCForPods(pods []resources.ResourceItem) []resources.ResourceItem {
	seen := map[string]bool{}
	out := make([]resources.ResourceItem, 0)
	for _, p := range pods {
		for _, n := range splitCSVNames(p.Extra["pvc-refs"]) {
			if seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, resources.ResourceItem{Name: n})
		}
	}
	return out
}

func selectorsOverlap(a, b map[string]string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	if resources.MatchesSelector(a, b) || resources.MatchesSelector(b, a) {
		return true
	}
	return false
}

func splitCSVNames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func containsName(items []string, name string) bool {
	for _, item := range items {
		if item == name {
			return true
		}
	}
	return false
}
