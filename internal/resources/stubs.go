package resources

import "strings"

type stubResource struct {
	name string
	key  rune
}

func (s *stubResource) Name() string { return s.name }
func (s *stubResource) Key() rune   { return s.key }

func (s *stubResource) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: s.name + "-alpha", Status: "Healthy", Ready: "", Restarts: "", Age: "2d"},
		{Name: s.name + "-beta", Status: "Healthy", Ready: "", Restarts: "", Age: "1d"},
		{Name: s.name + "-gamma", Status: "Warning", Ready: "", Restarts: "", Age: "3h"},
	}
	s.Sort(items)
	return items
}

func (s *stubResource) Sort(items []ResourceItem) {
	defaultSort(items)
}

func (s *stubResource) Detail(item ResourceItem) DetailData {
	return DetailData{
		StatusLine: "Healthy    node: n/a    ip: n/a    qos: n/a",
		Containers: nil,
		Conditions: nil,
		Events: []string{
			"3m ago   Normal   Synced   Sample event for " + item.Name,
		},
		Labels: []string{
			"owner=platform",
		},
	}
}

func (s *stubResource) Logs(item ResourceItem) []string {
	return []string{
		"Logs are not available for this resource type yet.",
	}
}

func (s *stubResource) Events(item ResourceItem) []string {
	return []string{
		"3m ago   Normal   Synced   Sample event for " + item.Name,
	}
}

func (s *stubResource) YAML(item ResourceItem) string {
	kind := s.name
	if len(kind) > 0 {
		kind = strings.ToUpper(kind[:1]) + kind[1:]
	}
	return strings.TrimSpace("kind: " + kind + "\nmetadata:\n  name: " + item.Name)
}

func NewDeployments() ResourceType { return &stubResource{name: "deployments", key: 'D'} }
func NewServices() ResourceType    { return &stubResource{name: "services", key: 'S'} }
func NewConfigMaps() ResourceType  { return &stubResource{name: "configmaps", key: 'C'} }
func NewSecrets() ResourceType     { return &stubResource{name: "secrets", key: 'K'} }
func NewNamespaces() ResourceType  { return &stubResource{name: "namespaces", key: 'N'} }
func NewNodes() ResourceType       { return &stubResource{name: "nodes", key: 'O'} }
func NewEvents() ResourceType      { return &stubResource{name: "events", key: 'E'} }
func NewContexts() ResourceType    { return &stubResource{name: "contexts", key: 'X'} }
