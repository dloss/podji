package resources

import "sort"

type Registry struct {
	resources []ResourceType
	byKey     map[rune]ResourceType
}

func DefaultRegistry() *Registry {
	resources := []ResourceType{
		NewPods(),
		NewDeployments(),
		NewServices(),
		NewConfigMaps(),
		NewSecrets(),
		NewNamespaces(),
		NewNodes(),
		NewEvents(),
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
		statusI := statusWeight(items[i].Status)
		statusJ := statusWeight(items[j].Status)
		if statusI != statusJ {
			return statusI < statusJ
		}
		return items[i].Name < items[j].Name
	})
}

func statusWeight(status string) int {
	switch status {
	case "CrashLoop", "Error", "Failed":
		return 0
	case "Pending", "Warning":
		return 1
	default:
		return 2
	}
}
