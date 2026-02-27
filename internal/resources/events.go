package resources

import (
	"fmt"
	"strings"
)

type Events struct {
	sortMode string
}

type ScopedEvents struct {
	base   *Events
	object string
	count  int
}

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
		{Name: "NAME", Width: 48},
		{Name: "TYPE", Width: 10},
		{Name: "REASON", Width: 24},
		{Name: "AGE", Width: 6},
	}
}

func (e *Events) TableRow(item ResourceItem) []string {
	parts := strings.SplitN(item.Name, ".", 2)
	object := parts[0]
	reason := ""
	if len(parts) > 1 {
		reason = parts[1]
	}
	return []string{object, item.Kind, reason, item.Age}
}

func NewEvents() *Events {
	return &Events{sortMode: "name"}
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
func (e *ScopedEvents) TableRow(item ResourceItem) []string {
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
func (e *ScopedEvents) Sort(items []ResourceItem) { e.base.Sort(items) }
func (e *ScopedEvents) ToggleSort()               { e.base.ToggleSort() }
func (e *ScopedEvents) SortMode() string          { return e.base.SortMode() }
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
		problemSort(items)
	case "age":
		ageSort(items)
	case "kind":
		kindSort(items)
	default:
		defaultSort(items)
	}
}

func (e *Events) ToggleSort() {
	e.sortMode = cycleSortMode(e.sortMode, []string{"name", "status", "kind", "age"})
}

func (e *Events) SortMode() string {
	return e.sortMode
}

func (e *Events) Detail(item ResourceItem) DetailData {
	parts := strings.SplitN(item.Name, ".", 2)
	object := parts[0]
	reason := ""
	if len(parts) > 1 {
		reason = parts[1]
	}

	message := "Sample event message for " + object
	switch reason {
	case "BackOff":
		message = "Back-off restarting failed container sidecar in pod " + object
	case "OOMKilling":
		message = "Memory capped at 128Mi for container sidecar"
	case "Pulled":
		message = "Successfully pulled image \"envoy:1.28\""
	case "NodeNotReady":
		message = "Node " + object + " status is now: NodeNotReady"
	case "FailedScheduling":
		message = "0/4 nodes are available: 1 node NotReady, 3 Insufficient cpu"
	case "ScalingReplicaSet":
		message = "Scaled up replica set " + object + " to 3"
	case "EnsuredLoadBalancer":
		message = "Load balancer provisioned: a1b2c3d4.elb.amazonaws.com"
	case "SuccessfulCreate":
		message = "Created pod: " + object + "-7m2kq"
	}

	return DetailData{
		StatusLine: item.Kind + "    reason: " + reason + "    object: " + object + "    age: " + item.Age,
		Events: []string{
			item.Age + " ago   " + item.Kind + "   " + reason + "   " + message,
		},
		Labels: []string{
			"involvedObject.name=" + object,
		},
	}
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
		"Namespace:        " + ActiveNamespace + "\n" +
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
  namespace: ` + ActiveNamespace + `
  creationTimestamp: "2026-02-21T09:50:00Z"
involvedObject:
  apiVersion: ` + objAPIVersion + `
  kind: ` + objKind + `
  name: ` + object + `
  namespace: ` + ActiveNamespace + `
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
