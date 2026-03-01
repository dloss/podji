package resources

import (
	"fmt"
	"strings"
)

type PersistentVolumeClaims struct {
	sortMode string
	sortDesc bool
}

func NewPersistentVolumeClaims() *PersistentVolumeClaims {
	return &PersistentVolumeClaims{sortMode: "name"}
}

func (p *PersistentVolumeClaims) Name() string { return "persistentvolumeclaims" }
func (p *PersistentVolumeClaims) Key() rune    { return 'V' }

func (p *PersistentVolumeClaims) TableColumns() []TableColumn {
	return namespacedColumns([]TableColumn{
		{ID: "name", Name: "NAME", Width: 32, Default: true},
		{ID: "capacity", Name: "CAPACITY", Width: 10, Default: true},
		{ID: "access-mode", Name: "ACCESS MODE", Width: 13, Default: true},
		{ID: "status", Name: "STATUS", Width: 10, Default: true},
		{ID: "storage-class", Name: "STORAGECLASS", Width: 14, Default: true},
		{ID: "age", Name: "AGE", Width: 6, Default: true},
	})
}

func (p *PersistentVolumeClaims) TableRow(item ResourceItem) map[string]string {
	// Kind holds the access mode, Ready holds the capacity.
	return map[string]string{
		"namespace":     item.Namespace,
		"name":          item.Name,
		"capacity":      item.Ready,
		"access-mode":   item.Kind,
		"status":        item.Status,
		"storage-class": pvcStorageClass(item.Name),
		"age":           item.Age,
	}
}

func pvcStorageClass(name string) string {
	switch name {
	case "prometheus-data", "loki-data":
		return "standard"
	default:
		return "gp3"
	}
}

func pvcVolumeName(name, status string) string {
	if status != "Bound" {
		return "<unbound>"
	}
	var h byte
	for i := 0; i < len(name); i++ {
		h = h*31 + name[i]
	}
	return fmt.Sprintf("pvc-%08x-0000-4000-0000-%012x", uint32(h)*0x01010101, uint64(h)*0x010101010101)
}

