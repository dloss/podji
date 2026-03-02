package resources

import (
	"fmt"
	"sort"
	"strings"
)

type Events struct {
	namespaceScope
	sortMode string
	sortDesc bool
}

type ScopedEvents struct {
	base   *Events
	object string
	count  int
}

func (e *ScopedEvents) SetNamespace(namespace string) { e.base.SetNamespace(namespace) }
func (e *ScopedEvents) Namespace() string             { return e.base.Namespace() }

func NewScopedEvents(object string, count int) *ScopedEvents {
	if count <= 0 {
		count = 1
	}
	return &ScopedEvents{
		base:   NewEvents(),
		object: strings.TrimSpace(object),
		count:  count,
	}
}

func (e *Events) TableColumns() []TableColumn {
	return []TableColumn{
		{ID: "name", Name: "NAME", Width: 48, Default: true},
		{ID: "type", Name: "TYPE", Width: 10, Default: true},
		{ID: "reason", Name: "REASON", Width: 24, Default: true},
		{ID: "message", Name: "MESSAGE", Width: 44, Default: false},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	}
}

func (e *Events) TableRow(item ResourceItem) map[string]string {
	object, reason := eventObjectAndReason(item)
	return map[string]string{
		"name":    object,
		"type":    item.Kind,
		"reason":  reason,
		"message": eventMessage(object, reason),
		"age":     item.Age,
	}
}

func NewEvents() *Events {
	return &Events{namespaceScope: newNamespaceScope(), sortMode: "name"}
}

func (e *Events) Name() string { return "events" }
func (e *Events) Key() rune    { return 'E' }

func (e *Events) Items() []ResourceItem {
	items := []ResourceItem{
		{Name: "api-7c6c8d5f7d-x8p2k.BackOff", Kind: "Warning", Status: "Warning", Age: "10m"},
		{Name: "api-7c6c8d5f7d-x8p2k.OOMKilling", Kind: "Warning", Status: "Warning", Age: "12m"},
		{Name: "api-7c6c8d5f7d-x8p2k.Pulled", Kind: "Normal", Status: "Healthy", Age: "12m"},
		{Name: "db-0.SuccessfulCreate", Kind: "Normal", Status: "Healthy", Age: "6h"},
		{Name: "worker-04.NodeNotReady", Kind: "Warning", Status: "Warning", Age: "5m"},
		{Name: "payment-service-6f8d9.FailedScheduling", Kind: "Warning", Status: "Warning", Age: "20m"},
		{Name: "search-indexer-7d8f9c.ScalingReplicaSet", Kind: "Normal", Status: "Healthy", Age: "45m"},
		{Name: "ingress-external.EnsuredLoadBalancer", Kind: "Normal", Status: "Healthy", Age: "2d"},
		{Name: "nightly-backup-289173.SuccessfulCreate", Kind: "Normal", Status: "Healthy", Age: "6h"},
	}
	items = expandMockItems(items, 42)
	e.Sort(items)
	return items
}

func (e *ScopedEvents) Name() string { return "events" }
func (e *ScopedEvents) Key() rune    { return 'E' }
func (e *ScopedEvents) TableColumns() []TableColumn {
	return e.base.TableColumns()
}
func (e *ScopedEvents) TableRow(item ResourceItem) map[string]string {
	return e.base.TableRow(item)
}

