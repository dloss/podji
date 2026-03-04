package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Header           = lipgloss.NewStyle().Bold(true)
	Separator        = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	Scope            = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	ScopeValue       = lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	ScopeActive      = lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	ScopeActiveValue = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	ContextProd      = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	Crumb            = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	CrumbValue       = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	CrumbSep         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	Active           = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	NavSep           = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	Footer           = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	FooterKey        = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	FooterLabel      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	Muted            = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	Warning          = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	Error            = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	Healthy          = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	FilterPrompt     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func SeparatorLine(width int) string {
	if width <= 0 {
		return Separator.Render("─")
	}
	return Separator.Render(strings.Repeat("─", width))
}

type statusSeverity int

const (
	statusHealthy statusSeverity = iota
	statusNeutral
	statusWarning
	statusError
)

func Status(value string) string {
	switch classifyStatus(value) {
	case statusError:
		return Error.Render(value)
	case statusWarning:
		return Warning.Render(value)
	case statusNeutral:
		return Muted.Render(value)
	default:
		return Healthy.Render(value)
	}
}

func classifyStatus(value string) statusSeverity {
	normalized := strings.ToLower(strings.TrimSpace(value))

	switch {
	case strings.Contains(normalized, "crashloop"),
		strings.Contains(normalized, "error"),
		strings.Contains(normalized, "fail"),
		strings.Contains(normalized, "oom"),
		strings.Contains(normalized, "backoff"):
		return statusError
	case strings.Contains(normalized, "pending"),
		strings.Contains(normalized, "warning"),
		strings.Contains(normalized, "degraded"),
		strings.Contains(normalized, "progress"),
		strings.Contains(normalized, "terminat"),
		strings.Contains(normalized, "unknown"):
		return statusWarning
	case strings.Contains(normalized, "suspend"):
		return statusNeutral
	default:
		return statusHealthy
	}
}
