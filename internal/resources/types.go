package resources

type ResourceItem struct {
	Name     string
	Kind     string
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
	Describe(item ResourceItem) string
}

// TableResource lets a resource define custom table columns and row rendering.
type TableResource interface {
	TableColumns() []TableColumn
	TableRow(item ResourceItem) []string
}

type TableColumn struct {
	Name  string
	Width int
}

// ToggleSortable lets a resource switch sort mode (for example, problem/name).
type ToggleSortable interface {
	ToggleSort()
	SortMode() string
}

// EmptyStateProvider customizes empty-state text for list pages.
type EmptyStateProvider interface {
	EmptyMessage(filtered bool, filter string) string
}

// BannerProvider returns a contextual banner message for list pages.
type BannerProvider interface {
	Banner() string
}

// ScenarioCycler allows mock resources to expose switchable demo states.
type ScenarioCycler interface {
	CycleScenario()
	Scenario() string
}
