package data

import (
	"fmt"
	"os"
	"strings"
)

const (
	ModeMock = "mock"
	ModeKube = "kube"
)

var newKubeStoreFn = NewKubeStore

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

func NewStoreForMode(mode string) (Store, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case ModeMock:
		return NewMockStore(), nil
	case "", ModeKube:
		store, err := newKubeStoreFn()
		if err == nil {
			return store, nil
		}
		return nil, fmt.Errorf("kube mode unavailable: %w", err)
	default:
		return nil, fmt.Errorf("unknown mode=%q", mode)
	}
}

// NewStoreFromEnv returns a store based on PODJI_MOCK.
// kube is the default; PODJI_MOCK truthy values force mock mode.
func NewStoreFromEnv() (Store, error) {
	if isTruthy(os.Getenv("PODJI_MOCK")) {
		return NewStoreForMode(ModeMock)
	}
	return NewStoreForMode(ModeKube)
}
