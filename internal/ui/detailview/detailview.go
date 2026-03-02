package detailview

import (
	"fmt"
	"strconv"
	"strings"

	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dloss/podji/internal/resources"
	"github.com/dloss/podji/internal/ui/describeview"
	"github.com/dloss/podji/internal/ui/eventview"
	"github.com/dloss/podji/internal/ui/logview"
	"github.com/dloss/podji/internal/ui/relatedview"
	"github.com/dloss/podji/internal/ui/style"
	"github.com/dloss/podji/internal/ui/viewstate"
	"github.com/dloss/podji/internal/ui/yamlview"
)

type View struct {
	item                 resources.ResourceItem
	resource             resources.ResourceType
	registry             *resources.Registry
	ContainerViewFactory func(item resources.ResourceItem, resource resources.ResourceType) viewstate.View
	width                int
	height               int
}

func New(item resources.ResourceItem, resource resources.ResourceType, registry *resources.Registry) *View {
	return &View{item: item, resource: resource, registry: registry}
}

func (v *View) Init() bubbletea.Cmd { return nil }

func (v *View) Update(msg bubbletea.Msg) viewstate.Update {
	if key, ok := msg.(bubbletea.KeyMsg); ok {
		switch key.String() {
		case "o":
			containers := v.resource.Detail(v.item).Containers
			if len(containers) > 1 && v.ContainerViewFactory != nil {
				return viewstate.Update{Action: viewstate.Push, Next: v.ContainerViewFactory(v.item, v.resource)}
			}
			return viewstate.Update{Action: viewstate.Push, Next: logview.New(v.item, v.resource)}
		case "d":
			return viewstate.Update{Action: viewstate.Push, Next: describeview.New(v.item, v.resource)}
		case "e":
			return viewstate.Update{Action: viewstate.Push, Next: eventview.New(v.item, v.resource)}
		case "y":
			return viewstate.Update{Action: viewstate.Push, Next: yamlview.New(v.item, v.resource)}
		}
	}
	return viewstate.Update{Action: viewstate.None, Next: v}
}

func (v *View) View() string {
	detail := v.resource.Detail(v.item)
	summary := make([]resources.SummaryField, 0, len(detail.Summary)+1)
	summary = append(summary, detail.Summary...)
	if n := relatedview.RelatedCount(v.item, v.resource, v.registry); n > 0 {
		summary = append(summary, resources.SummaryField{
			Key:   "related",
			Label: "Related",
			Value: strconv.Itoa(n),
			Tone:  resources.SummaryToneNeutral,
		})
	}
	sections := []string{}
	if line := renderSummary(summary); line != "" {
		sections = append(sections, line)
	}

	if useTwoColumnLayout(v.width, detail) {
		leftWidth, rightWidth := splitWidths(v.width, 2)
		left := []string{}
		left = append(left, renderContainers(detail.Containers, leftWidth)...)
		left = append(left, titledSection("CONDITIONS", detail.Conditions)...)
		left = append(left, titledSection("LABELS", detail.Labels)...)

		right := []string{}
		right = append(right, titledSection("RECENT EVENTS", detail.Events)...)

		leftCol := lipgloss.NewStyle().Width(leftWidth).Render(strings.Join(left, "\n"))
		rightCol := lipgloss.NewStyle().Width(rightWidth).Render(strings.Join(right, "\n"))
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol))
		return strings.Join(sections, "\n\n")
	}

	sections = append(sections, renderContainers(detail.Containers, v.width)...)
	sections = append(sections, titledSection("CONDITIONS", detail.Conditions)...)
	sections = append(sections, titledSection("RECENT EVENTS", detail.Events)...)
	sections = append(sections, titledSection("LABELS", detail.Labels)...)

	return strings.Join(compactSections(sections), "\n")
}

func (v *View) Breadcrumb() string {
	return v.item.Name
}

func (v *View) Footer() string {
	line1 := ""
	actions := []style.Binding{style.B("o", "logs"), style.B("r", "related")}
	line2 := style.ActionFooter(actions, v.width)
	return line1 + "\n" + line2
}

