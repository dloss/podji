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
  N                    Namespace
  X                    Context
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

TABLE
  tab                  Focus related panel (when open)
  / (slash)            Filter
  esc                  Clear filter
  s                    Sort (name/problem)
  v                    State (workloads)
  f <char>             Jump to first item by char
  d                    Describe
  y                    YAML
  e                    Events for selected item
  r                    Toggle related panel
  o                    Logs (or next table)
  space / pgup / pgdn  Page up / down
  c                    Copy mode (n name, k kind/name, p -n ns name)
  x                    Execute mode (d delete, r restart, s scale, f port-fwd, x shell)

RESOURCE BROWSER (A)
  / (slash)            Filter by kind or group
  f <char>             Jump to first resource by char
  enter / right        Open resource list

LOGS
  f                    Follow on/off
  w                    Wrap on/off
  t                    Current/previous
  /                    Search
  n / N                Next / previous match
  [ / ]                Cycle since window
  c                    Container picker (from container logs)
  up / down / j / k    Scroll
  space / pgup / pgdn  Page up / down

BOOKMARKS
  m <1-9>              Set bookmark (context + namespace + resource type)
  1-9                  Jump to bookmark

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
