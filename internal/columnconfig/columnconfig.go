package columnconfig

import (
	"strings"
	"sync"

	"github.com/dloss/podji/internal/resources"
)

// ColumnConfig stores the user-chosen visible column IDs for one resource type.
type ColumnConfig struct {
	Visible []string // column IDs in display order
}

// Store holds per-resource column visibility configs. Thread-safe.
type Store struct {
	mu      sync.RWMutex
	configs map[string]ColumnConfig // keyed by resource.Name()
}

var defaultStore = &Store{configs: make(map[string]ColumnConfig)}

// Default returns the package-level shared store.
func Default() *Store {
	return defaultStore
}

// Get returns the active column list for a resource. If no config is set,
// returns columns with Default=true from pool, in pool order.
func (s *Store) Get(resourceName string, pool []resources.TableColumn) []resources.TableColumn {
	s.mu.RLock()
	config, exists := s.configs[resourceName]
	s.mu.RUnlock()

	if !exists {
		var defaults []resources.TableColumn
		for _, col := range pool {
			if col.Default {
				defaults = append(defaults, col)
			}
		}
		return defaults
	}

	// Build lookup from pool.
	poolByID := make(map[string]resources.TableColumn, len(pool))
	for _, col := range pool {
		poolByID[col.ID] = col
	}

	var result []resources.TableColumn
	for _, id := range config.Visible {
		if col, ok := poolByID[id]; ok {
			result = append(result, col)
		} else if strings.HasPrefix(id, "label:") {
			// Reconstruct label column not yet in pool (from saved config).
			key := strings.TrimPrefix(id, "label:")
			width := len(key)
			if width < 12 {
				width = 12
			}
			if width > 20 {
				width = 20
			}
			result = append(result, resources.TableColumn{
				ID:      id,
				Name:    strings.ToUpper(key),
				Width:   width,
				Default: false,
			})
		}
	}

	// Always ensure "name" is first; add it if the stored config omitted it.
	if len(result) == 0 || result[0].ID != "name" {
		if _, ok := poolByID["name"]; ok {
			result = append([]resources.TableColumn{poolByID["name"]}, result...)
		}
	}

	return result
}

// Set stores user-chosen visible column IDs for a resource.
// "name" is always prepended (NAME must always be visible).
func (s *Store) Set(resourceName string, visible []string) {
	// Ensure "name" is always first.
	hasName := false
	for _, id := range visible {
		if id == "name" {
			hasName = true
			break
		}
	}
	if !hasName {
		visible = append([]string{"name"}, visible...)
	}

	s.mu.Lock()
	s.configs[resourceName] = ColumnConfig{Visible: visible}
	s.mu.Unlock()
}

// Reset removes user config for resourceName, reverting to defaults.
func (s *Store) Reset(resourceName string) {
	s.mu.Lock()
	delete(s.configs, resourceName)
	s.mu.Unlock()
}

// IsCustom reports whether the user has a non-default config for resourceName.
func (s *Store) IsCustom(resourceName string) bool {
	s.mu.RLock()
	_, exists := s.configs[resourceName]
	s.mu.RUnlock()
	return exists
}
