package style

import "github.com/charmbracelet/lipgloss"

var (
	Header = lipgloss.NewStyle().Bold(true)
	Scope  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	Crumb  = lipgloss.NewStyle().Bold(true)
	Active = lipgloss.NewStyle().Bold(true).Reverse(true)
	Footer = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ErrorBanner = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	Error = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	Healthy = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)