func (v *View) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func renderContainers(rows []resources.ContainerRow, width int) []string {
	if len(rows) == 0 {
		return nil
	}

	nameW, imageW, stateW, restartW, reasonW := containerColumnWidths(width)
	lines := []string{
		"CONTAINERS",
		fmt.Sprintf(
			"%s %s %s %s %s",
			cell("NAME", nameW),
			cell("IMAGE", imageW),
			cell("STATE", stateW),
			cell("RESTARTS", restartW),
			cell("REASON", reasonW),
		),
	}

	for _, row := range rows {
		lines = append(lines, fmt.Sprintf(
			"%s %s %s %s %s",
			cell(row.Name, nameW),
			cell(row.Image, imageW),
			cell(row.State, stateW),
			cell(row.Restarts, restartW),
			cell(row.Reason, reasonW),
		))
	}

	return lines
}

func titledSection(title string, lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	section := []string{title}
	section = append(section, lines...)
	return section
}

func compactSections(lines []string) []string {
	out := make([]string, 0, len(lines)+8)
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(out) > 0 && (strings.HasPrefix(line, "CONDITIONS") || strings.HasPrefix(line, "RECENT EVENTS") || strings.HasPrefix(line, "LABELS")) {
			out = append(out, "")
		}
		out = append(out, line)
	}
	return out
}

func containerColumnWidths(width int) (int, int, int, int, int) {
	if width < 80 {
		width = 80
	}

	usable := width - 4
	nameW := clamp(usable/8, 10, 18)
	imageW := clamp(usable/4, 18, 36)
	stateW := clamp(usable/6, 12, 22)
	restartW := clamp(usable/10, 8, 12)
	reasonW := usable - (nameW + imageW + stateW + restartW)

	if reasonW < 16 {
		shortfall := 16 - reasonW
		cutImage := min(shortfall, imageW-18)
		imageW -= cutImage
		shortfall -= cutImage

		cutState := min(shortfall, stateW-12)
		stateW -= cutState
		shortfall -= cutState

		cutName := min(shortfall, nameW-10)
		nameW -= cutName
		shortfall -= cutName

		reasonW = usable - (nameW + imageW + stateW + restartW)
	}

	return nameW, imageW, stateW, restartW, reasonW
}

func splitWidths(totalWidth, gap int) (int, int) {
	left := clamp((totalWidth*62)/100, 60, totalWidth-gap-28)
	right := totalWidth - left - gap
	return left, right
}

func useTwoColumnLayout(width int, detail resources.DetailData) bool {
	if width < 120 {
		return false
	}
	// Reserve two-column layout for resources with richer primary detail.
	return len(detail.Containers) > 0 || len(detail.Conditions) > 0
}

func renderSummary(fields []resources.SummaryField) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(field.Value)
		if value == "" {
			continue
		}

		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = strings.TrimSpace(field.Key)
		}
		if label == "" {
			continue
		}

		parts = append(parts, style.Crumb.Render(label+": ")+renderSummaryValue(field, value))
	}
	return strings.Join(parts, "    ")
}

func renderSummaryValue(field resources.SummaryField, value string) string {
	switch field.Tone {
	case resources.SummaryToneGood:
		return style.Healthy.Render(value)
	case resources.SummaryToneWarn:
		return style.Warning.Render(value)
	case resources.SummaryToneBad:
		return style.Error.Render(value)
	case resources.SummaryToneNeutral:
		return style.ScopeValue.Render(value)
	}

	if strings.EqualFold(field.Key, "status") {
		return style.Status(value)
	}
	return value
}

func cell(value string, width int) string {
	runes := []rune(value)
	if len(runes) > width {
		if width <= 1 {
			return "…"
		}
		value = string(runes[:width-1]) + "…"
	}

	padding := width - len([]rune(value))
	if padding > 0 {
		return value + strings.Repeat(" ", padding)
	}
	return value
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
