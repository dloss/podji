package resources

import "strings"

var singulars = map[string]string{
	"workloads":              "workload",
	"pods":                   "pod",
	"containers":             "container",
	"services":               "service",
	"configmaps":             "configmap",
	"secrets":                "secret",
	"namespaces":             "namespace",
	"nodes":                  "node",
	"events":                 "event",
	"deployments":            "deployment",
	"backends":               "backend",
	"consumers":              "consumer",
	"jobs":                   "job",
	"persistentvolumeclaims": "persistentvolumeclaim",
}

// SingularName returns the singular form of a plural resource name.
// Unknown names are returned unchanged.
func SingularName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if s, ok := singulars[lower]; ok {
		return s
	}
	return name
}
