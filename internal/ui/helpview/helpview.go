package helpview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

var helpText = strings.TrimSpace(`
GLOBAL (app navigation)
  enter / right / l    Selected item
  backspace / left / h Back
  esc                  Clear filter, then back
  N                    Namespace
  X                    Context
  m <1-9>              Set bookmark (context + namespace + resource type)
  1-9                  Jump to bookmark
  0                    Toggle all / last namespace
  A                    All resource types (built-ins + CRDs)
  W                    Workloads
  P                    Pods
  D                    Deployments
  S                    Services
  I                    Ingresses
  C                    ConfigMaps
  K                    Secrets
  V                    PersistentVolumeClaims
  O                    Nodes
  E                    Events (global)

TABLE (filterable lists, including A)
  / (slash)            Search
  &                    Filter
  n / b                Next / previous match
  esc                  Clear filter
  s                    Sort (name/problem)
  w                    Wide columns on/off
  p                    Column visibility picker
  f <char>             Jump to first item by char
  d                    Describe
  y                    YAML
  e                    Events for selected item
  r                    Toggle related panel
  o                    Logs (or next table)
  space / pgup / pgdn  Page up / down
  c                    Copy mode (n name, k kind/name, p -n ns name)
  x                    Execute mode (d delete, r restart, s scale, f port-fwd, x shell)

LOGS (logs view)
  f                    Follow on/off
  w                    Wrap on/off
  p                    Current/previous
  /                    Search
  &                    Filter
  n / b                Next / previous match
  , / .                Cycle since window
  c                    Container picker (from container logs)
  up / down / j / k    Scroll
  space / pgup / pgdn  Page up / down

APP (any view)
  :                    Command bar (from lists)
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
