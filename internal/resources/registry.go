package resources

import (
	"sort"
	"strconv"
	"strings"
)

// AllNamespaces is the sentinel value meaning "show resources from all namespaces".
const AllNamespaces = "(all)"

// ActiveNamespace is the currently selected namespace. Resources can use this
// to vary their stub data so namespace switching is visible.
var ActiveNamespace = "default"

// namespacedColumns prepends a NAMESPACE column when in all-namespaces mode.
func namespacedColumns(cols []TableColumn) []TableColumn {
	if ActiveNamespace != AllNamespaces {
		return cols
	}
	return append([]TableColumn{{ID: "namespace", Name: "NAMESPACE", Width: 16, Default: true}}, cols...)
}

// allNamespaceItems merges stub items from a representative set of namespaces,
// setting the Namespace field on each returned item.
func allNamespaceItems(fn func(string) []ResourceItem) []ResourceItem {
	stubs := []string{"default", "production", "staging", "monitoring"}
	var all []ResourceItem
	for _, ns := range stubs {
		for _, item := range fn(ns) {
			item.Namespace = ns
			all = append(all, item)
		}
	}
	return all
}

type Registry struct {
	resources []ResourceType
	byKey     map[rune]ResourceType
}

func DefaultRegistry() *Registry {
	resources := []ResourceType{
		NewWorkloads(),
		NewPods(),
		NewDeployments(),
		NewServices(),
		NewIngresses(),
		NewConfigMaps(),
		NewSecrets(),
		NewPersistentVolumeClaims(),
		NewNamespaces(),
		NewNodes(),
		NewEvents(),
		NewContexts(),
	}

	byKey := make(map[rune]ResourceType, len(resources))
	for _, res := range resources {
		byKey[res.Key()] = res
	}

	return &Registry{resources: resources, byKey: byKey}
}

func (r *Registry) ResourceByKey(key rune) ResourceType {
	return r.byKey[key]
}

func (r *Registry) Resources() []ResourceType {
	copyList := make([]ResourceType, len(r.resources))
	copy(copyList, r.resources)
	return copyList
}

// ByName returns the ResourceType whose Name() matches name (case-insensitive),
// or nil if not found.
func (r *Registry) ByName(name string) ResourceType {
	name = strings.ToLower(name)
	for _, res := range r.resources {
		if strings.ToLower(res.Name()) == name {
			return res
		}
	}
	return nil
}

func defaultSort(items []ResourceItem) {
	nameSort(items, false)
}

func nameSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		if desc {
			return items[i].Name > items[j].Name
		}
		return items[i].Name < items[j].Name
	})
}

// statusWeight returns a severity weight for sorting: lower = more problematic.
func statusWeight(status string) int {
	switch status {
	case "Failed", "CrashLoop", "CrashLoopBackOff":
		return 0
	case "Degraded", "NotReady", "Warning":
		return 1
	case "Pending", "Progressing", "Unknown":
		return 2
	case "Healthy", "Running", "Ready":
		return 3
	case "Suspended":
		return 4
	default:
		return 5
	}
}

// problemSort sorts items by status severity (most problematic first), then by name.
// Pass desc=true to reverse (healthy-first).
func problemSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		wi := statusWeight(items[i].Status)
		wj := statusWeight(items[j].Status)
		if wi != wj {
			if desc {
				return wi > wj
			}
			return wi < wj
		}
		return items[i].Name < items[j].Name
	})
}

// parseAge converts an age string like "3m", "6h", "2d" to minutes for comparison.
func parseAge(age string) int {
	age = strings.TrimSpace(age)
	if age == "" {
		return 0
	}
	suffix := age[len(age)-1]
	num, err := strconv.Atoi(age[:len(age)-1])
	if err != nil {
		return 0
	}
	switch suffix {
	case 'm':
		return num
	case 'h':
		return num * 60
	case 'd':
		return num * 60 * 24
	default:
		return 0
	}
}

// ageSort sorts items newest first (smallest parsed age), then by name.
// Pass desc=true to reverse (oldest first).
func ageSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		ai := parseAge(items[i].Age)
		aj := parseAge(items[j].Age)
		if ai != aj {
			if desc {
				return ai > aj
			}
			return ai < aj
		}
		return items[i].Name < items[j].Name
	})
}

// kindSort sorts items alphabetically by Kind, then by name.
// Pass desc=true to reverse.
func kindSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			if desc {
				return items[i].Kind > items[j].Kind
			}
			return items[i].Kind < items[j].Kind
		}
		return items[i].Name < items[j].Name
	})
}

// parseReady extracts the numerator from a "X/Y" ready string.
func parseReady(ready string) int {
	if idx := strings.Index(ready, "/"); idx > 0 {
		n, _ := strconv.Atoi(strings.TrimSpace(ready[:idx]))
		return n
	}
	n, _ := strconv.Atoi(strings.TrimSpace(ready))
	return n
}

// readySort sorts by ready count ascending (fewest ready first), then by name.
// Pass desc=true for most-ready-first.
func readySort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		ri := parseReady(items[i].Ready)
		rj := parseReady(items[j].Ready)
		if ri != rj {
			if desc {
				return ri > rj
			}
			return ri < rj
		}
		return items[i].Name < items[j].Name
	})
}

// parseRestarts extracts the restart count from strings like "5" or "5 (10m)".
func parseRestarts(restarts string) int {
	s := strings.TrimSpace(restarts)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(strings.Fields(s)[0])
	return n
}

// restartsSort sorts by restart count ascending (fewest first), then by name.
// Pass desc=true for most-restarts-first.
func restartsSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		ri := parseRestarts(items[i].Restarts)
		rj := parseRestarts(items[j].Restarts)
		if ri != rj {
			if desc {
				return ri > rj
			}
			return ri < rj
		}
		return items[i].Name < items[j].Name
	})
}

// sortKeysFor returns SortKey entries for the given mode names.
// Supported modes: "name" (n), "status" (s), "kind" (k), "age" (a),
// "ready" (r), "restarts" (r, may conflict — resolved by column-char derivation).
func sortKeysFor(modes []string) []SortKey {
	m := map[string]SortKey{
		"name":     {Char: 'n', Mode: "name", Label: "name"},
		"status":   {Char: 's', Mode: "status", Label: "status"},
		"kind":     {Char: 'k', Mode: "kind", Label: "kind"},
		"age":      {Char: 'a', Mode: "age", Label: "age"},
		"ready":    {Char: 'r', Mode: "ready", Label: "ready"},
		"restarts": {Char: 'r', Mode: "restarts", Label: "restarts"},
	}
	keys := make([]SortKey, 0, len(modes))
	for _, mode := range modes {
		if sk, ok := m[mode]; ok {
			keys = append(keys, sk)
		}
	}
	return keys
}