func pvcMountedBy(name, status string) string {
	if status != "Bound" {
		return "<none>"
	}
	switch name {
	case "postgres-data":
		return "db-0"
	case "redis-data":
		return "cache-0"
	case "grafana-data":
		return "grafana-59d8f9b4c6-7xkpz"
	case "elasticsearch-data-0":
		return "elasticsearch-0"
	case "prometheus-data":
		return "prometheus-0"
	case "loki-data":
		return "loki-0"
	default:
		return name[:min(len(name), 8)] + "-pod"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *PersistentVolumeClaims) Items() []ResourceItem {
	var items []ResourceItem
	if ActiveNamespace == AllNamespaces {
		items = allNamespaceItems(pvcItemsForNamespace)
	} else {
		items = pvcItemsForNamespace(ActiveNamespace)
		items = expandMockItems(items, 20)
	}
	p.Sort(items)
	return items
}

func pvcItemsForNamespace(ns string) []ResourceItem {
	switch ns {
	case "production":
		return []ResourceItem{
			{Name: "elasticsearch-data-0", Status: "Bound", Ready: "100Gi", Kind: "RWO", Age: "60d"},
			{Name: "nightly-backup-data", Status: "Pending", Ready: "10Gi", Kind: "RWO", Age: "2m"},
			{Name: "postgres-data", Status: "Bound", Ready: "50Gi", Kind: "RWO", Age: "45d"},
			{Name: "redis-data", Status: "Bound", Ready: "10Gi", Kind: "RWO", Age: "45d"},
		}
	case "kube-system":
		return []ResourceItem{
			{Name: "loki-data", Status: "Bound", Ready: "20Gi", Kind: "RWO", Age: "90d"},
			{Name: "prometheus-data", Status: "Bound", Ready: "50Gi", Kind: "RWO", Age: "30d"},
		}
	default:
		return []ResourceItem{
			{Name: "grafana-data", Status: "Bound", Ready: "2Gi", Kind: "RWO", Age: "30d"},
			{Name: "postgres-data", Status: "Bound", Ready: "20Gi", Kind: "RWO", Age: "45d"},
			{Name: "redis-data", Status: "Bound", Ready: "5Gi", Kind: "RWO", Age: "45d"},
		}
	}
}

func (p *PersistentVolumeClaims) Sort(items []ResourceItem) {
	switch p.sortMode {
	case "status":
		problemSort(items, p.sortDesc)
	case "age":
		ageSort(items, p.sortDesc)
	default:
		nameSort(items, p.sortDesc)
	}
}

func (p *PersistentVolumeClaims) SetSort(mode string, desc bool) { p.sortMode = mode; p.sortDesc = desc }
func (p *PersistentVolumeClaims) SortMode() string               { return p.sortMode }
func (p *PersistentVolumeClaims) SortDesc() bool                 { return p.sortDesc }
func (p *PersistentVolumeClaims) SortKeys() []SortKey {
	return sortKeysFor([]string{"name", "status", "age"})
}

func (p *PersistentVolumeClaims) Detail(item ResourceItem) DetailData {
	statusLine := item.Status + "    capacity: " + item.Ready + "    access: " + item.Kind + "    class: " + pvcStorageClass(item.Name)
	events := []string{"—   No recent events"}
	if item.Status == "Pending" {
		events = []string{
			"2m ago   Warning   ProvisioningFailed   waiting for a volume to be created, either due to unresolvable PVC or StorageClass",
		}
	}
	conditions := []string{
		"Used By:  " + pvcMountedBy(item.Name, item.Status),
		"Volume:   " + pvcVolumeName(item.Name, item.Status),
	}
	return DetailData{
		StatusLine: statusLine,
		Conditions: conditions,
		Events:     events,
		Labels:     []string{"app.kubernetes.io/managed-by=helm"},
	}
}

func (p *PersistentVolumeClaims) Logs(item ResourceItem) []string {
	return expandMockLogs([]string{"Logs are not available for persistentvolumeclaims."}, 30)
}

func (p *PersistentVolumeClaims) Events(item ResourceItem) []string {
	if item.Status == "Pending" {
		return []string{
			"2m ago   Warning   ProvisioningFailed   waiting for a volume to be created, either due to unresolvable PVC or StorageClass",
		}
	}
	return []string{"—   No recent events"}
}

func (p *PersistentVolumeClaims) Describe(item ResourceItem) string {
	volume := pvcVolumeName(item.Name, item.Status)
	mountedBy := pvcMountedBy(item.Name, item.Status)
	sc := pvcStorageClass(item.Name)

	events := "Events:  <none>"
	if item.Status == "Pending" {
		events = "Events:\n" +
			"  Type     Reason              Age  Message\n" +
			"  ----     ------              ---  -------\n" +
			"  Warning  ProvisioningFailed  2m   waiting for a volume to be created"
	}

	return "Name:          " + item.Name + "\n" +
		"Namespace:     " + ActiveNamespace + "\n" +
		"StorageClass:  " + sc + "\n" +
		"Status:        " + item.Status + "\n" +
		"Volume:        " + volume + "\n" +
		"Labels:        app.kubernetes.io/managed-by=helm\n" +
		"Finalizers:    [kubernetes.io/pvc-protection]\n" +
		"Capacity:      " + item.Ready + "\n" +
		"Access Modes:  " + item.Kind + "\n" +
		"VolumeMode:    Filesystem\n" +
		"Used By:       " + mountedBy + "\n" +
		events
}

func (p *PersistentVolumeClaims) YAML(item ResourceItem) string {
	volume := pvcVolumeName(item.Name, item.Status)
	sc := pvcStorageClass(item.Name)
	phase := item.Status

	yaml := strings.TrimSpace(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ` + item.Name + `
  namespace: ` + ActiveNamespace + `
  labels:
    app.kubernetes.io/managed-by: helm
  finalizers:
  - kubernetes.io/pvc-protection
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: ` + item.Ready + `
  storageClassName: ` + sc + `
  volumeMode: Filesystem`)

	if phase == "Bound" {
		yaml += "\n  volumeName: " + volume
	}

	yaml += `
status:
  phase: ` + phase
	if phase == "Bound" {
		yaml += `
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: ` + item.Ready
	}
	return yaml
}
