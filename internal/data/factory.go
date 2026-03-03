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

func NewStoreForMode(mode string) (Store, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", ModeMock:
		return NewMockStore(), nil
	case ModeKube:
		store, err := newKubeStoreFn()
		if err == nil {
			return store, nil
		}
		return nil, fmt.Errorf("kube mode unavailable: %w", err)
	default:
		return nil, fmt.Errorf("unknown PODJI_MODE=%q", mode)
	}
}

// NewStoreFromEnv returns a store based on PODJI_MODE.
// Supported values: "mock" (default), "kube".
func NewStoreFromEnv() (Store, error) {
	return NewStoreForMode(os.Getenv("PODJI_MODE"))
}
