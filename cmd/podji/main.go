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
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		_, _ = fmt.Fprintf(out, "Podji - Kubernetes navigation TUI\n\n")
		_, _ = fmt.Fprintf(out, "Usage:\n")
		_, _ = fmt.Fprintf(out, "  %s [flags]\n\n", os.Args[0])
		_, _ = fmt.Fprintf(out, "Flags:\n")
		flag.PrintDefaults()
		_, _ = fmt.Fprintf(out, "\nEnvironment:\n")
		_, _ = fmt.Fprintf(out, "  PODJI_MOCK=1                  Force mock mode\n")
		_, _ = fmt.Fprintf(out, "  PODJI_MOCK_SCENARIO=<name>    Mock scenario (normal|empty|forbidden|partial|offline|stress)\n")
		_, _ = fmt.Fprintf(out, "  PODJI_SCENARIO=<name>         Legacy fallback for scenario\n")
		_, _ = fmt.Fprintf(out, "  PODJI_STRESS=1                Enable synthetic stress expansion in mock mode\n")
		_, _ = fmt.Fprintf(out, "  PODJI_DEBUG_DATA=1            Log startup/data timing debug lines\n")
	}

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