func (e *ScopedEvents) Items() []ResourceItem {
	object := e.object
	if object == "" {
		object = "object"
	}

	templates := []struct {
		reason string
		kind   string
		status string
		age    string
	}{
		{reason: "BackOff", kind: "Warning", status: "Warning", age: "10m"},
		{reason: "OOMKilling", kind: "Warning", status: "Warning", age: "12m"},
		{reason: "Pulled", kind: "Normal", status: "Healthy", age: "12m"},
		{reason: "FailedScheduling", kind: "Warning", status: "Warning", age: "20m"},
		{reason: "ScalingReplicaSet", kind: "Normal", status: "Healthy", age: "45m"},
		{reason: "SuccessfulCreate", kind: "Normal", status: "Healthy", age: "6h"},
		{reason: "Scheduled", kind: "Normal", status: "Healthy", age: "8m"},
		{reason: "Killing", kind: "Warning", status: "Warning", age: "25m"},
		{reason: "Created", kind: "Normal", status: "Healthy", age: "11m"},
		{reason: "Pulling", kind: "Normal", status: "Healthy", age: "13m"},
		{reason: "Started", kind: "Normal", status: "Healthy", age: "9m"},
		{reason: "Ready", kind: "Normal", status: "Healthy", age: "7m"},
	}

	items := make([]ResourceItem, 0, e.count)
	for i := 0; i < e.count; i++ {
		tmpl := templates[i%len(templates)]
		items = append(items, ResourceItem{
			Name:   fmt.Sprintf("%s.%s", object, tmpl.reason),
			Kind:   tmpl.kind,
			Status: tmpl.status,
			Age:    tmpl.age,
		})
	}

	e.base.Sort(items)
	return items
}
func (e *ScopedEvents) Sort(items []ResourceItem)      { e.base.Sort(items) }
func (e *ScopedEvents) SetSort(mode string, desc bool) { e.base.SetSort(mode, desc) }
func (e *ScopedEvents) SortMode() string               { return e.base.SortMode() }
func (e *ScopedEvents) SortDesc() bool                 { return e.base.SortDesc() }
func (e *ScopedEvents) SortKeys() []SortKey            { return e.base.SortKeys() }
func (e *ScopedEvents) Detail(item ResourceItem) DetailData {
	return e.base.Detail(item)
}
func (e *ScopedEvents) Logs(item ResourceItem) []string {
	return e.base.Logs(item)
}
func (e *ScopedEvents) Events(item ResourceItem) []string {
	return e.base.Events(item)
}
func (e *ScopedEvents) Describe(item ResourceItem) string {
	return e.base.Describe(item)
}
func (e *ScopedEvents) YAML(item ResourceItem) string {
	return e.base.YAML(item)
}

func (e *Events) Sort(items []ResourceItem) {
	switch e.sortMode {
	case "status":
		problemSort(items, e.sortDesc)
	case "age":
		ageSort(items, e.sortDesc)
	case "kind":
		kindSort(items, e.sortDesc)
	case "message":
		eventMessageSort(items, e.sortDesc)
	default:
		nameSort(items, e.sortDesc)
	}
}

func (e *Events) SetSort(mode string, desc bool) { e.sortMode = mode; e.sortDesc = desc }
func (e *Events) SortMode() string               { return e.sortMode }
func (e *Events) SortDesc() bool                 { return e.sortDesc }
func (e *Events) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "kind", "message", "age"})
}

func (e *Events) Detail(item ResourceItem) DetailData {
	object, reason := eventObjectAndReason(item)
	message := eventMessage(object, reason)

	return DetailData{
		Summary: []SummaryField{
			{Key: "status", Label: "Type", Value: item.Kind},
			{Key: "reason", Label: "Reason", Value: reason},
			{Key: "object", Label: "Object", Value: object},
			{Key: "age", Label: "Age", Value: item.Age},
		},
		Events: []string{
			item.Age + " ago   " + item.Kind + "   " + reason + "   " + message,
		},
		Labels: []string{
			"involvedObject.name=" + object,
		},
	}
}

func eventObjectAndReason(item ResourceItem) (string, string) {
	parts := strings.SplitN(item.Name, ".", 2)
	object := parts[0]
	reason := ""
	if len(parts) > 1 {
		reason = parts[1]
	}
	return object, reason
}

