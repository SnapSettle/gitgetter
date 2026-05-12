// Package styles holds all Lipgloss styles used across the TUI.
// Palette: Tokyo Night (https://github.com/enkia/tokyo-night-vscode-theme)
package styles

import "github.com/charmbracelet/lipgloss"

// ── Palette ──────────────────────────────────────────────────────────────────

var (
	ColorBg      = lipgloss.Color("#1a1b26")
	ColorBgAlt   = lipgloss.Color("#16161e")
	ColorBgSel   = lipgloss.Color("#283457")
	ColorBorder  = lipgloss.Color("#292e42")
	ColorComment = lipgloss.Color("#565f89")
	ColorText    = lipgloss.Color("#c0caf5")
	ColorSubtext = lipgloss.Color("#a9b1d6")
	AccentBlue   = lipgloss.Color("#7aa2f7")
	AccentPurple = lipgloss.Color("#bb9af7")
	AccentCyan   = lipgloss.Color("#7dcfff")
	AccentGreen  = lipgloss.Color("#9ece6a")
	AccentYellow = lipgloss.Color("#e0af68")
	AccentRed    = lipgloss.Color("#f7768e")
	AccentOrange = lipgloss.Color("#ff9e64")
)

// ── Base ──────────────────────────────────────────────────────────────────────

var (
	Base  = lipgloss.NewStyle().Foreground(ColorText)
	Muted = lipgloss.NewStyle().Foreground(ColorComment)
	Bold  = lipgloss.NewStyle().Bold(true).Foreground(ColorText)
)

// ── Title bar ─────────────────────────────────────────────────────────────────

var (
	TitleLogo = lipgloss.NewStyle().
			Background(AccentPurple).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 2)

	TitleRight = lipgloss.NewStyle().
			Background(ColorBgAlt).
			Foreground(ColorSubtext).
			Padding(0, 2)

	TitleBar = lipgloss.NewStyle().
			Background(ColorBgAlt).
			Width(0)
)

// ── Tabs ──────────────────────────────────────────────────────────────────────

var (
	Tab = lipgloss.NewStyle().
		Foreground(ColorComment).
		Padding(0, 2)

	ActiveTab = lipgloss.NewStyle().
			Foreground(AccentBlue).
			Bold(true).
			Padding(0, 2)

	TabDivider = lipgloss.NewStyle().
			Foreground(ColorBorder)
)

// ── Panels ────────────────────────────────────────────────────────────────────

var (
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	FocusedPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentBlue).
			Padding(1, 2)
)

// ── List items ────────────────────────────────────────────────────────────────

var (
	ItemNormal = lipgloss.NewStyle().Foreground(ColorSubtext)

	ItemSelected = lipgloss.NewStyle().
			Background(ColorBgSel).
			Foreground(ColorText).
			Bold(true)

	ItemCursor = lipgloss.NewStyle().Foreground(AccentBlue).Bold(true)

	ItemChecked = lipgloss.NewStyle().Foreground(AccentGreen)
)

// ── Status icons ──────────────────────────────────────────────────────────────

var (
	IconStaged    = lipgloss.NewStyle().Foreground(AccentGreen).Bold(true)
	IconUnstaged  = lipgloss.NewStyle().Foreground(AccentYellow).Bold(true)
	IconUntracked = lipgloss.NewStyle().Foreground(ColorComment)
	IconDeleted   = lipgloss.NewStyle().Foreground(AccentRed).Bold(true)
	IconRenamed   = lipgloss.NewStyle().Foreground(AccentCyan).Bold(true)
)

// ── Git-specific ──────────────────────────────────────────────────────────────

var (
	BranchCurrent = lipgloss.NewStyle().Foreground(AccentGreen).Bold(true)
	BranchLocal   = lipgloss.NewStyle().Foreground(AccentCyan)
	BranchRemote  = lipgloss.NewStyle().Foreground(ColorComment)

	CommitHash = lipgloss.NewStyle().Foreground(AccentPurple)
	CommitMsg  = lipgloss.NewStyle().Foreground(ColorText)
	CommitMeta = lipgloss.NewStyle().Foreground(ColorComment)

	RefTag    = lipgloss.NewStyle().Background(AccentOrange).Foreground(ColorBg).Padding(0, 1)
	RefBranch = lipgloss.NewStyle().Background(AccentBlue).Foreground(ColorBg).Padding(0, 1)
	RefHead   = lipgloss.NewStyle().Background(AccentCyan).Foreground(ColorBg).Bold(true).Padding(0, 1)
)

// ── Pills / badges ────────────────────────────────────────────────────────────

var (
	PillGreen  = lipgloss.NewStyle().Background(AccentGreen).Foreground(ColorBg).Bold(true).Padding(0, 1)
	PillYellow = lipgloss.NewStyle().Background(AccentYellow).Foreground(ColorBg).Bold(true).Padding(0, 1)
	PillRed    = lipgloss.NewStyle().Background(AccentRed).Foreground(ColorText).Bold(true).Padding(0, 1)
	PillBlue   = lipgloss.NewStyle().Background(AccentBlue).Foreground(ColorBg).Bold(true).Padding(0, 1)
	PillGray   = lipgloss.NewStyle().Background(ColorBorder).Foreground(ColorSubtext).Padding(0, 1)
)

// ── Notifications ─────────────────────────────────────────────────────────────

var (
	NotifSuccess = lipgloss.NewStyle().
			Background(AccentGreen).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)

	NotifError = lipgloss.NewStyle().
			Background(AccentRed).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)

	NotifInfo = lipgloss.NewStyle().
			Background(AccentBlue).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)
)

// ── Section titles ────────────────────────────────────────────────────────────

var SectionTitle = lipgloss.NewStyle().Foreground(AccentCyan).Bold(true)

// ── Help bar ──────────────────────────────────────────────────────────────────

var (
	HelpBar = lipgloss.NewStyle().
		Background(ColorBgAlt).
		Foreground(ColorComment).
		Padding(0, 1)

	HelpKey  = lipgloss.NewStyle().Foreground(AccentYellow).Bold(true)
	HelpDesc = lipgloss.NewStyle().Foreground(ColorComment)
)

// ── Input ─────────────────────────────────────────────────────────────────────

var (
	InputLabel = lipgloss.NewStyle().Foreground(AccentCyan).Bold(true)
	InputHint  = lipgloss.NewStyle().Foreground(ColorComment)
)

// ── Confirm dialog ────────────────────────────────────────────────────────────

var (
	ConfirmBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentRed).
			Padding(1, 3)

	ConfirmTitle = lipgloss.NewStyle().Foreground(AccentRed).Bold(true)
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// Truncate clips s to max runes, adding "…" if clipped.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
