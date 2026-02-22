package helpview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

var helpText = strings.TrimSpace(`
GLOBAL
  enter / right / l    Selected item
  backspace / left / h Back
  esc                  Clear filter, then back
  home / 0             Lens root
  shift+home           Default lens
  tab / shift+tab      Lens (Apps, Network, Infrastructure)
  N                    Namespace
  X                    Context
  W                    Workloads
  P                    Pods
  D                    Deployments
  S                    Services
  C                    ConfigMaps
  K                    Secrets
  O                    Nodes
  E                    Events (global)

TABLE
  / (slash)            Filter
  esc                  Clear filter
  s                    Sort (name/problem)
  v                    State (workloads)
  f <char>             Jump to first item by char
  d                    Describe
  y                    YAML
  e                    Events for selected item
  r                    Related resources for selected item
  o                    Logs (or next table)
  space / pgup / pgdn  Page up / down

LOGS
  f                    Follow on/off
  w                    Wrap on/off
  t                    Current/previous
  up / down / j / k    Scroll
  space / pgup / pgdn  Page up / down

APP
  ?                    This help
  q / ctrl+c           Quit
`)

type View struct {
	viewport viewport.Model
}

func New() *View {
	vp := viewport.New(0, 0)
	vp.SetContent(helpText)
	return &View{viewport: vp}
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	updated, cmd := v.viewport.Update(msg)
	v.viewport = updated
	return viewstate.Update{Action: viewstate.None, Next: v, Cmd: cmd}
}

func (v *View) View() string {
	return v.viewport.View()
}

func (v *View) Breadcrumb() string {
	return "help"
}

func (v *View) Footer() string {
	line1 := ""
	line2 := style.ActionFooter(nil, v.viewport.Width)
	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
}