func eventMessage(object, reason string) string {
	switch reason {
	case "BackOff":
		return "Back-off restarting failed container sidecar in pod " + object
	case "OOMKilling":
		return "Memory capped at 128Mi for container sidecar"
	case "Pulled":
		return "Successfully pulled image \"envoy:1.28\""
	case "NodeNotReady":
		return "Node " + object + " status is now: NodeNotReady"
	case "FailedScheduling":
		return "0/4 nodes are available: 1 node NotReady, 3 Insufficient cpu"
	case "ScalingReplicaSet":
		return "Scaled up replica set " + object + " to 3"
	case "EnsuredLoadBalancer":
		return "Load balancer provisioned: a1b2c3d4.elb.amazonaws.com"
	case "SuccessfulCreate":
		return "Created pod: " + object + "-7m2kq"
	default:
		return "Sample event message for " + object
	}
}

func eventMessageSort(items []ResourceItem, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		oi, ri := eventObjectAndReason(items[i])
		oj, rj := eventObjectAndReason(items[j])
		mi := eventMessage(oi, ri)
		mj := eventMessage(oj, rj)
		if mi != mj {
			if desc {
				return mi > mj
			}
			return mi < mj
		}
		return items[i].Name < items[j].Name
	})
}

func (e *Events) Logs(item ResourceItem) []string {
	return expandMockLogs([]string{
		"Logs are not available for events.",
	}, 30)
}

func (e *Events) Events(item ResourceItem) []string {
	d := e.Detail(item)
	return d.Events
}

func (e *Events) Describe(item ResourceItem) string {
	parts := strings.SplitN(item.Name, ".", 2)
	object := parts[0]
	reason := ""
	if len(parts) > 1 {
		reason = parts[1]
	}
	d := e.Detail(item)
	message := "Event occurred"
	if len(d.Events) > 0 {
		evParts := strings.SplitN(d.Events[0], "   ", 4)
		if len(evParts) >= 4 {
			message = strings.TrimSpace(evParts[3])
		}
	}
	return "Name:             " + item.Name + "\n" +
		"Namespace:        " + e.Namespace() + "\n" +
		"Involved Object:  " + object + "\n" +
		"Reason:           " + reason + "\n" +
		"Message:          " + message + "\n" +
		"Type:             " + item.Kind + "\n" +
		"Count:            3\n" +
		"Age:              " + item.Age
}

func (e *Events) YAML(item ResourceItem) string {
	parts := strings.SplitN(item.Name, ".", 2)
	object := parts[0]
	reason := ""
	if len(parts) > 1 {
		reason = parts[1]
	}

	d := e.Detail(item)
	message := "Event occurred"
	if len(d.Events) > 0 {
		// Extract message from the last segment of the event line.
		evParts := strings.SplitN(d.Events[0], "   ", 4)
		if len(evParts) >= 4 {
			message = strings.TrimSpace(evParts[3])
		}
	}

	objKind := "Pod"
	objAPIVersion := "v1"
	if strings.Contains(object, "worker-") && !strings.Contains(object, "-") || reason == "NodeNotReady" {
		objKind = "Node"
	}
	if reason == "ScalingReplicaSet" {
		objKind = "Deployment"
		objAPIVersion = "apps/v1"
	}
	if reason == "EnsuredLoadBalancer" {
		objKind = "Service"
	}

	return strings.TrimSpace(`apiVersion: v1
kind: Event
metadata:
  name: ` + item.Name + `.17a3b4c5d6e7f8
  namespace: ` + e.Namespace() + `
  creationTimestamp: "2026-02-21T09:50:00Z"
involvedObject:
  apiVersion: ` + objAPIVersion + `
  kind: ` + objKind + `
  name: ` + object + `
  namespace: ` + e.Namespace() + `
  uid: f1e2d3c4-b5a6-9788-7654-321fedcba098
reason: ` + reason + `
message: ` + message + `
type: ` + item.Kind + `
count: 3
firstTimestamp: "2026-02-21T09:45:00Z"
lastTimestamp: "2026-02-21T09:50:00Z"
source:
  component: kubelet
  host: worker-03
reportingComponent: kubelet
reportingInstance: worker-03`)
}
