package main

import (
	"flag"
	"os"
	"strings"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/app"
)

func main() {
	modeFlag := flag.String("mode", "", "store mode: mock or kube")
	flag.Parse()
	if mode := strings.TrimSpace(*modeFlag); mode != "" {
		_ = os.Setenv("PODJI_MODE", mode)
	}

	program := bubbletea.NewProgram(app.New(), bubbletea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
