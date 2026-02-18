package viewstate

import bubbletea "github.com/charmbracelet/bubbletea"

type Action int

const (
	None Action = iota
	Push
	Pop
	Replace
)

type Update struct {
	Action Action
	Next   View
	Cmd    bubbletea.Cmd
}

type View interface {
	Init() bubbletea.Cmd
	Update(msg bubbletea.Msg) Update
	View() string
	Breadcrumb() string
	Footer() string
	SetSize(width, height int)
}
