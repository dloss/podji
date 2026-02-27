package viewstate

import (
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/dloss/podji/internal/resources"
)

type Action int

const (
	None Action = iota
	Push
	Pop
	Replace
	OpenRelated // signal app.go to open the related side panel
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

// SelectionProvider is implemented by views that have a selected item.
type SelectionProvider interface {
	SelectedItem() resources.ResourceItem
}

// Focusable is implemented by views that can show focused/unfocused visual state.
type Focusable interface {
	SetFocused(bool)
}
