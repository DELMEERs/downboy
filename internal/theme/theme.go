package theme

import "github.com/charmbracelet/lipgloss"

// theme color constants
const (
	// primary
	ColorPrimaryBrand = "#FF023D"

	// primary
	ColorPrimaryHoverActive = "#D60032"

	// accent
	ColorAccentSecondary = "#800020"

	// background
	ColorDarkBackground    = "#1A1A1A"
	ColorDarkBackgroundAlt = "#1A0A0E"

	// status & indicator colors
	ColorSuccess = "#00FF87"
	ColorFailure = "#FF023D"
	ColorInfo    = "#888888"
)

// semantic lipgloss color variables
var (
	PrimaryBrand       = lipgloss.Color(ColorPrimaryBrand)
	PrimaryHoverActive = lipgloss.Color(ColorPrimaryHoverActive)
	AccentSecondary    = lipgloss.Color(ColorAccentSecondary)
	DarkBackground     = lipgloss.Color(ColorDarkBackground)
	DarkBackgroundAlt  = lipgloss.Color(ColorDarkBackgroundAlt)
	Success            = lipgloss.Color(ColorSuccess)
	Failure            = lipgloss.Color(ColorFailure)
	Info               = lipgloss.Color(ColorInfo)
)

// semantic UI styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryBrand)

	UpBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Success)

	DownBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Failure)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Info)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Failure)

	SummaryStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryBrand).
			Padding(0, 1).
			MarginTop(1)

	AccentStyle = lipgloss.NewStyle().
			Foreground(AccentSecondary)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Success)

	FailureStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Failure)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Info)
)

// LinearGradientCSS returns the theme CSS linear gradient string
func LinearGradientCSS() string {
	return "linear-gradient(135deg, " + ColorPrimaryBrand + " 0%, " + ColorAccentSecondary + " 100%)"
}
