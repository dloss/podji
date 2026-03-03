package main

import (
	"flag"
	"os"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/app"
)

func main() {
	mockFlag := flag.Bool("mock", false, "run with mock data")
	flag.Parse()
	if *mockFlag {
		_ = os.Setenv("PODJI_MOCK", "1")
	}

	model, err := app.NewFromEnv()
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	program := bubbletea.NewProgram(model, bubbletea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
