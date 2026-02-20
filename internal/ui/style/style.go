package style

import "github.com/charmbracelet/lipgloss"

var (
	Header = lipgloss.NewStyle().Bold(true)
	Scope  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	Crumb  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	Active = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	Footer = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ErrorBanner = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	Error = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	Healthy = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)
