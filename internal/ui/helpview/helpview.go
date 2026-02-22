package helpview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

var helpText = strings.TrimSpace(`
NAVIGATION
  enter / right / l    Open selected item
  backspace / left / h Back / pop view
  esc                  Clear filter, then back
  home                 Go to lens root
  shift+home           Go to default lens

VIEWS
  tab / shift+tab      Cycle lens (Apps, Network, Infrastructure)
  N                    Switch namespace
  X                    Switch context
  W                    Workloads
  P                    Pods
  D                    Deployments
  S                    Services
  O                    Nodes

LIST VIEWS
  / (slash)            Start filter
  esc                  Clear filter
  s                    Toggle sort (name/problem)
  v                    Cycle scenario (workloads)
  d                    Describe
  e                    Events
  y                    YAML
  f <char>             Jump to first item starting with char
  R                    Related resources
  L                    Logs (direct)
  pgup / pgdn          Page up / down

DETAIL VIEW
  d                    Describe
  L                    Logs
  e                    Events
  y                    YAML
  R                    Related

LOG VIEW
  f                    Toggle follow
  w                    Toggle wrap
  t                    Toggle current/previous
  up / down / j / k    Scroll

GENERAL
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
