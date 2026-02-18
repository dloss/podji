package resources

type ResourceItem struct {
	Name     string
	Status   string
	Ready    string
	Restarts string
	Age      string
}

type DetailData struct {
	StatusLine string
	Containers []ContainerRow
	Conditions []string
	Events     []string
	Labels     []string
}

type ContainerRow struct {
	Name     string
	Image    string
	State    string
	Restarts string
	Reason   string
}

type ResourceType interface {
	Name() string
	Key() rune
	Items() []ResourceItem
	Sort(items []ResourceItem)
	Detail(item ResourceItem) DetailData
	Logs(item ResourceItem) []string
	Events(item ResourceItem) []string
	YAML(item ResourceItem) string
}
