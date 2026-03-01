package resources

type ResourceItem struct {
	Name      string
	Namespace string
	Kind      string
	Status    string
	Ready     string
	Restarts  string
	Age       string
	Labels    map[string]string // pod/resource labels (e.g. {"app": "api", "env": "prod"})
	Selector  map[string]string // label selector for resources that select other resources
	Extra     map[string]string // wide-mode and future fields: "node", "ip", "qos", "selector", etc.
}

// MatchesSelector reports whether labels satisfies selector: every key/value
// pair in selector must appear in labels with the same value.
// An empty or nil selector never matches (explicit selection required).
func MatchesSelector(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
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
	TableRow(item ResourceItem) map[string]string // keyed by column ID
}

// WideResource lets a resource define an expanded column set for wide mode (w key).
type WideResource interface {
	TableColumnsWide() []TableColumn
	TableRowWide(item ResourceItem) map[string]string // keyed by column ID
}

type TableColumn struct {
	ID      string // stable key, e.g. "status", "restarts", "label:app"
	Name    string // display header
	Width   int    // default width hint
	Default bool   // included in default view; false = opt-in only (wide or label columns)
}

// SortKey maps a single keystroke to a sort mode for the sort picker.
// Lowercase char = primary direction; uppercase = reversed.
type SortKey struct {
	Char  rune   // lowercase key (e.g. 'n' for name, 's' for status)
	Mode  string // internal mode name
	Label string // footer display label
}

// Sortable lets a resource expose column-based sort control.
// s enters sort mode in the list view; the next keypress selects a column:
// lowercase = primary/natural direction, uppercase = reversed.
type Sortable interface {
	SetSort(mode string, desc bool)
	SortMode() string
	SortDesc() bool
	SortKeys() []SortKey
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
