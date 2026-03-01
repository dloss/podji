package describeview

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
)

var (
	sectionStyle   = lipgloss.NewStyle().Bold(true)
	containerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	keyStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	imageStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	resourceStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	probeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type View struct {
	item     resources.ResourceItem
	resource resources.ResourceType
	viewport viewport.Model
}

func New(item resources.ResourceItem, resource resources.ResourceType) *View {
	vp := viewport.New(0, 0)
	vp.SetContent(highlightDescribe(resource.Describe(item)))
	return &View{item: item, resource: resource, viewport: vp}
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
	return "describe"
}

func (v *View) Footer() string {
	line1 := ""
	line2 := style.ActionFooter([]style.Binding{style.B("←", "back")}, v.viewport.Width)
	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	if width == 0 || height == 0 {
		return
	}
	v.viewport.Width = width
	v.viewport.Height = height
}

// highlightDescribe applies color styling to kubectl-style describe output,
// making images, resource limits/requests, and probe config visually prominent.
func highlightDescribe(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inLimits := false
	inRequests := false
	for _, line := range lines {
		out = append(out, highlightLine(line, &inLimits, &inRequests))
	}
	return strings.Join(out, "\n")
}

func highlightLine(line string, inLimits, inRequests *bool) string {
	trimmed := strings.TrimLeft(line, " ")
	if trimmed == "" {
		return line
	}
	indent := len(line) - len(trimmed)

	// Reset limit/request tracking when indent returns to container-field level.
	if indent <= 4 {
		*inLimits = false
		*inRequests = false
	}

	// Find the first ": " to distinguish "key: value" from "key:" (header).
	sepIdx := strings.Index(line, ": ")

	if sepIdx < 0 {
		// No value on this line — section header, container name, or plain row.
		if strings.HasSuffix(trimmed, ":") {
			key := strings.TrimSuffix(trimmed, ":")
			switch indent {
			case 0:
				return sectionStyle.Render(line)
			case 2:
				return containerStyle.Render(line)
			case 4:
				switch key {
				case "Limits":
					*inLimits = true
					return resourceStyle.Render(line)
				case "Requests":
					*inRequests = true
					return resourceStyle.Render(line)
				}
			}
		}
		// Event severity prefix coloring ("  Warning …", "  Normal …").
		if indent == 2 {
			if strings.HasPrefix(trimmed, "Warning") {
				return style.Warning.Render(line)
			}
		}
		return line
	}

	// Split "    Key:   value" → keyPart ("    Key:"), afterColon ("   value").
	keyPart := line[:sepIdx+1]
	afterColon := line[sepIdx+1:]
	key := strings.TrimSpace(keyPart)

	valStart := 0
	for valStart < len(afterColon) && afterColon[valStart] == ' ' {
		valStart++
	}
	valueSep := afterColon[:valStart]
	value := afterColon[valStart:]

	switch indent {
	case 0:
		// Top-level fields: Name, Namespace, Status, IP, …
		styledKey := keyStyle.Render(keyPart)
		if key == "Status" {
			return styledKey + valueSep + style.Status(value)
		}
		return styledKey + valueSep + value

	case 4:
		// Container fields.
		styledKey := keyStyle.Render(keyPart)
		switch key {
		case "Image":
			return styledKey + valueSep + imageStyle.Render(value)
		case "State":
			return styledKey + valueSep + style.Status(value)
		case "Liveness", "Readiness", "Startup":
			return styledKey + valueSep + probeStyle.Render(value)
		case "Restart Count":
			n := parseRestarts(value)
			if n > 10 {
				return styledKey + valueSep + style.Error.Render(value)
			} else if n > 0 {
				return styledKey + valueSep + style.Warning.Render(value)
			}
		}
		return styledKey + valueSep + value

	case 6:
		// Nested values: resource quantities or state sub-fields (Reason, Started).
		styledKey := keyStyle.Render(keyPart)
		if *inLimits || *inRequests {
			if key == "memory" {
				return styledKey + valueSep + resourceStyle.Render(value)
			}
			return styledKey + valueSep + value
		}
		// State sub-fields.
		if key == "Reason" {
			return styledKey + valueSep + style.Error.Render(value)
		}
		return styledKey + valueSep + value
	}

	return keyStyle.Render(keyPart) + valueSep + value
}

// parseRestarts returns the integer restart count from strings like "5" or "5 (10m ago)".
func parseRestarts(s string) int {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(fields[0])
	return n
}
