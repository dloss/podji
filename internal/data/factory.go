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

// NewStoreFromEnv returns a store based on PODJI_MODE.
// Supported values: "mock" (default), "kube".
// When kube mode initialization fails, this falls back to mock and returns a warning.
func NewStoreFromEnv() (Store, string) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PODJI_MODE")))
	switch mode {
	case "", ModeMock:
		return NewMockStore(), ""
	case ModeKube:
		store, err := NewKubeStore()
		if err == nil {
			return store, ""
		}
		return NewMockStore(), fmt.Sprintf("kube mode unavailable: %v (using mock mode)", err)
	default:
		return NewMockStore(), fmt.Sprintf("unknown PODJI_MODE=%q (using mock mode)", mode)
	}
}
