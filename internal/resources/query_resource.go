package resources

// QueryResource is a synthetic list backed by a fixed item set.
type QueryResource struct {
	name  string
	items []ResourceItem
	base  ResourceType
}

func NewQueryResource(name string, items []ResourceItem, base ResourceType) *QueryResource {
	copyItems := make([]ResourceItem, len(items))
	copy(copyItems, items)
	return &QueryResource{name: name, items: copyItems, base: base}
}

func (r *QueryResource) Name() string                        { return r.name }
func (r *QueryResource) Key() rune                           { return 0 }
func (r *QueryResource) Items() []ResourceItem               { return r.items }
func (r *QueryResource) Sort(items []ResourceItem)           { defaultSort(items) }
func (r *QueryResource) Detail(item ResourceItem) DetailData { return r.base.Detail(item) }
func (r *QueryResource) Logs(item ResourceItem) []string     { return r.base.Logs(item) }
func (r *QueryResource) Events(item ResourceItem) []string   { return r.base.Events(item) }
func (r *QueryResource) YAML(item ResourceItem) string       { return r.base.YAML(item) }
func (r *QueryResource) Describe(item ResourceItem) string   { return r.base.Describe(item) }

func (r *QueryResource) TableColumns() []TableColumn {
	if tr, ok := r.base.(TableResource); ok {
		return tr.TableColumns()
	}
	return nil
}

func (r *QueryResource) TableRow(item ResourceItem) map[string]string {
	if tr, ok := r.base.(TableResource); ok {
		return tr.TableRow(item)
	}
	return map[string]string{"name": item.Name, "status": item.Status, "age": item.Age}
}

func (r *QueryResource) EmptyMessage(filtered bool, filter string) string {
	if filtered {
		return "No matches."
	}
	return "No items."
}
