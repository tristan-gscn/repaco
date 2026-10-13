package tui

import "github.com/charmbracelet/lipgloss"

var (
	// JetBrains Dark Theme Palette
	JBBackground  = lipgloss.Color("#1E1F22")
	JBHeaderBg    = lipgloss.Color("#2B2D30")
	JBBorder      = lipgloss.Color("#393B40")
	JBSelection   = lipgloss.Color("#2E436E")
	JBTextPrimary = lipgloss.Color("#DFE1E5")
	JBTextMuted   = lipgloss.Color("#868A91")
	JBAccent      = lipgloss.Color("#3574F0") // JetBrains Blue
	JBGreen       = lipgloss.Color("#59A869") // JetBrains Green
	JBRed         = lipgloss.Color("#DB5860") // JetBrains Red
	JBAmber       = lipgloss.Color("#E5A038") // JetBrains Warning
	JBPurple      = lipgloss.Color("#A970FF")
	JBCyan        = lipgloss.Color("#499C54")

	// Header Styles
	HeaderStyle = lipgloss.NewStyle().
			Background(JBHeaderBg).
			Foreground(JBTextPrimary).
			Bold(true).
			Padding(0, 1)

	ProjectTagStyle = lipgloss.NewStyle().
			Background(JBAccent).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)

	// Status Badges
	BadgePass = lipgloss.NewStyle().
			Foreground(JBGreen).
			Bold(true)

	BadgeFail = lipgloss.NewStyle().
			Foreground(JBRed).
			Bold(true)

	BadgeCrash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4D4D")).
			Bold(true)

	BadgeWarning = lipgloss.NewStyle().
			Foreground(JBAmber).
			Bold(true)

	BadgeLeak = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4AA8FF")).
			Bold(true)

	// Panel Styles (zero padding so selection stretches edge-to-edge)
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(JBBorder)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(JBAccent)

	// List Items (border to border highlight)
	SelectedRowStyle = lipgloss.NewStyle().
				Background(JBSelection).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	NormalRowStyle = lipgloss.NewStyle().
			Foreground(JBTextPrimary)

	MutedTextStyle = lipgloss.NewStyle().
			Foreground(JBTextMuted)

	// Detail Sections
	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(JBTextPrimary).
				Bold(true)

	SectionLabelStyle = lipgloss.NewStyle().
				Foreground(JBAccent).
				Bold(true)

	DiffExpectedStyle = lipgloss.NewStyle().
				Foreground(JBGreen)

	DiffActualStyle = lipgloss.NewStyle().
			Foreground(JBRed)

	AdviceStyle = lipgloss.NewStyle().
			Foreground(JBGreen).
			Bold(true)

	CodeSnippetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C8D3F5")).
			Background(lipgloss.Color("#222436")).
			Padding(0, 1)

	// Footer Styles
	FooterBarStyle = lipgloss.NewStyle().
			Background(JBHeaderBg).
			Foreground(JBTextPrimary).
			Padding(0, 1)

	FooterKeyStyle = lipgloss.NewStyle().
			Foreground(JBAccent).
			Bold(true)

	FooterDescStyle = lipgloss.NewStyle().
			Foreground(JBTextMuted)
)
