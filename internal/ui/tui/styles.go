package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Palette
	PrimaryColor   = lipgloss.Color("#7C3AED") // Purple
	SecondaryColor = lipgloss.Color("#6366F1") // Indigo
	SuccessColor   = lipgloss.Color("#10B981") // Green
	ErrorColor     = lipgloss.Color("#EF4444") // Red
	CrashColor     = lipgloss.Color("#DC2626") // Dark Red
	WarningColor   = lipgloss.Color("#F59E0B") // Amber
	InfoColor      = lipgloss.Color("#3B82F6") // Blue
	MutedColor     = lipgloss.Color("#6B7280") // Gray
	DarkBgColor    = lipgloss.Color("#1F2937") // Dark Slate
	BorderColor    = lipgloss.Color("#374151") // Border Gray

	// Header Styles
	HeaderTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(PrimaryColor).
				Bold(true).
				Padding(0, 1)

	HeaderSubStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")).
			Bold(false).
			Padding(0, 1)

	BadgePass = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(SuccessColor).
			Bold(true).
			Padding(0, 1)

	BadgeFail = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ErrorColor).
			Bold(true).
			Padding(0, 1)

	BadgeCrash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(CrashColor).
			Bold(true).
			Padding(0, 1)

	// List Pane Styles
	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#374151")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	NormalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	MutedTextStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Panel Borders
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(SecondaryColor).
				Padding(0, 1)

	// Detail Pane Styles
	DetailHeaderStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor).
				Bold(true).
				MarginBottom(1)

	SectionLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Bold(true)

	DiffExpectedStyle = lipgloss.NewStyle().
				Foreground(SuccessColor)

	DiffActualStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	AdviceStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	// Footer Styles
	FooterKeyStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	FooterDescStyle = lipgloss.NewStyle().
			Foreground(MutedColor)
)
