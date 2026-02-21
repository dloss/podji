package resources

// ContainerResource wraps a pod's containers as a ResourceType + TableResource,
// so that the container picker can reuse listview.View for table rendering.
type ContainerResource struct {
	podItem    ResourceItem
	parentRes  ResourceType
	containers []ContainerRow
}

func NewContainerResource(podItem ResourceItem, parent ResourceType) *ContainerResource {
	return &ContainerResource{
		podItem:    podItem,
		parentRes:  parent,
		containers: parent.Detail(podItem).Containers,
	}
}

func (c *ContainerResource) PodItem() ResourceItem   { return c.podItem }
func (c *ContainerResource) ParentResource() ResourceType { return c.parentRes }

func (c *ContainerResource) Name() string { return "containers" }
func (c *ContainerResource) Key() rune    { return 0 }

func (c *ContainerResource) Items() []ResourceItem {
	items := make([]ResourceItem, 0, len(c.containers))
	for _, cr := range c.containers {
		items = append(items, ResourceItem{
			Name:     cr.Name,
			Status:   cr.State,
			Restarts: cr.Restarts,
		})
	}
	return items
}

func (c *ContainerResource) Sort(_ []ResourceItem) {}

func (c *ContainerResource) Detail(item ResourceItem) DetailData {
	return c.parentRes.Detail(c.podItem)
}

func (c *ContainerResource) Logs(item ResourceItem) []string {
	return c.parentRes.Logs(c.podItem)
}

func (c *ContainerResource) Events(item ResourceItem) []string {
	return c.parentRes.Events(c.podItem)
}

func (c *ContainerResource) YAML(item ResourceItem) string {
	return c.parentRes.YAML(c.podItem)
}

func (c *ContainerResource) TableColumns() []TableColumn {
	return []TableColumn{
		{Name: "NAME", Width: 16},
		{Name: "STATUS", Width: 16},
		{Name: "RESTARTS", Width: 9},
		{Name: "IMAGE", Width: 57},
	}
}

func (c *ContainerResource) TableRow(item ResourceItem) []string {
	for _, cr := range c.containers {
		if cr.Name == item.Name {
			return []string{cr.Name, cr.State, cr.Restarts, cr.Image}
		}
	}
	return []string{item.Name, item.Status, item.Restarts, ""}
}
