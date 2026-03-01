package resources

import (
	"sort"
	"strconv"
	"strings"
)

// UnhealthyItems returns non-healthy resources across selected types.
func UnhealthyItems() []ResourceItem {
	types := []ResourceType{NewPods(), NewDeployments(), NewPersistentVolumeClaims()}
	var out []ResourceItem
	for _, res := range types {
		for _, item := range res.Items() {
			if isUnhealthy(item) {
				item.Kind = strings.ToUpper(SingularName(res.Name()))
				out = append(out, item)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		si := unhealthySeverity(out[i])
		sj := unhealthySeverity(out[j])
		if si != sj {
			return si < sj
		}
		ai := parseAge(out[i].Age)
		aj := parseAge(out[j].Age)
		if ai != aj {
			return ai < aj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// PodsByRestarts returns all pods with restart count > 0 sorted desc.
func PodsByRestarts() []ResourceItem {
	pods := NewPods().Items()
	out := make([]ResourceItem, 0, len(pods))
	for _, item := range pods {
		if parseRestartCount(item.Restarts) > 0 {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri := parseRestartCount(out[i].Restarts)
		rj := parseRestartCount(out[j].Restarts)
		if ri != rj {
			return ri > rj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func parseRestartCount(raw string) int {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return n
}

func unhealthySeverity(item ResourceItem) int {
	if strings.Contains(strings.ToLower(item.Status), "fail") || strings.Contains(strings.ToLower(item.Status), "crash") {
		return 0
	}
	if strings.Contains(strings.ToLower(item.Status), "degrad") {
		return 1
	}
	return 2
}

func isUnhealthy(item ResourceItem) bool {
	status := strings.ToLower(item.Status)
	if status == "" || status == "healthy" || status == "running" || status == "bound" {
		return false
	}
	return true
}

// MatchesLabelSelector reports whether item labels satisfy key=value pairs.
func MatchesLabelSelector(item ResourceItem, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return false
	}
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return false
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" || item.Labels[k] != v {
			return false
		}
	}
	return true
}
