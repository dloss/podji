package data

import (
	"errors"
	"testing"

	"github.com/dloss/podji/internal/resources"
)

func TestNewStoreFromEnvDefaultsToKube(t *testing.T) {
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

	t.Setenv("PODJI_MOCK", "")
	store, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*KubeStore); !ok {
		t.Fatalf("expected kube store, got %T", store)
	}
}

func TestNewStoreForModeUnknownModeReturnsError(t *testing.T) {
	_, err := NewStoreForMode("wat")
	if err == nil {
		t.Fatal("expected unknown-mode error")
	}
}

func TestNewStoreForModeKubeReturnsErrorOnFailure(t *testing.T) {
	prev := newKubeStoreFn
	newKubeStoreFn = func() (*KubeStore, error) {
		return nil, errors.New("kube unavailable")
	}
	t.Cleanup(func() { newKubeStoreFn = prev })

	_, err := NewStoreForMode(ModeKube)
	if err == nil {
		t.Fatal("expected kube init error")
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

	store, err := NewStoreForMode(ModeKube)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*KubeStore); !ok {
		t.Fatalf("expected kube store, got %T", store)
	}
}

func TestNewStoreFromEnvKubeReturnsErrorOnFailure(t *testing.T) {
	prev := newKubeStoreFn
	newKubeStoreFn = func() (*KubeStore, error) {
		return nil, errors.New("kube unavailable")
	}
	t.Cleanup(func() { newKubeStoreFn = prev })

	t.Setenv("PODJI_MOCK", "")
	_, err := NewStoreFromEnv()
	if err == nil {
		t.Fatal("expected kube error")
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

	t.Setenv("PODJI_MOCK", "")
	store, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*KubeStore); !ok {
		t.Fatalf("expected kube store, got %T", store)
	}
}

func TestNewStoreFromEnvMockEnvForcesMock(t *testing.T) {
	t.Setenv("PODJI_MOCK", "1")
	store, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := store.(*MockStore); !ok {
		t.Fatalf("expected mock store, got %T", store)
	}
}
