package data

import (
	"errors"
	"strings"
	"testing"

	"github.com/dloss/podji/internal/resources"
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

func TestNewStoreForModeKubeFallbackOnError(t *testing.T) {
	prev := newKubeStoreFn
	newKubeStoreFn = func() (*KubeStore, error) {
		return nil, errors.New("kube unavailable")
	}
	t.Cleanup(func() { newKubeStoreFn = prev })

	store, warning := NewStoreForMode(ModeKube)
	if _, ok := store.(*MockStore); !ok {
		t.Fatalf("expected mock store fallback, got %T", store)
	}
	if !strings.Contains(warning, "kube mode unavailable") {
		t.Fatalf("expected kube fallback warning, got %q", warning)
	}
}

func TestNewStoreForModeKubeSuccess(t *testing.T) {
	prev := newKubeStoreFn
	newKubeStoreFn = func() (*KubeStore, error) {
		reg := resources.DefaultRegistry()
		reg.SetNamespace(resources.DefaultNamespace)
		return &KubeStore{
			registry:  reg,
			read:      NewMockReadModel(reg),
			relations: newMockRelationIndex(reg),
			scope: Scope{
				Context:   "dev",
				Namespace: resources.DefaultNamespace,
			},
			api: fakeKubeAPI{
				contexts: []string{"dev"},
			},
		}, nil
	}
	t.Cleanup(func() { newKubeStoreFn = prev })

	store, warning := NewStoreForMode(ModeKube)
	if _, ok := store.(*KubeStore); !ok {
		t.Fatalf("expected kube store, got %T", store)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
}

func TestNewStoreFromEnvKubeFallbackOnError(t *testing.T) {
	prev := newKubeStoreFn
	newKubeStoreFn = func() (*KubeStore, error) {
		return nil, errors.New("kube unavailable")
	}
	t.Cleanup(func() { newKubeStoreFn = prev })

	t.Setenv("PODJI_MODE", "kube")
	store, warning := NewStoreFromEnv()
	if _, ok := store.(*MockStore); !ok {
		t.Fatalf("expected mock store fallback, got %T", store)
	}
	if !strings.Contains(warning, "kube mode unavailable") {
		t.Fatalf("expected kube fallback warning, got %q", warning)
	}
}

func TestNewStoreFromEnvKubeSuccess(t *testing.T) {
	prev := newKubeStoreFn
	newKubeStoreFn = func() (*KubeStore, error) {
		reg := resources.DefaultRegistry()
		reg.SetNamespace(resources.DefaultNamespace)
		return &KubeStore{
			registry:  reg,
			read:      NewMockReadModel(reg),
			relations: newMockRelationIndex(reg),
			scope: Scope{
				Context:   "dev",
				Namespace: resources.DefaultNamespace,
			},
			api: fakeKubeAPI{
				contexts: []string{"dev"},
			},
		}, nil
	}
	t.Cleanup(func() { newKubeStoreFn = prev })

	t.Setenv("PODJI_MODE", "kube")
	store, warning := NewStoreFromEnv()
	if _, ok := store.(*KubeStore); !ok {
		t.Fatalf("expected kube store, got %T", store)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
}
