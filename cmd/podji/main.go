package main

import (
	"os"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/app"
)

func main() {
	program := bubbletea.NewProgram(app.New())
	if _, err := program.Run(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
