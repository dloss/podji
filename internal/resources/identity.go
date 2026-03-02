package resources

import "strings"

type ObjectRef struct {
	UID        string
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

func (i ResourceItem) Ref() ObjectRef {
	return ObjectRef{
		UID:        strings.TrimSpace(i.UID),
		APIVersion: strings.TrimSpace(i.APIVersion),
		Kind:       strings.TrimSpace(i.Kind),
		Namespace:  strings.TrimSpace(i.Namespace),
		Name:       strings.TrimSpace(i.Name),
	}
}

// StableKey returns a deterministic identity key for map/index use.
// UID is preferred when available, and falls back to api/kind/ns/name.
func (r ObjectRef) StableKey() string {
	if uid := strings.TrimSpace(r.UID); uid != "" {
		return "uid:" + uid
	}
	return strings.Join([]string{
		strings.TrimSpace(r.APIVersion),
		strings.TrimSpace(r.Kind),
		strings.TrimSpace(r.Namespace),
		strings.TrimSpace(r.Name),
	}, "/")
}
