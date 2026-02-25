package resources

import (
	"sort"
	"strconv"
	"strings"
)

// ActiveNamespace is the currently selected namespace. Resources can use this
// to vary their stub data so namespace switching is visible.
var ActiveNamespace = "default"

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

func defaultSort(items []ResourceItem) {
	sort.SliceStable(items, func(i, j int) bool {
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
func problemSort(items []ResourceItem) {
	sort.SliceStable(items, func(i, j int) bool {
		wi := statusWeight(items[i].Status)
		wj := statusWeight(items[j].Status)
		if wi != wj {
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
func ageSort(items []ResourceItem) {
	sort.SliceStable(items, func(i, j int) bool {
		ai := parseAge(items[i].Age)
		aj := parseAge(items[j].Age)
		if ai != aj {
			return ai < aj
		}
		return items[i].Name < items[j].Name
	})
}

// kindSort sorts items alphabetically by Kind, then by name.
func kindSort(items []ResourceItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].Name < items[j].Name
	})
}

// cycleSortMode advances to the next mode in the given slice.
func cycleSortMode(current string, modes []string) string {
	for idx, m := range modes {
		if m == current {
			return modes[(idx+1)%len(modes)]
		}
	}
	return modes[0]
}