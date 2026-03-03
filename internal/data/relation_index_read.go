package data

import (
	"strings"

	"github.com/dloss/podji/internal/resources"
)

type readRelationIndex struct {
	read ReadModel
}

func newReadRelationIndex(read ReadModel) RelationIndex {
	return &readRelationIndex{read: read}
}

func (r *readRelationIndex) Related(scope Scope, resourceName string, item resources.ResourceItem) map[string][]resources.ResourceItem {
	if r.read == nil {
		return map[string][]resources.ResourceItem{}
	}
	name := strings.ToLower(strings.TrimSpace(resourceName))
	out := map[string][]resources.ResourceItem{}

	switch {
	case name == "workloads" || name == "deployments":
		pods := r.list("pods", scope)
		services := r.list("services", scope)
		out["pods"] = relatedPodsForWorkload(item, pods)
		out["services"] = relatedServicesForSelector(item.Selector, services)
		out["config"] = relatedConfigForPods(out["pods"])
		out["storage"] = relatedPVCForPods(out["pods"])
	case strings.HasPrefix(name, "pods"):
		workloads := r.list("workloads", scope)
		services := r.list("services", scope)
		out["owner"] = relatedOwnerForPod(item, workloads)
		out["services"] = relatedServicesForPod(item, services)
		out["config"] = relatedConfigForPods([]resources.ResourceItem{item})
		out["storage"] = relatedPVCForPods([]resources.ResourceItem{item})
	case strings.HasPrefix(name, "services"):
		pods := r.list("pods", scope)
		ingresses := r.list("ingresses", scope)
		out["backends"] = relatedBackendsForService(item, pods)
		out["ingresses"] = relatedIngressesForService(item, ingresses)
	case strings.HasPrefix(name, "ingresses"):
		services := r.list("services", scope)
		out["services"] = relatedServicesForIngress(item, services)
	case name == "nodes":
		pods := r.list("pods", scope)
		out["pods"] = relatedPodsForNode(item, pods)
	case name == "persistentvolumeclaims":
		pods := r.list("pods", scope)
		out["mounted-by"] = relatedPodsForPVC(item, pods)
	default:
		return map[string][]resources.ResourceItem{}
	}
	return out
}

func (r *readRelationIndex) list(resourceName string, scope Scope) []resources.ResourceItem {
	items, err := r.read.List(resourceName, scope)
	if err != nil {
		return nil
	}
	return items
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
	controller := pod.Extra["controlled-by"]
	if controller == "" {
		return nil
	}
	parts := strings.SplitN(controller, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	name := parts[1]
	out := make([]resources.ResourceItem, 0, 1)
	for _, w := range workloads {
		if w.Name == name || strings.HasPrefix(name, w.Name+"-") {
			out = append(out, w)
		}
	}
	return out
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
