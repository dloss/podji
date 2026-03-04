package main

import (
	"flag"
	"fmt"
	"os"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/app"
	"github.com/dloss/podji/internal/buildinfo"
)

func main() {
	mockFlag := flag.Bool("mock", false, "run with mock data")
	versionFlag := flag.Bool("version", false, "print version and exit")
	versionShortFlag := flag.Bool("v", false, "print version and exit")
	flag.Parse()

	if *versionFlag || *versionShortFlag {
		_, _ = fmt.Fprintln(os.Stdout, "podji", buildinfo.Full())
		return
	}

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
