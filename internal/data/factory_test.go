package data

import (
	"strings"
	"testing"
)

func TestNewStoreFromEnvDefaultsToMock(t *testing.T) {
	t.Setenv("PODJI_MODE", "")
	store, warning := NewStoreFromEnv()
	if _, ok := store.(*MockStore); !ok {
		t.Fatalf("expected mock store, got %T", store)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
}

func TestNewStoreFromEnvUnknownModeFallsBackToMock(t *testing.T) {
	t.Setenv("PODJI_MODE", "wat")
	store, warning := NewStoreFromEnv()
	if _, ok := store.(*MockStore); !ok {
		t.Fatalf("expected mock store, got %T", store)
	}
	if !strings.Contains(warning, "unknown PODJI_MODE") {
		t.Fatalf("expected unknown-mode warning, got %q", warning)
	}
}
